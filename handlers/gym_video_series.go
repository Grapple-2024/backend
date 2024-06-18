package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type GymVideoSeriesHandler struct {
	*dynamodbsdk.Client
	*AuthService
	*s3.PresignClient ``
	videoSeriesTable  string
	videosTable       string
}

type GymVideoSeries struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID       string     `json:"gym_id,omitempty" dynamodbav:"gym_id,omitempty"`
	Title       string     `validator:"nonzero" json:"title,omitempty" dynamodbav:"title,omitempty"`
	Description string     `json:"description,omitempty" dynamodbav:"description,omitempty"`
	Difficulty  string     `validator:"nonzero" json:"difficulty,omitempty" dynamodbav:"difficulty,omitempty"`
	Disciplines []string   `json:"disciplines,omitempty" dynamodbav:"disciplines,stringsets,omitempty"`
	Videos      []GymVideo `json:"videos,omitempty" dynamodbav:"videos,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at"`

	// used for sorting by updated_at timestamp, workaround for dynamodb
	Dummy string `json:"-" dynamodbav:"dummy,omitempty"`
}

func NewGymVideoSeriesHandler(ctx context.Context, dynamoEndpoint string) (*GymVideoSeriesHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)

	return &GymVideoSeriesHandler{
		Client:           db,
		AuthService:      authSVC,
		PresignClient:    psc,
		videoSeriesTable: os.Getenv("GYM_VIDEO_SERIES_TABLE_NAME"),
		videosTable:      os.Getenv("GYM_VIDEOS_TABLE_NAME"),
	}, nil
}

func (h *GymVideoSeriesHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	title := req.QueryStringParameters["title"]
	disciplines := req.MultiValueQueryStringParameters["discipline"]
	difficulties := req.MultiValueQueryStringParameters["difficulty"]
	ascending := parseBool(req.QueryStringParameters["ascending"], true)
	gym := req.QueryStringParameters["gym"]
	if gym == "" {
		return lambda.ClientError(http.StatusBadRequest, "must specify ?gym query parameter")
	}

	builder := expression.NewBuilder().WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))
	filter := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"gym_id": {
			Value:    gym,
			Operator: "Equal",
		},
		"difficulty": {
			Value:    difficulties,
			Operator: "StringIn", // find gym videos with a difficulty that matches one of the difficulties in the difficulties slice.
		},
		"disciplines": {
			Value:    disciplines,
			Operator: "ContainsOr", // find gym videos that have a discipline associated with any of the disciplines in the disciplines slice.
		},
		"title": {
			Value:    title,
			Operator: "Contains",
		},
	})

	if filter != nil {
		builder = builder.WithFilter(*filter)
	}

	expr, err := builder.Build()
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to build expression: %v", err))
	}

	// temporary workaround to ensure number of results are in the page
	scanLimit := limit + 1000
	result, err := h.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &h.videoSeriesTable,
		Limit:                     &scanLimit,
		ScanIndexForward:          &ascending,
		IndexName:                 aws.String("LastUpdatedIndex"),
		ExclusiveStartKey:         startKey,
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query dynamodb: %v", err))
	}

	var videoSeries []GymVideoSeries
	resp, err := dynamodbsdk.MarshalResponse(
		aws.String("updated_at"), limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &videoSeries,
	)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("error marshalling response: %v", err))
	}

	// Get gym videos for each series
	for i, s := range videoSeries {
		builder = expression.NewBuilder().WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))
		filter = dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
			"series_id": {
				Value:    s.PK,
				Operator: "Equal",
			},
		})
		builder = builder.WithFilter(*filter)
		expr, err = builder.Build()
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to build expression: %v", err))
		}

		v, err := h.Query(ctx, &dynamodb.QueryInput{
			TableName:                 &h.videosTable,
			Limit:                     &scanLimit,
			ScanIndexForward:          &ascending,
			IndexName:                 aws.String("LastUpdatedIndex"),
			KeyConditionExpression:    expr.KeyCondition(),
			FilterExpression:          expr.Filter(),
			ExpressionAttributeNames:  expr.Names(),
			ExpressionAttributeValues: expr.Values(),
		})
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to query dynamodb: %v", err))
		}

		log.Info().Msgf("Videos: %+v", v.Items)

		videos := []GymVideo{}
		if err := attributevalue.UnmarshalListOfMaps(v.Items, &videos); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal gym videos: %v", err))
		}

		videoSeries[i].Videos = videos

		for _, v := range videos {
			key := fmt.Sprintf("%s/%s", s.GymID, v.S3Object)
			presignedURL, err := h.getPresignedURL(key)
			if err != nil {
				return lambda.ServerError(fmt.Errorf("failed to create presigned url: %v", err))
			}
			v.PresignedURL = presignedURL.Value
		}
	}

	// Marshal resp back
	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoSeriesHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	result, err := h.GetByID(ctx, h.videoSeriesTable, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	// if len(result.Items) > 0 {
	// 	gymID := result.Items[0]["gym_id"].(*types.AttributeValueMemberS).Value

	// 	for _, v := range result.Items[0]["videos"].(*types.AttributeValueMemberL).Value {
	// 		s3Key := v.(*types.AttributeValueMemberM).Value["s3_object"].(*types.AttributeValueMemberS).Value
	// 		key := fmt.Sprintf("%s/%s", gymID, s3Key)
	// 		url, err := h.getPresignedURL(key)
	// 		if err != nil {
	// 			return lambda.ServerError(fmt.Errorf("failed to get presigned url: %v", err))
	// 		}

	// 		log.Info().Msgf("Fetched presigned S3 url for video: %v", url)
	// 		v.(*types.AttributeValueMemberM).Value["url"] = url
	// 	}

	// }

	var requests []GymVideoSeries
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(requests[0])
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoSeriesHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymVideo GymVideoSeries
	if err := json.Unmarshal([]byte(req.Body), &gymVideo); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}
	if err := validate.Struct(&gymVideo); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("request body failed validation: %v", err))
	}

	gymVideo.CreatedAt = time.Now().UTC()
	gymVideo.UpdatedAt = gymVideo.CreatedAt
	gymVideo.Dummy = "dumb"
	gymVideo.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymVideo#%s/%s/%d", gymVideo.GymID, gymVideo.Title, gymVideo.CreatedAt.Unix())),
	)

	res, err := h.Insert(ctx, h.videoSeriesTable, &gymVideo)
	if err != nil {
		return lambda.ServerError(err)
	}

	var returnGym GymVideoSeries
	err = attributevalue.UnmarshalMap(res.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&gymVideo)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymVideoSeriesHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	// Fetch the Gym Request
	result, err := h.GetByID(ctx, h.videoSeriesTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym request not found")
	}

	var videos []GymVideoSeries
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &videos); err != nil {
		return lambda.ServerError(err)
	}

	log.Printf("Received DELETE request with id = %s", id)

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Delete(ctx, h.videoSeriesTable, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}
func (h *GymVideoSeriesHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	var payload GymVideoSeries
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	// Marshal
	av, _ := attributevalue.MarshalMap(payload)
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "gym_id" || k == "created_at" || k == "updated_at" {
			continue
		}

		log.Info().Msgf("Updating field %v to %v", k, v)
		update = update.Set(expression.Name(k), expression.Value(v))
	}

	update = update.Set(expression.Name("updated_at"), expression.Value(time.Now().UTC()))
	log.Info().Msgf("Update query: %+v", update)
	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	).WithUpdate(update)

	expr, err := builder.Build()
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request payload")
	}

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Update(ctx, h.videoSeriesTable, key, &expr)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var gymVideo GymVideoSeries
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gymVideo); err != nil {
		return lambda.ServerError(err)
	}

	log.Info().Msgf("Gym request: %v", gymVideo)
	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoSeriesHandler) getPresignedURL(key string) (*types.AttributeValueMemberS, error) {
	// Get pre-signed URL
	params := &s3.GetObjectInput{
		Bucket: aws.String("grapple-gym-videos"),
		Key:    aws.String(key),
	}

	r, err := h.PresignClient.PresignGetObject(context.TODO(), params, func(opts *s3.PresignOptions) {
		opts.Expires = time.Minute * 30
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get a presigned download url for object %v: %w", key, err)
	}

	return &types.AttributeValueMemberS{
		Value: r.URL,
	}, nil
}

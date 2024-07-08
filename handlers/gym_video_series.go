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

var (
	videoSeriesTableName = os.Getenv("GYM_VIDEO_SERIES_TABLE_NAME")
	videosTableName      = os.Getenv("GYM_VIDEOS_TABLE_NAME")
)

type DynamoQueryer interface {
	Query(ctx context.Context, params *dynamodb.QueryInput, optFns ...func(*dynamodb.Options)) (*dynamodb.QueryOutput, error)
}

type GymVideoSeriesHandler struct {
	*dynamodbsdk.Client
	*AuthService
	*s3.PresignClient
}

type GymVideoSeries struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID       string     `json:"gym_id,omitempty" dynamodbav:"gym_id,omitempty"`
	Title       string     `validator:"nonzero" json:"title,omitempty" dynamodbav:"title,omitempty"`
	Description string     `json:"description,omitempty" dynamodbav:"description,omitempty"`
	Videos      []GymVideo `json:"videos,omitempty" dynamodbav:"videos,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at"`

	// These fields are auto-computed at runtime when new videos are added/removed to a GymVideoSeries

	Difficulties []string `json:"difficulties" dynamodbav:"difficulties,stringsets,omitempty"`
	Disciplines  []string `json:"disciplines" dynamodbav:"disciplines,stringsets,omitempty"`

	// Dummy field is used for sorting by updated_at timestamp; workaround for dynamodb
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

	return &GymVideoSeriesHandler{
		Client:        db,
		AuthService:   authSVC,
		PresignClient: s3.NewPresignClient(s3.NewFromConfig(cfg)),
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

	resp, err := fetchGymVideoSeries(h, ctx, gym, title, difficulties, disciplines, ascending, limit, startKey)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to fetch gym video series: %v", err))
	}

	data := resp.Data.(*[]GymVideoSeries)
	// Fetch all underlying videos for each VideoSeries in the slice (JOIN the dynamo tables)
	for i, s := range *data {
		if err := fetchVideosForSeries(ctx, h, h.PresignClient, &(*data)[i], ascending); err != nil {
			return lambda.ServerError(fmt.Errorf("error fetching videos for series %q: %v", s.PK, err))
		}
	}

	// Marshal to JSON and return the HTTP response
	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoSeriesHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	result, err := h.GetByID(ctx, videoSeriesTableName, id)
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
	var series GymVideoSeries
	if err := json.Unmarshal([]byte(req.Body), &series); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}
	if err := validate.Struct(&series); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("request body failed validation: %v", err))
	}

	series.CreatedAt = time.Now().UTC()
	series.UpdatedAt = series.CreatedAt
	series.Dummy = "dumb"
	series.Disciplines = []string{}
	series.Difficulties = []string{}
	series.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymVideoSeries#%s/%s/%d", series.GymID, series.Title, series.CreatedAt.Unix())),
	)

	_, err := h.Insert(ctx, videoSeriesTableName, &series)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&series)
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

	// Fetch the Gym Series from DB before deleting it
	result, err := h.GetByID(ctx, videoSeriesTableName, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "video series not found")
	}

	var series []GymVideoSeries
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &series); err != nil {
		return lambda.ServerError(err)
	}

	// Cascade Deletion of underlying Video objects in the Series.
	for _, v := range series[0].Videos {
		pk, err := attributevalue.Marshal(v.PK)
		if err != nil {
			return lambda.ServerError(err)
		}
		key := map[string]types.AttributeValue{
			"pk": pk,
		}

		_, err = h.Delete(ctx, videosTableName, key)
		if err != nil {
			return lambda.ServerError(err)
		}
	}

	// Delete the series
	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Delete(ctx, videoSeriesTableName, key)
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

	resp, err := h.Update(ctx, videoSeriesTableName, key, &expr)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var series GymVideoSeries
	if err := attributevalue.UnmarshalMap(resp.Attributes, &series); err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func getPresignedURL(ctx context.Context, client *s3.PresignClient, key string) (*types.AttributeValueMemberS, error) {
	// Get pre-signed URL
	params := &s3.GetObjectInput{
		Bucket: aws.String("grapple-gym-videos"),
		Key:    aws.String(key),
	}

	r, err := client.PresignGetObject(ctx, params, func(opts *s3.PresignOptions) {
		opts.Expires = time.Minute * 30
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get a presigned download url for object %v: %w", key, err)
	}

	return &types.AttributeValueMemberS{
		Value: r.URL,
	}, nil
}

func fetchVideosForSeries(ctx context.Context, h DynamoQueryer, presignClient *s3.PresignClient, series *GymVideoSeries, ascending bool) error {
	// Get videos for each series
	log.Info().Msgf("Fetching videos for series %v", series.PK)

	builder := expression.NewBuilder().WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))
	filter := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"series_id": {
			Value:    series.PK,
			Operator: "Equal",
		},
	})
	builder = builder.WithFilter(*filter)
	expr, err := builder.Build()
	if err != nil {
		return fmt.Errorf("failed to build expression: %v", err)
	}

	var limit int32 = 10000
	v, err := h.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &videosTableName,
		Limit:                     &limit,
		ScanIndexForward:          &ascending,
		IndexName:                 aws.String("LastUpdatedIndex"),
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
	})
	if err != nil {
		return fmt.Errorf("failed to query dynamodb: %v", err)
	}

	videos := []GymVideo{}
	if err := attributevalue.UnmarshalListOfMaps(v.Items, &videos); err != nil {
		return fmt.Errorf("failed to unmarshal gym videos: %v", err)
	}

	// generat presigned urls for each video in the series
	for i, v := range videos {
		key := fmt.Sprintf("%s/%s", series.GymID, v.S3Object)
		presignedURL, err := getPresignedURL(ctx, presignClient, key)
		if err != nil {
			return fmt.Errorf("failed to create presigned url: %v", err)
		}
		videos[i].PresignedURL = presignedURL.Value
	}
	(*series).Videos = videos

	return nil
}

func fetchGymVideoSeries(h DynamoQueryer, ctx context.Context, gym, title string, difficulties, disciplines []string, ascending bool, limit int32, startKey map[string]types.AttributeValue) (*dynamodbsdk.GetResponse, error) {
	builder := expression.NewBuilder().WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))
	filter := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"gym_id": {
			Value:    gym,
			Operator: "Equal",
		},
		"title": {
			Value:    title,
			Operator: "Contains",
		},
		"difficulties": {
			Value:    difficulties,
			Operator: "ContainsOr", // find gym video series with a difficulty that matches one of the difficulties in the difficulties slice.
		},
		"disciplines": {
			Value:    disciplines,
			Operator: "ContainsOr", // find gym video series that have a discipline associated with any of the disciplines in the disciplines slice.
		},
	})

	if filter != nil {
		builder = builder.WithFilter(*filter)
	}

	expr, err := builder.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build expression: %w", err)
	}

	// Query dynamodb database
	scanLimit := int32(limit + 1000)
	result, err := h.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &videoSeriesTableName,
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
		return nil, fmt.Errorf("failed to query dynamodb: %v", err)
	}

	series := []GymVideoSeries{}
	return dynamodbsdk.MarshalResponse(aws.String("updated_at"), limit, result.Count, result.ScannedCount,
		result.LastEvaluatedKey, result.Items, &series)
}

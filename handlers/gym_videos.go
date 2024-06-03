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

type GymVideoHandler struct {
	*dynamodbsdk.Client
	*AuthService
	*s3.PresignClient
	videosTable string
}

type GymVideo struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID   string `json:"gym_id,omitempty" dynamodbav:"gym_id,omitempty"`
	Title   string `validator:"nonzero" json:"title,omitempty" dynamodbav:"title,omitempty"`
	Content string `json:"content,omitempty" dynamodbav:"content,omitempty"`

	Difficulty  string    `validator:"nonzero" json:"difficulty,omitempty" dynamodbav:"difficulty,omitempty"`
	Disciplines []string  `json:"disciplines,omitempty" dynamodbav:"disciplines,stringsets,omitempty"`
	S3Object    string    `json:"s3_object,omitempty" dynamodbav:"s3_object,omitempty"`
	URL         string    `json:"url,omitempty" dynamodbav:"url,omitempty"`
	CreatedAt   time.Time `json:"created_at,omitempty" dynamodbav:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at"`

	Dummy string `json:"-" dynamodbav:"dummy,omitempty"`
}

func NewGymVideoHandler(ctx context.Context, dynamoEndpoint string) (*GymVideoHandler, error) {
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

	return &GymVideoHandler{
		Client:        db,
		AuthService:   authSVC,
		PresignClient: psc,
		videosTable:   os.Getenv("GYM_VIDEOS_TABLE_NAME"),
	}, nil
}

func (h *GymVideoHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
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
		TableName:                 &h.videosTable,
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

	for _, v := range result.Items {
		s3Key := v["s3_object"].(*types.AttributeValueMemberS).Value
		params := &s3.GetObjectInput{
			Bucket: aws.String("grapple-gym-videos"),
			Key:    aws.String(s3Key),
		}

		r, err := h.PresignClient.PresignGetObject(context.TODO(), params, func(opts *s3.PresignOptions) {
			opts.Expires = time.Minute * 30
		})
		if err != nil {
			log.Info().Msgf("Couldn't get a presigned download url for object %v. Here's why: %v", s3Key, err)
			return lambda.ServerError(err)
		}
		log.Info().Msgf("Fetched presigned S3 url for video: %v", r.URL)
		v["url"] = &types.AttributeValueMemberS{
			Value: r.URL,
		}
	}

	var gymVideos []GymVideo
	resp, err := dynamodbsdk.MarshalResponse(
		aws.String("updated_at"), limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &gymVideos,
	)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("error marshalling response: %v", err))
	}

	log.Info().Msgf("resp.Data: %v", resp.Data)

	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	result, err := h.GetByID(ctx, h.videosTable, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	if len(result.Items) > 0 {
		// Get pre-signed URL
		s3Key := result.Items[0]["s3_object"].(*types.AttributeValueMemberS).Value
		params := &s3.GetObjectInput{
			Bucket: aws.String("grapple-gym-videos"),
			Key:    aws.String(s3Key),
		}

		r, err := h.PresignClient.PresignGetObject(context.TODO(), params, func(opts *s3.PresignOptions) {
			opts.Expires = time.Minute * 30
		})
		if err != nil {
			log.Info().Msgf("Couldn't get a presigned download url for object %v. Here's why: %v", s3Key, err)
			return lambda.ServerError(err)
		}

		log.Info().Msgf("Fetched presigned S3 url for video: %v", r.URL)
		result.Items[0]["url"] = &types.AttributeValueMemberS{
			Value: r.URL,
		}
	}

	var requests []GymVideo
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(requests[0])
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched Gyms by ID: %s", string(json))

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil

}

func (h *GymVideoHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymVideo GymVideo
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

	res, err := h.Insert(ctx, h.videosTable, &gymVideo)
	if err != nil {
		return lambda.ServerError(err)
	}

	var returnGym GymVideo
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

func (h *GymVideoHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	// Fetch the Gym Request
	result, err := h.GetByID(ctx, h.videosTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym request not found")
	}

	var videos []GymVideo
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

	resp, err := h.Delete(ctx, h.videosTable, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}
func (h *GymVideoHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	var payload GymVideo
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

	resp, err := h.Update(ctx, h.videosTable, key, &expr)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var gymVideo GymVideo
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

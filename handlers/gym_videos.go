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
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GymVideoHandler struct {
	*dynamodbsdk.Client
	*AuthService
	videosTable string
}

type GymVideo struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID   string `json:"gym_id" dynamodbav:"gym_id,omitempty"`
	Title   string `json:"title" dynamodbav:"title,omitempty"`
	Content string `json:"content" dynamodbav:"content,omitempty"`

	Difficulty  string   `json:"difficulty,omitempty"`
	Disciplines []string `json:"disciplines" dynamodbav:"disciplines,stringsets,omitempty"`
	URL         string   `json:"url,omitempty"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
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

	return &GymVideoHandler{
		Client:      db,
		AuthService: authSVC,
		videosTable: os.Getenv("GYM_VIDEOS_TABLE_NAME"),
	}, nil
}

func (h *GymVideoHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	gym := req.QueryStringParameters["gym"]
	discipline := req.QueryStringParameters["discipline"]
	if gym == "" {
		return lambda.ClientError(http.StatusBadRequest, "must specify ?gym query parameter")
	}

	filter := dynamodbsdk.Filter{
		FilterExpression: aws.String("gym_id = :gym_id"),
		ExpressionAttributeValues: map[string]any{
			":gym_id": gym,
		},
	}

	if discipline != "" {
		filter.ExpressionAttributeValues[":discipline"] = discipline
		expr := fmt.Sprintf("%s and contains(disciplines, :discipline)", *filter.FilterExpression)
		filter.FilterExpression = aws.String(expr)
	}

	indexName := aws.String("GymIndex")
	result, err := h.QueryPage(ctx, h.videosTable, limit, startKey, indexName, &filter, true)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("filter: %+v, err: %w", filter, err))
	}

	var gymVideos []GymVideo
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gymVideos)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("bad request: %v", err))
	}

	lastEvaluatedKey := dynamodbsdk.LastEvaluated{}
	if err := attributevalue.UnmarshalMap(result.LastEvaluatedKey, &lastEvaluatedKey); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("error unmarshalling last evaluated key: %v", err))
	}
	responseObject := dynamodbsdk.GetResponse{
		Data:      gymVideos,
		Count:     result.Count,
		NextToken: &lastEvaluatedKey,
	}
	if result.Count == 0 {
		responseObject.NextToken = nil
	}

	json, err := json.Marshal(responseObject)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched GymVideo item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymVideos by ID request")

	result, err := h.GetByID(ctx, h.videosTable, id)
	if err != nil {
		return lambda.ServerError(err)
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
	gymVideo.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymVideo#%s/%s/%d", gymVideo.GymID, gymVideo.Title, gymVideo.CreatedAt.Unix())),
	)
	log.Info().Msgf("Inserting gym video: %++v", gymVideo)

	if err := json.Unmarshal([]byte(req.Body), &gymVideo); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}
	if err := validate.Struct(&gymVideo); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("request body failed validation: %v", err))
	}

	gymVideo.CreatedAt = time.Now().UTC()
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

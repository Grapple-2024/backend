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
}

type GymVideo struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID   string `json:"gym_id" dynamodbav:"gym_id"`
	Title   string `json:"title" dynamodbav:"title"`
	Content string `json:"content" dynamodbav:"content"`

	Difficulty string `json:"difficulty"`
	Discipline string `json:"discipline"`
	URL        string `json:"url"`

	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

func NewGymVideoHandler(ctx context.Context, dynamoEndpoint string) (*GymVideoHandler, error) {
	tableName := os.Getenv("GYM_VIDEOS_TABLE_NAME")

	db, err := dynamodbsdk.NewClient(dynamoEndpoint, tableName)
	if err != nil {
		return nil, err
	}

	return &GymVideoHandler{
		Client: db,
	}, nil
}

func (h *GymVideoHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error) {
	gym := req.QueryStringParameters["gym"]

	var filter dynamodbsdk.Filter
	var indexName *string
	if gym != "" {
		filter.FilterExpression = aws.String("gym_id = :gym_id")
		filter.ExpressionAttributeValues = map[string]any{
			":gym_id": gym,
		}
		indexName = aws.String("GymIndex")
	}
	result, err := h.Get(ctx, limit, exclusiveStartKey, indexName, &filter)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("filter: %+v, err: %w", filter, err))
	}

	var gymVideos []GymVideo
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gymVideos)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}

	lastEvaluatedID := ""
	if len(gymVideos) > 0 {
		lastEvaluatedID = gymVideos[len(gymVideos)-1].PK
	}
	responseObject := GetResponse{
		Data:             gymVideos,
		LastEvaluatedKey: &lastEvaluatedID,
		Count:            result.Count,
	}

	json, err := json.Marshal(responseObject)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched GymVideo item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoHandler) ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymVideos by ID request")

	result, err := h.GetByID(ctx, id)
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
	res, err := h.Insert(ctx, &gymVideo)
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
	result, err := h.GetByID(ctx, id)
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

	resp, err := h.Delete(ctx, key)
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

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	)
	if payload.Title != "" {
		builder.WithUpdate(
			expression.Set(
				expression.Name("title"),
				expression.Value(payload.Title),
			))
	}
	if payload.Content != "" {
		builder.WithUpdate(
			expression.Set(
				expression.Name("content"),
				expression.Value(payload.Content),
			))
	}

	// Update the timestamp on the announcement
	payload.UpdatedAt = time.Now().UTC()

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

	resp, err := h.Update(ctx, key, &expr)
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

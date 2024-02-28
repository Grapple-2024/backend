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

type GymAnnouncementHandler struct {
	*dynamodbsdk.Client
}

type GymAnnouncement struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID     string    `json:"gym_id" dynamodbav:"gym_id"`
	Title     string    `json:"title" dynamodbav:"title"`
	Content   string    `json:"content" dynamodbav:"content"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
}

func NewGymAnnouncementHandler(ctx context.Context, dynamoEndpoint string) (*GymAnnouncementHandler, error) {
	tableName := os.Getenv("GYM_ANNOUNCEMENTS_TABLE_NAME")

	db, err := dynamodbsdk.NewClient(dynamoEndpoint, tableName)
	if err != nil {
		return nil, err
	}

	log.Info().Msgf("Gym announcements table: %v", tableName)
	return &GymAnnouncementHandler{
		Client: db,
	}, nil
}

func (h *GymAnnouncementHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error) {
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

	var gymAnnouncements []GymAnnouncement
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gymAnnouncements)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}

	lastEvaluatedID := ""
	if len(gymAnnouncements) > 0 {
		lastEvaluatedID = gymAnnouncements[len(gymAnnouncements)-1].PK
	}
	responseObject := GetResponse{
		Data:             gymAnnouncements,
		LastEvaluatedKey: &lastEvaluatedID,
		Count:            result.Count,
	}

	json, err := json.Marshal(responseObject)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched GymAnnouncement item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymAnnouncements by ID request")

	result, err := h.GetByID(ctx, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	var requests []GymAnnouncement
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

func (h *GymAnnouncementHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymAnnouncement GymAnnouncement
	if err := json.Unmarshal([]byte(req.Body), &gymAnnouncement); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	err := validate.Struct(&gymAnnouncement)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	gymAnnouncement.CreatedAt = time.Now().UTC()
	gymAnnouncement.UpdatedAt = gymAnnouncement.CreatedAt

	gymAnnouncement.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymAnnouncement#%s/%s/%d", gymAnnouncement.GymID, gymAnnouncement.Title, gymAnnouncement.CreatedAt.Unix())),
	)
	res, err := h.Insert(ctx, &gymAnnouncement)
	if err != nil {
		return lambda.ServerError(err)
	}

	var returnGym GymAnnouncement
	err = attributevalue.UnmarshalMap(res.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&gymAnnouncement)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	var announcements []GymAnnouncement
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &announcements); err != nil {
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
func (h *GymAnnouncementHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	var payload GymAnnouncement
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	).WithUpdate(
		expression.Set(
			expression.Name("title"),
			expression.Value(payload.Title),
		).Set(
			expression.Name("content"),
			expression.Value(payload.Content),
		),
	)

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

	var gymAnnouncement GymAnnouncement
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gymAnnouncement); err != nil {
		return lambda.ServerError(err)
	}

	log.Info().Msgf("Gym request: %v", gymAnnouncement)
	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

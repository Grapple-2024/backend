package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
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
	*AuthService
	*dynamodbsdk.Client
	announcementsTable string
}

type GymAnnouncement struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID     string    `json:"gym_id" dynamodbav:"gym_id"`
	Title     string    `json:"title" dynamodbav:"title,omitempty"`
	Content   string    `json:"content" dynamodbav:"content,omitempty"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
	Dummy     string    `json:"-" dynamodbav:"dummy"`
}

func NewGymAnnouncementHandler(ctx context.Context, dynamoEndpoint string) (*GymAnnouncementHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &GymAnnouncementHandler{
		AuthService:        authSVC,
		Client:             db,
		announcementsTable: os.Getenv("GYM_ANNOUNCEMENTS_TABLE_NAME"),
	}, nil
}

func (h *GymAnnouncementHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	gym := req.QueryStringParameters["gym"]
	if gym == "" {
		return lambda.ClientError(http.StatusBadRequest, "?gym query parameter is required")
	}
	var ascending bool
	ascendingS := req.QueryStringParameters["ascending"]
	if ascendingS == "" {
		ascendingS = "true"
	}

	ascending, err := strconv.ParseBool(ascendingS)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "?ascending query param must be a boolean")
	}

	isNotCoach := h.IsCoach(ctx, req.Headers, gym)
	isNotStudent := h.IsStudent(ctx, req.Headers, gym)
	if isNotCoach != nil && isNotStudent != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("user is neither a coach or student of this gym: %v\n %v", isNotStudent, isNotCoach))
	}

	indexName := aws.String("LastUpdatedIndex")
	filter := dynamodbsdk.Filter{
		KeyConditionExpression: aws.String("dummy = :dummy"),
		ExpressionAttributeValues: map[string]any{
			":dummy": "dumb",
		},
	}

	if startKey["pk"] != nil {
		startKey["gym_id"] = &types.AttributeValueMemberS{
			Value: gym,
		}
	}

	log.Info().Msgf("QueryPage with startKey: %+v", startKey)
	result, err := h.QueryPage(ctx, h.announcementsTable, limit, startKey, indexName, &filter, ascending)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("filter: %+v, err: %w", filter, err))
	}

	var gymAnnouncements []GymAnnouncement
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gymAnnouncements)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}
	lastEvaluatedKey := dynamodbsdk.Key{}
	if err := attributevalue.UnmarshalMap(result.LastEvaluatedKey, &lastEvaluatedKey); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("error unmarshalling last evaluated key: %v", err))
	}
	responseObject := dynamodbsdk.GetResponse{
		Data:      gymAnnouncements,
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
	log.Printf("Successfully fetched GymAnnouncement item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymAnnouncements by ID request")

	result, err := h.GetByID(ctx, h.announcementsTable, id)
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

	if err := h.IsCoach(ctx, req.Headers, gymAnnouncement.GymID); err != nil {
		// user is not a coach of this gym, deny the request to create an announcement
		return lambda.ClientError(http.StatusForbidden, err.Error())
	}

	if err := validate.Struct(&gymAnnouncement); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	gymAnnouncement.CreatedAt = time.Now().UTC()
	gymAnnouncement.UpdatedAt = gymAnnouncement.CreatedAt
	gymAnnouncement.Dummy = "dumb"
	gymAnnouncement.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymAnnouncement#%s/%d", gymAnnouncement.GymID, gymAnnouncement.CreatedAt.Unix())),
	)

	res, err := h.Insert(ctx, h.announcementsTable, &gymAnnouncement)
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
	result, err := h.GetByID(ctx, h.announcementsTable, id)
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

	resp, err := h.Delete(ctx, h.announcementsTable, key)
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

	// Marshal to AV
	av, _ := attributevalue.MarshalMap(payload)
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "gym_id" || k == "created_at" || k == "updated_at" {
			continue
		}
		update = update.Set(expression.Name(k), expression.Value(v))
	}

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	).WithUpdate(update)

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

	resp, err := h.Update(ctx, h.announcementsTable, key, &expr)
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

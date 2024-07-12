package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
)

type UserPreferences struct {
	UserID string `json:"user_id" dynamodbav:"user_id"`

	// only for students
	NotifyOnAnnouncements bool `json:"notify_on_announcements" dynamodbav:"notify_on_announcement"`

	// only for coaches
	NotifyOnGymRequests bool `json:"notify_on_gym_requests" dynamodbav:"notify_on_gym_request"`

	CreatedAt time.Time `json:"created_at,omitempty" dynamodbav:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at,omitempty"`
}

type UserPreferencesHandler struct {
	*AuthService
	DynamoClient             *dynamodbsdk.Client
	userPreferencesTableName string
}

func NewUserPreferencesHandler(ctx context.Context, dynamoEndpoint, region string) (*UserPreferencesHandler, error) {
	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &UserPreferencesHandler{
		DynamoClient:             db,
		AuthService:              authSVC,
		userPreferencesTableName: os.Getenv("USER_PREFERENCES_TABLE_NAME"),
	}, nil
}

func (h *UserPreferencesHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// Unmarshal request body into UserPreferences struct
	var up UserPreferences
	if err := json.Unmarshal([]byte(req.Body), &up); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", err))
	}
	up.UserID = token.Sub
	up.CreatedAt = time.Now().UTC()
	up.UpdatedAt = up.CreatedAt

	_, err = h.DynamoClient.Insert(ctx, h.userPreferencesTableName, up, "user_id")
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to insert user asset into dynamo: %v", err))
	}

	bytes, err := json.Marshal(up)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error marshaling presigned url response to json: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(bytes), nil), nil
}

func (h *UserPreferencesHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}
	if token.Sub != id {
		return lambda.ClientError(http.StatusForbidden, "permission denied: ID in url path does not match token's user ID")
	}

	// Unmarshal JSON http request body into UserPreferences struct
	var up UserPreferences
	if err := json.Unmarshal([]byte(req.Body), &up); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	// Send update request to Dynamodb
	r, err := h.updateUserPreference(ctx, id, &up)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update user preference: %v", err))
	}

	// Unmarshal Dynamodb response into GymVideo
	var returnUP UserPreferences
	if err := attributevalue.UnmarshalMap(r.Attributes, &returnUP); err != nil {
		return lambda.ServerError(err)
	}

	// Marshal GymVideo into JSON and serve the response
	json, err := json.Marshal(returnUP)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h UserPreferencesHandler) updateUserPreference(ctx context.Context, id string, up *UserPreferences) (*dynamodb.UpdateItemOutput, error) {
	// Marshal request payload into map[string]types.AttributeValue
	av, err := attributevalue.MarshalMap(up)
	if err != nil {
		return nil, err
	}

	// Build update expression for dynamodb
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "user_id" || k == "created_at" || k == "updated_at" {
			continue // continue on immutable fields
		}
		update = update.Set(expression.Name(k), expression.Value(v))
	}
	update = update.Set(expression.Name("updated_at"), expression.Value(time.Now().UTC()))
	builder := expression.NewBuilder().WithCondition(expression.Equal(
		expression.Name("user_id"),
		expression.Value(id),
	),
	).WithUpdate(update)

	expr, err := builder.Build()
	if err != nil {
		return nil, err
	}

	userID, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}

	key := map[string]types.AttributeValue{
		"user_id": userID,
	}

	log.Info().Msgf("Updating preference with Key: %+v", key["user_id"])
	return h.Update(ctx, h.userPreferencesTableName, key, &expr)
}

func (h *UserPreferencesHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	up, err := h.getUserPreferences(ctx, token.Sub)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to get user preferences: %v", err))
	}

	log.Info().Msgf("found user preferences for token: %++v, token: %v", up, token.Sub)
	upBytes, err := json.Marshal(up)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to marshal response from dynamo: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(upBytes), nil), nil
}

func (h *UserPreferencesHandler) getUserPreferences(ctx context.Context, id string) (*UserPreferences, error) {
	// construct key for the object to fetch
	userID, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}
	key := map[string]types.AttributeValue{
		"user_id": userID,
	}

	qo, err := h.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &h.userPreferencesTableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	} else if qo.Item == nil {
		return nil, fmt.Errorf("no user preferences found for user %s", id)
	}

	var up UserPreferences
	if err = attributevalue.UnmarshalMap(qo.Item, &up); err != nil {
		return nil, err
	}

	return &up, nil
}

// Needed to satisfy interface, but not implemented
func (h *UserPreferencesHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *UserPreferencesHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

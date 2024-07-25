package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/pkg/dynamodb"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
)

type UserProfile struct {
	UserID string `json:"user_id" dynamodbav:"user_id"`

	// only for students
	NotifyOnAnnouncements bool `json:"notify_on_announcements" dynamodbav:"notify_on_announcements"`

	// only for coaches
	NotifyOnGymRequests bool `json:"notify_on_gym_requests" dynamodbav:"notify_on_gym_requests"`

	// list of users public assets (profile images for now)
	Assets []UserAsset `json:"assets" dynamodbav:"assets"`

	CreatedAt time.Time `json:"created_at,omitempty" dynamodbav:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at,omitempty"`
}

type UserProfileHandler struct {
	*AuthService
	DynamoClient          *dynamodbsdk.Client
	userProfilesTableName string
	userAssetsTableName   string
}

func NewUserProfileHandler(ctx context.Context, dynamoEndpoint, region string) (*UserProfileHandler, error) {
	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &UserProfileHandler{
		DynamoClient:          db,
		AuthService:           authSVC,
		userProfilesTableName: os.Getenv("USER_PROFILES_TABLE_NAME"),
		userAssetsTableName:   os.Getenv("PUBLIC_USER_ASSETS_TABLE_NAME"),
	}, nil
}

func (h *UserProfileHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// Unmarshal request body into UserProfile struct
	var up UserProfile
	if err := json.Unmarshal([]byte(req.Body), &up); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", err))
	}
	up.UserID = token.Sub
	up.CreatedAt = time.Now().UTC()
	up.UpdatedAt = up.CreatedAt

	_, err = h.DynamoClient.Insert(ctx, h.userProfilesTableName, up, "user_id")
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to insert user asset into dynamo: %v", err))
	}

	userAssets, err := h.getUserAssets(ctx, token.Sub)
	if err == nil {
		up.Assets = userAssets
	}

	bytes, err := json.Marshal(up)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error marshaling presigned url response to json: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(bytes), nil), nil
}

func (h *UserProfileHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
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

	// Unmarshal JSON http request body into UserProfile struct
	var up UserProfile
	if err := json.Unmarshal([]byte(req.Body), &up); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	// Send update request to Dynamodb
	r, err := h.updateUserProfile(ctx, id, &up)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update user preference: %v", err))
	}

	returnUP := UserProfile{
		Assets: []UserAsset{},
	}
	if err := attributevalue.UnmarshalMap(r.Attributes, &returnUP); err != nil {
		return lambda.ServerError(err)
	}

	// fetch assets for this user
	userAssets, err := h.getUserAssets(ctx, token.Sub)
	if err != nil {
		log.Warn().Msgf("failed to find user assets for user ID %s", token.Sub)
	} else {
		returnUP.Assets = userAssets
	}

	json, err := json.Marshal(returnUP)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h UserProfileHandler) updateUserProfile(ctx context.Context, id string, up *UserProfile) (*dynamodb.UpdateItemOutput, error) {
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
	// update = update.Set(expression.Name("user_id"), expression.Value(id))

	// // builder := expression.NewBuilder().WithCondition(expression.Equal(
	// // 	expression.Name("user_id"),
	// // 	expression.Value(id),
	// // ),
	// ).WithUpdate(update)

	builder := expression.NewBuilder().WithUpdate(update)

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

	log.Info().Msgf("Updating user profile with Key: %+v", key["user_id"])
	return h.Update(ctx, h.userProfilesTableName, key, &expr, true)
}

func (h *UserProfileHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	id, err := attributevalue.Marshal(token.Sub)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("cannot marshal token to attributevalue: %v", err))
	}

	userIDPK := map[string]types.AttributeValue{
		"user_id": id,
	}
	params := &dynamodb.GetItemInput{
		TableName: &h.userProfilesTableName,
		Key:       userIDPK,
	}

	out, err := h.GetItem(ctx, params, func(opts *dynamodb.Options) {
		opts.Region = "us-west-1"
	})
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to create GET transaction: %v", err))
	} else if out.Item == nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("no user profile found for token %s", token.Sub))
	}

	log.Info().Msgf("User profile: %+v", out.Item)
	var up UserProfile
	if err := attributevalue.UnmarshalMap(out.Item, &up); err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to fetch user profile from dynamo: %v", err))
	}
	up.Assets = []UserAsset{}

	// check if there are any assets for this user
	userAssets, err := h.getUserAssets(ctx, token.Sub)
	if err != nil {
		log.Warn().Msgf("failed to find user assets for user ID %s", token.Sub)
	} else {
		up.Assets = userAssets
	}

	upBytes, err := json.Marshal(up)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to marshal response from dynamo: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(upBytes), nil), nil
}

func (h *UserProfileHandler) getUserProfile(ctx context.Context, id string) (*UserProfile, error) {
	// construct key for the object to fetch
	userID, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}
	key := map[string]types.AttributeValue{
		"user_id": userID,
	}

	qo, err := h.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &h.userProfilesTableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	} else if qo.Item == nil {
		return nil, fmt.Errorf("no user preferences found for user %s", id)
	}

	var up UserProfile
	if err = attributevalue.UnmarshalMap(qo.Item, &up); err != nil {
		return nil, err
	}

	return &up, nil
}

// Needed to satisfy interface, but not implemented
func (h *UserProfileHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *UserProfileHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

func (h *UserProfileHandler) getUserAssets(ctx context.Context, userID string) ([]UserAsset, error) {
	// check if there are any assets for this user
	builder := expression.NewBuilder().WithKeyCondition(
		expression.Key("user_id").Equal(expression.Value(userID)),
	)
	e, err := builder.Build()
	if err != nil {
		return nil, err
	}

	queryParams := &dynamodb.QueryInput{
		TableName:                 &h.userAssetsTableName,
		IndexName:                 aws.String("UserIndex"),
		KeyConditionExpression:    e.KeyCondition(),
		FilterExpression:          e.Filter(),
		ExpressionAttributeNames:  e.Names(),
		ExpressionAttributeValues: e.Values(),
	}

	o, err := h.Query(ctx, queryParams)
	if err != nil {
		return nil, err
	}

	var userAssets []UserAsset
	if err := attributevalue.UnmarshalListOfMaps(o.Items, &userAssets); err != nil {
		return nil, err
	}

	return userAssets, nil
}

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/MicahParks/keyfunc/v3"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type GymRequestHandler struct {
	*dynamodbsdk.Client
}

type GetResponse struct {
	Data             any     `json:"data"`
	LastEvaluatedKey *string `json:"lastEvaluatedKey"`
	Count            int32   `json:"count"`
}

type GymRequest struct {
	PK string `json:"pk" dynamodbav:"pk"`

	Requestor string `json:"requestor_id" dynamodbav:"requestor_id"`
	GymID     string `json:"gym_id" dynamodbav:"gym_id"`
	Status    string `json:"status" dynamodbav:"status"`
}

func NewGymRequestHandler(ctx context.Context, dynamoEndpoint string) (*GymRequestHandler, error) {
	gymRequestsTableName := os.Getenv("GYM_REQUESTS_TABLE_NAME")

	db, err := dynamodbsdk.NewClient(dynamoEndpoint, gymRequestsTableName)
	if err != nil {
		return nil, err
	}

	return &GymRequestHandler{
		Client: db,
	}, nil
}

func (h *GymRequestHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error) {
	requestor := req.QueryStringParameters["requestor"]
	gym := req.QueryStringParameters["gym"]

	var filter dynamodbsdk.Filter
	var indexName *string
	if requestor != "" && gym != "" {
		// get a request by requestor and gym IDs
		filter.FilterExpression = aws.String("pk = :pk")
		filter.ExpressionAttributeValues = map[string]any{
			":pk": fmt.Sprintf("gymRequest#%s/%s", requestor, gym),
		}
	} else if requestor == "" && gym != "" {
		// get all requests for a given gym
		filter.FilterExpression = aws.String("gym_id = :gym_id")
		filter.ExpressionAttributeValues = map[string]any{
			":gym_id": gym,
		}
		indexName = aws.String("GymIndex")

	} else if requestor != "" && gym == "" {
		filter.FilterExpression = aws.String("requestor_id = :requestor_id")
		filter.ExpressionAttributeValues = map[string]any{
			":requestor_id": requestor,
		}
		indexName = aws.String("RequestorIndex")
	}

	result, err := h.Get(ctx, limit, exclusiveStartKey, indexName, &filter)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("filter: %+v, err: %w", filter, err))
	}

	var gymRequests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gymRequests)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}

	lastEvaluatedID := ""
	if len(gymRequests) > 0 {
		lastEvaluatedID = gymRequests[len(gymRequests)-1].PK
	}
	responseObject := GetResponse{
		Data:             gymRequests,
		LastEvaluatedKey: &lastEvaluatedID,
		Count:            result.Count,
	}

	json, err := json.Marshal(responseObject)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched GymRequest item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymRequestHandler) ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymRequests by ID request")

	result, err := h.GetByID(ctx, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	var requests []GymRequest
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

func (h *GymRequestHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var gymRequest GymRequest
	if err := json.Unmarshal([]byte(req.Body), &gymRequest); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	err = validate.Struct(&gymRequest)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	gymRequest.PK = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("gymRequest#%s/%s", gymRequest.Requestor, gymRequest.GymID)))
	gymRequest.Requestor = token.User
	gymRequest.Status = "Pending"
	res, err := h.Insert(ctx, &gymRequest)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Info().Msgf("Insert result: %+v", res)

	var returnGym GymRequest
	err = attributevalue.UnmarshalMap(res.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&gymRequest)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymRequestHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

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

	var requests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}
	requestor := requests[0].Requestor
	if requestor != token.User {
		return lambda.ClientError(http.StatusForbidden, "permission denied: you must be the creator of the gym request to delete it")
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
func (h *GymRequestHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	var payload GymRequest
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
			expression.Name("status"),
			expression.Value(payload.Status),
		),
	)

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

	var gymRequest GymRequest
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gymRequest); err != nil {
		return lambda.ServerError(err)
	}

	log.Info().Msgf("Gym request: %v", gymRequest)
	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// validateJWT takes a token string and validates it
func (h *GymRequestHandler) ValidateJWT(tokenString string) error {
	regionID := "us-west-1"
	userPoolID := "us-west-1_HT5oR6AwO"
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return err
	}

	// Parse the JWT.
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return err
	}

	// Check if the token is valid.
	if !token.Valid {
		return err
	}

	log.Info().Msgf("Token is valid!\n%+v\n", token)
	log.Info().Msgf("Token claim!\n%+v\n", token.Claims)

	return nil
}

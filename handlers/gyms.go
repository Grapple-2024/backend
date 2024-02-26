package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	cip "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/go-http-utils/headers"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"

	"github.com/Grapple-2024/backend/cognito"
	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	validator "github.com/go-playground/validator/v10"
)

const (
	gymsTable = "grapple-gyms"
)

type GymHandler struct {
	*dynamodbsdk.Client
	CognitoClient *cognito.Client
}

type Gym struct {
	PK string `json:"pk" dynamodbav:"pk"`
	SK string `json:"sk" dynamodbav:"sk"`

	Name    string `json:"name" dynamodbav:"name"`
	Creator string `json:"creator" dynamodbav:"creator"`

	// Address
	AddressLine1 string `json:"address_line_1" dynamodbav:"address_line_1"`
	AddressLine2 string `json:"address_line_2" dynamodbav:"address_line_2"`
	City         string `json:"city" dynamodbav:"city"`
	State        string `json:"state" dynamodbav:"state"`
	ZIP          string `json:"zip" dynamodbav:"zip"`
	Country      string `json:"country" dynamodbav:"country"`

	// Disciplines
	Disciplines []string           `json:"disciplines" dynammodbav:"disciplines"`
	Schedule    map[string][]Event `json:"schedule" dynamodbav:"schedule"`
}

type Event struct {
	Title string `json:"title" dynamodbav:"title"`
	Start string `json:"start" dynamodbav:"start"`
	End   string `json:"end" dynamodbav:"end"`
}

var (
	validate = validator.New()

	// AWS Cognito
	userPoolID          = "us-west-1_HT5oR6AwO"
	region              = "us-west-1"
	cognitoClientID     = "40s9oop5e9srair8mljupn000j"
	cognitoClientSecret = "1fifmgpshit01l5eqppj95o1kjt2v16n32kaunve5ntv2n938ei9"
)

func NewGymHandler(ctx context.Context, dynamoEndpoint string) (*GymHandler, error) {
	gymsTableName := os.Getenv("GYMS_TABLE_NAME")
	db, err := dynamodbsdk.NewClient(dynamoEndpoint, gymsTableName)
	if err != nil {
		return nil, err
	}

	cc, err := cognito.NewClient(
		region,
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		return nil, err
	}

	return &GymHandler{
		Client:        db,
		CognitoClient: cc,
	}, nil
}

func (h *GymHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error) {
	creatorID := req.QueryStringParameters["creator"]

	var filter dynamodbsdk.Filter
	if creatorID != "" {
		e := "sk = :i"
		filter.FilterExpression = &e
		filter.ExpressionAttributeValues = map[string]any{
			":i": creatorID,
		}
	}

	result, err := h.Get(ctx, limit, exclusiveStartKey, nil, &filter)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("filter: %+v, err: %w", filter, err))
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gyms)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}

	lastEvaluatedID := ""
	if len(gyms) > 0 {
		lastEvaluatedID = gyms[len(gyms)-1].PK
	}
	responseObject := GetResponse{
		Data:             gyms,
		LastEvaluatedKey: &lastEvaluatedID,
		Count:            result.Count,
	}

	json, err := json.Marshal(responseObject)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched Gym item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymHandler) ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
	log.Info().Msgf("Received GET Gym by ID request with ID: %v", id)

	result, err := h.GetByID(ctx, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, err.Error())
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gyms)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(gyms[0])
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched Gyms by ID: %s", string(json))

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Validate JWT token
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to verify jwt token: %v", err))
	}

	// Unmarshal request body
	var gym Gym
	if err = json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	// Validate request body
	err = validate.Struct(&gym)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	// Insert Gym into dynamodb
	gym.PK = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("gym#%s/%s", gym.Creator, gym.Name)))
	gym.SK = gym.Creator
	log.Info().Msgf("Inserting gym: %+v", gym)
	res, err := h.Insert(ctx, &gym)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Info().Msgf("Insert result: %+v", res)

	var returnGym Gym
	err = attributevalue.UnmarshalMap(res.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&gym)
	if err != nil {
		return lambda.ServerError(err)
	}

	// Add user to cognito group
	log.Info().Msgf("Token.User: %v", token.User)
	log.Info().Msgf("user pool id: %v", userPoolID)

	coachGroup := "coach"
	_, err = h.CognitoClient.AdminAddUserToGroup(&cip.AdminAddUserToGroupInput{
		UserPoolId: &userPoolID,
		Username:   aws.String(token.User),
		GroupName:  &coachGroup,
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to add creator to coach group: %v", err))
	}
	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, "permission denied deleting gym")
	}

	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	// Check permissions
	if !slices.Contains(token.Roles, "arn:aws:iam::381491926210:role/us-west-1_HT5oR6AwO-coachGroupRole") {
		return lambda.ClientError(http.StatusForbidden, "permission denied: you must be a coach to delete a gym")
	}
	result, err := h.GetByID(ctx, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gyms)
	if err != nil {
		return lambda.ServerError(err)
	}
	if gyms[0].SK != token.User {
		return lambda.ClientError(http.StatusForbidden, "permission denied: you must be the creator of the gym to delete it")
	}

	log.Printf("Received DELETE request with id = %s", id)

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	sk, err := attributevalue.Marshal(gyms[0].SK)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
		"sk": sk,
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

func (h *GymHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	// validate and fetch token from header
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied updating gym: %v", err))
	}

	// Fetch the Gym
	result, err := h.GetByID(ctx, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gyms)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request")
	}

	// Check that the user owns the gym
	if token.User != gyms[0].SK {
		return lambda.ClientError(
			http.StatusForbidden,
			fmt.Sprintf("permission denied updating gym: resource is owned by another user: %v", err),
		)
	}

	// Update the Gym
	var gymUpdatePayload Gym
	if err := json.Unmarshal([]byte(req.Body), &gymUpdatePayload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	).WithUpdate(
		// expression.Set(
		// 	expression.Name("name"),
		// 	expression.Value(gymUpdatePayload.Name),
		// ).
		expression.Set(
			expression.Name("schedule"),
			expression.Value(gymUpdatePayload.Schedule),
		).Set(
			expression.Name("address_line_1"),
			expression.Value(gymUpdatePayload.AddressLine1),
		).Set(
			expression.Name("address_line_2"),
			expression.Value(gymUpdatePayload.AddressLine2),
		).Set(
			expression.Name("city"),
			expression.Value(gymUpdatePayload.City),
		).Set(
			expression.Name("state"),
			expression.Value(gymUpdatePayload.State),
		).Set(
			expression.Name("zip"),
			expression.Value(gymUpdatePayload.ZIP),
		).Set(
			expression.Name("country"),
			expression.Value(gymUpdatePayload.Country),
		).Set(
			expression.Name("disciplines"),
			expression.Value(gymUpdatePayload.Disciplines),
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
	sk, err := attributevalue.Marshal(gyms[0].SK)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
		"sk": sk,
	}

	resp, err := h.Update(ctx, key, &expr)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var gym Gym
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gym); err != nil {
		return lambda.ServerError(err)
	}
	log.Info().Msgf("Updated Gym: %+v", gym)

	json, err := json.Marshal(&gym)
	if err != nil {
		return lambda.ServerError(err)

	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// validateJWT takes a token string and validates it
type Token struct {
	User  string   `mapstructure:"cognito:username"`
	Roles []string `mapstructure:"cognito:roles"`
}

func ValidateJWT(hdrs map[string]string) (*Token, error) {
	authHeader := hdrs[headers.Authorization]
	if len(authHeader) <= 1 {
		return nil, fmt.Errorf("auth header not valid: %v", authHeader)
	}
	bearer := strings.Split(authHeader, "Bearer")
	var err error
	if len(bearer) <= 1 {
		return nil, fmt.Errorf("auth header not valid: %v", authHeader)
	}

	tokenString := strings.TrimSpace(bearer[1])

	regionID := "us-west-1"
	userPoolID := "us-west-1_HT5oR6AwO"
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}

	// Parse the JWT.
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return nil, err
	}

	// Check if the token is valid.
	if !token.Valid {
		return nil, err
	}

	log.Info().Msgf("Token is valid: %+v", token)
	log.Info().Msgf("Token claim: %+v", token.Claims)

	var t *Token
	if err := mapstructure.Decode(token.Claims.(jwt.MapClaims), &t); err != nil {
		return nil, err
	}
	log.Info().Msgf("Token decoded: %+v", t)

	return t, nil
}

package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	cip "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/rs/zerolog/log"

	"github.com/Grapple-2024/backend/pkg/cognito"
	dynamodbsdk "github.com/Grapple-2024/backend/pkg/dynamodb"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	validator "github.com/go-playground/validator/v10"
)

type GymHandler struct {
	*dynamodbsdk.Client
	*AuthService
	CognitoClient *cognito.Client
	gymsTable     string
}

type Gym struct {
	PK string `json:"pk" dynamodbav:"pk"`

	Name        string `json:"name,omitempty" dynamodbav:"name"`
	Description string `json:"description,omitempty" dynamodbav:"description"`
	Creator     string `json:"creator" dynamodbav:"creator"`

	// Address
	AddressLine1 string `json:"address_line_1,omitempty" dynamodbav:"address_line_1,omitempty"`
	AddressLine2 string `json:"address_line_2,omitempty" dynamodbav:"address_line_2,omitempty"`
	City         string `json:"city,omitempty" dynamodbav:"city,omitempty"`
	State        string `json:"state,omitempty" dynamodbav:"state,omitempty"`
	ZIP          string `json:"zip,omitempty" dynamodbav:"zip,omitempty"`
	Country      string `json:"country,omitempty" dynamodbav:"country,omitempty"`
	PublicEmail  string `json:"public_email,omitempty" dynamodbav:"public_email,omitempty"`
	CoachEmail   string `json:"coach_email" dynamodbav:"coach_email,omitempty"`

	// s3 object key of banner image in grapple s3 bucket
	BannerImage string `json:"banner_image,omitempty" dynamodbav:"banner_image,omitempty"`

	// Disciplines
	Disciplines []string           `json:"disciplines" dynamodbav:"disciplines,omitempty,stringsets"`
	Schedule    map[string][]Event `json:"schedule,omitempty" dynamodbav:"schedule,omitempty"`
}

type Event struct {
	Title string `json:"title" dynamodbav:"title,omitempty"`
	Start string `json:"start" dynamodbav:"start,omitempty"`
	End   string `json:"end" dynamodbav:"end,omitempty"`
}

var (
	validate = validator.New()

	// AWS Cognito
	userPoolID = "us-west-1_HT5oR6AwO"
	region     = "us-west-1"
	// cognitoClientID     = "40s9oop5e9srair8mljupn000j"
	// cognitoClientSecret = "1fifmgpshit01l5eqppj95o1kjt2v16n32kaunve5ntv2n938ei9"
)

func NewGymHandler(ctx context.Context, dynamoEndpoint, cognitoClientID, cognitoClientSecret string) (*GymHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
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

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &GymHandler{
		Client:        db,
		CognitoClient: cc,
		AuthService:   authSVC,
		gymsTable:     os.Getenv("GYMS_TABLE_NAME"),
	}, nil
}

func (h *GymHandler) scanGyms(ctx context.Context, expr *expression.Expression, limit *int32, startKey map[string]types.AttributeValue) (*dynamodbsdk.GetResponse, error) {
	input := &dynamodb.ScanInput{
		TableName: &h.gymsTable,
		Limit:     limit,
	}
	if startKey != nil {
		input.ExclusiveStartKey = startKey
	}

	if expr != nil {
		input.FilterExpression = (*expr).Filter()
		input.ExpressionAttributeNames = (*expr).Names()
		input.ExpressionAttributeValues = (*expr).Values()
	}

	result, err := h.Scan(ctx, input)
	if err != nil {
		return nil, err
	}

	var gyms []Gym
	resp, err := dynamodbsdk.MarshalResponse(nil, *limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &gyms)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *GymHandler) queryGymsByCreator(ctx context.Context, limit *int32, creatorID string, startKey map[string]types.AttributeValue) (*dynamodbsdk.GetResponse, error) {
	keyEx := expression.Key("creator").Equal(expression.Value(creatorID))
	expr, err := expression.NewBuilder().WithKeyCondition(keyEx).Build()
	if err != nil {
		return nil, err
	}

	input := &dynamodb.QueryInput{
		TableName:                 &h.gymsTable,
		IndexName:                 aws.String("CreatorIndex"),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		KeyConditionExpression:    expr.KeyCondition(),
		Limit:                     limit,
	}
	if startKey != nil {
		input.ExclusiveStartKey = startKey
	}

	result, err := h.Query(ctx, input)
	if err != nil {
		return nil, err
	}

	var gyms []Gym
	resp, err := dynamodbsdk.MarshalResponse(nil, *limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &gyms)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (h *GymHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	creatorID := req.QueryStringParameters["creator"]
	name := req.QueryStringParameters["name"]

	var resp *dynamodbsdk.GetResponse
	var err error
	if creatorID != "" {
		resp, err = h.queryGymsByCreator(ctx, &limit, creatorID, startKey)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find gyms by creator ID: %v", err))
		}

	} else if name != "" {
		condition := expression.Name("name").Contains(name)
		expr, err := expression.NewBuilder().WithFilter(condition).Build()
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to scan gyms table: %v", err))
		}

		resp, err = h.scanGyms(ctx, &expr, &limit, startKey)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to scan gyms table: %v", err))
		}
	} else {
		resp, err = h.scanGyms(ctx, nil, &limit, startKey)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to scan gyms table: %v", err))
		}
	}

	// marshal response object to json
	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	log.Info().Msgf("Received GET Gym by ID request with ID: %v", id)

	result, err := h.GetByID(ctx, h.gymsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, err.Error())
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(result.Items, &gyms)
	if err != nil {
		return lambda.ServerError(err)
	}

	if len(gyms) == 0 {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("no gyms found"))
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
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to verify jwt token: %v", err))
	}

	// Unmarshal request body
	var gym Gym
	if err = json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}
	gym.Creator = token.Sub

	// Validate request body
	err = validate.Struct(&gym)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	// Insert Gym into dynamodb
	gymPK := base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("gym#%s/%d", gym.Creator, time.Now().Unix())))
	gym.PK = gymPK
	gym.CoachEmail = token.Email
	res, err := h.Insert(ctx, h.gymsTable, &gym, "pk")
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	var returnGym Gym
	err = attributevalue.UnmarshalMap(res.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&gym)
	if err != nil {
		return lambda.ServerError(err)
	}

	// Add user to cognito group: "Coach"
	coachGroup := "coach"
	_, err = h.CognitoClient.AdminAddUserToGroup(&cip.AdminAddUserToGroupInput{
		UserPoolId: &userPoolID,
		Username:   aws.String(token.Username),
		GroupName:  &coachGroup,
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to add creator %q to coach group: %v", gym.Creator, err))
	}
	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}
	if err := h.IsCoach(ctx, req.Headers, id); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("you must be the coach of the gym to delete it: %v", err))
	}

	result, err := h.GetByID(ctx, h.gymsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}

	log.Printf("Received DELETE request with id = %s", id)

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}
	resp, err := h.Delete(ctx, h.gymsTable, key)
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

	if err := h.IsCoach(ctx, req.Headers, id); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("must be a coach to modify the gym: %v", err))
	}

	// Fetch the Gym
	result, err := h.GetByID(ctx, h.gymsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}

	// Update the Gym
	var gymUpdatePayload Gym
	if err := json.Unmarshal([]byte(req.Body), &gymUpdatePayload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	// Marshal
	av, _ := attributevalue.MarshalMap(gymUpdatePayload)
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "creator" || k == "created_at" || k == "updated_at" {
			continue
		}

		log.Info().Msgf("Updating field %v to %v", k, v)
		update = update.Set(expression.Name(k), expression.Value(v))
	}

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

	resp, err := h.Update(ctx, h.gymsTable, key, &expr, false)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}
	log.Info().Msgf("Update metadata: %v", resp.ResultMetadata)

	o, err := h.GetByID(ctx, h.gymsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, err.Error())
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(o.Items, &gyms)
	if err != nil {
		return lambda.ServerError(err)
	} else if len(gyms) == 0 {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("no gyms found"))
	}

	json, err := json.Marshal(gyms[0])
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// validateJWT takes a token string and validates it
type Token struct {
	Username string   `mapstructure:"cognito:username"`
	Email    string   `mapstructure:"email"`
	Roles    []string `mapstructure:"cognito:roles"`
	Groups   []string `mapstructure:"cognito:groups"`

	Sub string `mapstructure:"sub"`
}

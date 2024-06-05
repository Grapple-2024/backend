package handlers

import (
	"context"
	"fmt"
	"net/http"

	"github.com/Grapple-2024/backend/cognito"
	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	cip "github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
	"github.com/rs/zerolog/log"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const (
	reqPathAddCoach = "/cognito/add-coach"
)

type CognitoHandler struct {
	*dynamodbsdk.Client
	*AuthService
	*s3.PresignClient
	CognitoClient *cognito.Client
}

type User struct {
	PK   string `json:"pk" dynamodbav:"pk"`
	Role string `json:"role" dynamodbav:"role"`
}

func NewCognitoHandler(ctx context.Context, dynamoEndpoint string) (*CognitoHandler, error) {
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

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)

	return &CognitoHandler{
		AuthService:   authSVC,
		PresignClient: psc,
		CognitoClient: cc,
	}, nil
}

func (h *CognitoHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *CognitoHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *CognitoHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusCreated, string(``), nil), nil
}

func (h *CognitoHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPut: "PUT /users" currently only allows you to update a Cognito User's role to "Coach"
func (h *CognitoHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info().Msgf("Request path: %s", req.Path)

	if req.Path == reqPathAddCoach {
		// Validate JWT token
		token, err := token(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to verify jwt token: %v", err))
		}

		// Add user to cognito group: "Coach"
		coachGroup := "coach"
		_, err = h.CognitoClient.AdminAddUserToGroup(&cip.AdminAddUserToGroupInput{
			UserPoolId: &userPoolID,
			Username:   aws.String(token.User),
			GroupName:  &coachGroup,
		})
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to add Cognito user %q to coach group: %w", token.User, err))
		}
		log.Info().Msgf("Successfully added user %s to coach group!", token.User)
	}

	return lambda.NewResponse(http.StatusOK, string(`Process PUT`), nil), nil
}

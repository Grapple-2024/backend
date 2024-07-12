package handlers

import (
	"context"
	"encoding/json"
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
	reqPathGetGroup = "/cognito/get-groups"
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
	switch req.Path {
	case reqPathGetGroup:
		// Validate JWT token
		token, err := token(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to verify jwt token: %v", err))
		}

		// Add user to cognito group: "Coach"
		u, err := h.CognitoClient.AdminListGroupsForUser(&cip.AdminListGroupsForUserInput{
			UserPoolId: &userPoolID,
			Username:   aws.String(token.Username),
		})
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to list Cognito user %q to coach group: %w", token.Username, err))
		}

		// compile list of group names only
		groupNames := make([]string, len(u.Groups))
		for i, g := range u.Groups {
			groupNames[i] = *g.GroupName
		}

		json, err := json.Marshal(groupNames)
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to marshal groups to json: %w", err))
		}
		return lambda.NewResponse(http.StatusOK, string(json), nil), nil
	}

	return lambda.NewResponse(http.StatusNotFound, string(`404 not found`), nil), nil
}

func (h *CognitoHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusNotFound, string(``), nil), nil
}

func (h *CognitoHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusNotFound, string(``), nil), nil
}

func (h *CognitoHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusNotFound, string(``), nil), nil
}

// ProcessPut: "PUT /users" currently only allows you to update a Cognito User's role to "Coach"
func (h *CognitoHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info().Msgf("Request path: %s", req.Path)

	switch req.Path {

	// Add user to coach group
	case reqPathAddCoach:
		token, err := token(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to verify jwt token: %v", err))
		}

		// Add user to cognito group: "Coach"
		_, err = h.CognitoClient.AdminGetUser(&cip.AdminGetUserInput{
			UserPoolId: &userPoolID,
			Username:   aws.String(token.Username),
		})
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to add Cognito user %q to coach group: %w", token.Username, err))
		}

		log.Info().Msgf("Successfully added user %s to coach group!", token.Username)
	}

	return lambda.NewResponse(http.StatusNotFound, string(`404 not found`), nil), nil
}

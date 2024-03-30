package handlers

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Grapple-2024/backend/dynamodb"
	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/MicahParks/keyfunc/v3"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/go-http-utils/headers"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

// Service provides authorization checks against resources in the database.
type AuthService struct {
	*dynamodb.Client
	gymsTable        string
	gymRequestsTable string
}

func NewAuthService(dynamoEndpoint string) (*AuthService, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &AuthService{
		Client:           db,
		gymsTable:        os.Getenv("GYMS_TABLE_NAME"),
		gymRequestsTable: os.Getenv("GYM_REQUESTS_TABLE_NAME"),
	}, nil
}

func (s *AuthService) IsStudent(ctx context.Context, headers map[string]string, gymID string) error {
	o, err := s.GetByID(ctx, s.gymsTable, gymID)
	if err != nil {
		return err
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(o.Items, &gyms)
	if err != nil {
		return err
	}

	if len(gyms) == 0 {
		return fmt.Errorf("no gym found with id: %s", gymID)
	}
	// Get Cognito user ID
	token, err := token(headers)
	if err != nil {
		return err
	}

	filter := dynamodbsdk.Filter{
		FilterExpression: aws.String("requestor_id = :requestor_id"),
		ExpressionAttributeValues: map[string]any{
			":requestor_id": token.Sub,
		},
	}

	so, err := s.Get(ctx, s.gymRequestsTable, 10, nil, aws.String("RequestorIndex"), &filter)
	if err != nil {
		return err
	}

	log.Info().Msgf("Get gym requests by requestor output: %+v", so)

	var gymRequests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(so.Items, &gymRequests)
	if err != nil {
		return err
	}
	if len(gymRequests) == 0 {
		return fmt.Errorf("no gym requests found for requestor id %v in gym %v", token.Sub, gymID)
	}

	log.Info().Msgf("found gym request for requestor id %q: %+v", token.Sub, gymRequests[0])
	if gymRequests[0].Status != StatusAccepted {
		return fmt.Errorf("gym request is not accepted for user %v in gym %v", gymRequests[0].RequestorID, gymID)
	}

	return nil
}

func (s *AuthService) IsCoach(ctx context.Context, headers map[string]string, gymID string) error {
	o, err := s.GetByID(ctx, s.gymsTable, gymID)
	if err != nil {
		return fmt.Errorf("failed to get gym by ID %q: %w", gymID, err)
	}

	var gyms []Gym
	err = attributevalue.UnmarshalListOfMaps(o.Items, &gyms)
	if err != nil {
		return err
	}
	if len(gyms) == 0 {
		return fmt.Errorf("no gym found with id %s", gymID)
	}

	// Get Cognito user ID
	token, err := token(headers)
	if err != nil {
		return err
	}

	if gyms[0].Creator != token.Sub {
		return fmt.Errorf("user %v is not the coach of this gym. The owner of the gym is %v", token.Sub, gyms[0].Creator)
	}

	return nil
}

func token(hdrs map[string]string) (*Token, error) {
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

	var t *Token
	if err := mapstructure.Decode(token.Claims.(jwt.MapClaims), &t); err != nil {
		return nil, err
	}

	return t, nil
}

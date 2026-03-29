package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service/announcements"
	"github.com/Grapple-2024/backend/internal/service/email"
	"github.com/Grapple-2024/backend/internal/service/gym_requests"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/mapbox"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	s3_service "github.com/Grapple-2024/backend/internal/service/s3"
	"github.com/Grapple-2024/backend/internal/service/search"
	"github.com/Grapple-2024/backend/internal/service/subscriptions"
	"github.com/Grapple-2024/backend/internal/service/techniques"
	"github.com/Grapple-2024/backend/pkg/aws/s3"
	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

const (
	envCognitoUserPoolID      = "COGNITO_USER_POOL_ID"
	envCognitoClientID        = "COGNITO_CLIENT_ID"
	envCognitoClientSecretID  = "COGNITO_CLIENT_SECRET"
	envMongoEndpoint          = "MONGO_ENDPOINT"
	envSendGridAPIKey         = "SENDGRID_API_KEY"
	envVideosBucketName       = "GYM_VIDEOS_BUCKET_NAME"
	envPublicAssetsBucketName = "PUBLIC_USER_ASSETS_BUCKET_NAME"
	envAWSRegion              = "AWS_REGION"
	envStripeAPIKey           = "STRIPE_API_KEY"
	envMapBoxAPIKey           = "MAPBOX_API_KEY"
)

func main() {
	mongoEndpoint, ok := os.LookupEnv(envMongoEndpoint)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envMongoEndpoint)
	}
	sendGridAPIKey, ok := os.LookupEnv(envSendGridAPIKey)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envSendGridAPIKey)
	}
	cognitoClientID, ok := os.LookupEnv(envCognitoClientID)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envCognitoClientID)
	}
	cognitoClientSecret, ok := os.LookupEnv(envCognitoClientSecretID)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envCognitoClientSecretID)
	}
	cognitoUserPoolID, ok := os.LookupEnv(envCognitoUserPoolID)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envCognitoUserPoolID)
	}
	gymVideosBucketName, ok := os.LookupEnv(envVideosBucketName)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envVideosBucketName)
	}
	publicAssetsBucketName, ok := os.LookupEnv(envPublicAssetsBucketName)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envPublicAssetsBucketName)
	}
	awsRegion, ok := os.LookupEnv(envAWSRegion)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envAWSRegion)
	}
	mapboxAPIKey, ok := os.LookupEnv(envMapBoxAPIKey)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", envMapBoxAPIKey)
	}

	sendGridClient := sendgrid.NewSendClient(sendGridAPIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	mongoClient, err := mongo.New(ctx, mongoEndpoint)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to connect to mongo endpoint: %q", mongoEndpoint)
	}
	defer func() {
		if err = mongoClient.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to disconnect from mongo")
		}
	}()

	cognitoClient, err := cognito.NewClient(
		awsRegion,
		cognito.WithUserPool(cognitoUserPoolID),
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create cognito client")
	}

	mapbox, err := mapbox.NewService(ctx, mapboxAPIKey)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize MapBox Service")
	}

	profiles, err := profiles.NewService(ctx, mongoClient, publicAssetsBucketName, awsRegion, cognitoUserPoolID, cognitoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Profiles Service")
	}

	rbac, err := rbac.New(profiles, cognitoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize RBAC Service")
	}

	s3Client, err := s3.New(awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize S3 Client")
	}

	s3, err := s3_service.NewService(ctx, s3Client, gymVideosBucketName)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize S3 Service")
	}
	techniques, err := techniques.NewService(ctx, mongoClient, gymVideosBucketName, awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Techniques Service")
	}
	gyms, err := gyms.NewService(ctx, publicAssetsBucketName, awsRegion, mongoClient, rbac, cognitoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gyms Service")
	}
	announcements, err := announcements.NewService(ctx, mongoClient, sendGridClient, profiles, rbac)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Announcements Service")
	}
	requests, err := gym_requests.NewService(ctx, mongoClient, sendGridClient, rbac)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Requests Service")
	}
	series, err := gym_series.NewService(ctx, mongoClient, s3Client, rbac, gymVideosBucketName, publicAssetsBucketName)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Series Service")
	}
	search, err := search.NewService(ctx, mongoClient, rbac)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Search Service")
	}
	subscriptions, err := subscriptions.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Subscriptions Service")
	}
	emails, err := email.NewService(ctx, mongoClient, sendGridClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Emails Service")
	}

	lambdas := map[string]lambda.Lambda{
		"profiles":      profiles,
		"gyms":          gyms,
		"announcements": announcements,
		"techniques":    techniques,
		"gym-requests":  requests,
		"gym-series":    series,
		"search":        search,
		"mapbox":        mapbox,
		"subscriptions": subscriptions,
		"s3":            s3,
		"emails":        emails,
	}

	router := lambda.NewRouter(lambdas)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Info().Msgf("Starting server on port %s", port)
	if err := http.ListenAndServe(":"+port, router); err != nil {
		log.Fatal().Err(err).Msg("server failed")
	}
}
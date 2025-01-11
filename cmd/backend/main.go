package main

import (
	"context"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/service/announcements"
	"github.com/Grapple-2024/backend/internal/service/gym_requests"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/mapbox"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/internal/service/search"
	"github.com/Grapple-2024/backend/internal/service/techniques"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

const (
	region = "us-west-1"

	// env variable keys
	EnvCognitoClientID        = "COGNITO_CLIENT_ID"
	EnvCognitoClientSecretID  = "COGNITO_CLIENT_SECRET"
	EnvMongoEndpoint          = "MONGO_ENDPOINT"
	EnvSendGridAPIKey         = "SENDGRID_API_KEY"
	EnvVideosBucketName       = "GYM_VIDEOS_BUCKET_NAME"
	EnvPublicAssetsBucketName = "PUBLIC_USER_ASSETS_BUCKET_NAME"
	EnvAWSRegion              = "AWS_REGION"
	EnvStripeAPIKey           = "STRIPE_API_KEY"
	EnvMapBoxAPIKey           = "MAPBOX_API_KEY"
)

func main() {
	// read all environment variables
	mongoEndpoint, ok := os.LookupEnv(EnvMongoEndpoint)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvMongoEndpoint)
	}
	sendGridAPIKey, ok := os.LookupEnv(EnvSendGridAPIKey)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvSendGridAPIKey)
	}
	cognitoClientID, ok := os.LookupEnv(EnvCognitoClientID)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvCognitoClientID)
	}
	cognitoClientSecret, ok := os.LookupEnv(EnvCognitoClientSecretID)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvCognitoClientSecretID)
	}
	gymVideosBucketName, ok := os.LookupEnv(EnvVideosBucketName)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvCognitoClientSecretID)
	}
	publicAssetsBucketName, ok := os.LookupEnv(EnvPublicAssetsBucketName)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvPublicAssetsBucketName)
	}
	awsRegion, ok := os.LookupEnv(EnvAWSRegion)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvAWSRegion)
	}
	mapboxAPIKey, ok := os.LookupEnv(EnvMapBoxAPIKey)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvMapBoxAPIKey)
	}
	log.Debug().Msgf("AWS Region: %v", awsRegion)
	log.Debug().Msgf("connected to mongo server: %s", mongoEndpoint)

	// Create sendgrid client
	sendGridClient := sendgrid.NewSendClient(sendGridAPIKey)

	// Create mongo client
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

	// Create services for each api controller/handler
	mapbox, err := mapbox.NewService(ctx, mapboxAPIKey)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize MapBox Service")
	}
	gyms, err := gyms.NewService(ctx, publicAssetsBucketName, region, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gyms Service")
	}
	search, err := search.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Search Service")
	}
	profiles, err := profiles.NewService(ctx, mongoClient, publicAssetsBucketName, awsRegion, cognitoClientID, cognitoClientSecret)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Profiles Service")
	}
	announcements, err := announcements.NewService(ctx, mongoClient, sendGridClient, profiles)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Announcements Service")
	}
	techniques, err := techniques.NewService(ctx, mongoClient, gymVideosBucketName, awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Techniques Service")
	}

	requests, err := gym_requests.NewService(ctx, mongoClient, sendGridClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Requests Service")
	}

	series, err := gym_series.NewService(ctx, mongoClient, gymVideosBucketName, publicAssetsBucketName, awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Series Service")
	}

	log.Info().Msgf("Grapple API URL: %v", os.Getenv("API_URL"))

	lambdas := map[string]lambda_v2.Lambda{
		// v2 endpoints are using mongodb
		"profiles":      profiles,
		"gyms":          gyms,
		"announcements": announcements,
		"techniques":    techniques,
		"gym-requests":  requests,
		"gym-series":    series,
		"search":        search,
		"mapbox":        mapbox,
	}

	router := lambda_v2.NewRouter(lambdas)
	lambda.Start(router)
}

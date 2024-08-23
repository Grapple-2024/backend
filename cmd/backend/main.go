package main

import (
	"context"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/service/announcements"
	"github.com/Grapple-2024/backend/internal/service/gym_requests"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/internal/service/techniques"
	"github.com/Grapple-2024/backend/pkg/handlers"
	lambdaext "github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

const (
	region = "us-west-1"

	// env variable keys
	EnvCognitoClientID        = "COGNITO_CLIENT_ID"
	EnvCognitoClientSecretID  = "COGNITO_CLIENT_SECRET"
	EnvDynamoEndpoint         = "DYNAMODB_ENDPOINT"
	EnvMongoEndpoint          = "MONGO_ENDPOINT"
	EnvSendGridAPIKey         = "SENDGRID_API_KEY"
	EnvVideosBucketName       = "GYM_VIDEOS_BUCKET_NAME"
	EnvPublicAssetsBucketName = "PUBLIC_ASSETS_BUCKET_NAME"
	EnvAWSRegion              = "AWS_REGION"
)

func main() {
	// read all environment variables
	mongoEndpoint, ok := os.LookupEnv(EnvMongoEndpoint)
	if !ok {

		log.Fatal().Msgf("missing required env var: %s", EnvMongoEndpoint)
	}
	dynamoEndpoint, ok := os.LookupEnv(EnvDynamoEndpoint)
	if !ok {
		log.Fatal().Msgf("missing required env var: %s", EnvDynamoEndpoint)
	}
	// sendGridAPIKey, ok := os.LookupEnv(EnvSendGridAPIKey)
	// if !ok {
	// 	log.Fatal().Msgf("missing required env var: %s", EnvSendGridAPIKey)
	// }
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
	log.Debug().Msgf("AWS Region: %v", awsRegion)
	log.Debug().Msgf("connected to dynamodb server: %s", dynamoEndpoint)
	log.Debug().Msgf("connected to mongo server: %s", mongoEndpoint)

	// Create mongo client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
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

	// Create V1 Handlers
	handlerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s3h, err := handlers.NewS3Handler(handlerCtx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	ch, err := handlers.NewCognitoHandler(handlerCtx, dynamoEndpoint, cognitoClientID, cognitoClientSecret)
	if err != err {
		panic(err)
	}

	eh, err := handlers.NewEmailHandler(handlerCtx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	uah, err := handlers.NewUserAssetHandler(handlerCtx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	// Create V2 Handlers (Mongo DB)
	gyms, err := gyms.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gyms Service")
	}
	announcements, err := announcements.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Announcements Service")
	}
	techniques, err := techniques.NewService(ctx, mongoClient, gymVideosBucketName, awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Techniques Service")
	}
	profiles, err := profiles.NewService(ctx, mongoClient, publicAssetsBucketName, awsRegion, cognitoClientID, cognitoClientSecret)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Profiles Service")
	}

	requests, err := gym_requests.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Requests Service")
	}

	series, err := gym_series.NewService(ctx, mongoClient, gymVideosBucketName, awsRegion)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize Gym Series Service")
	}

	log.Info().Msgf("Grapple API URL: %v", os.Getenv("API_URL"))

	lambdas := map[string]lambdaext.Lambda{
		"s3-presign-url": s3h,
		"cognito":        ch,
		"emails":         eh,
		"user-assets":    uah,

		// v2 endpoints are using mongodb
		"profiles":      profiles,
		"gyms":          gyms,
		"announcements": announcements,
		"techniques":    techniques,
		"gym-requests":  requests,
		"gym-series":    series,
	}

	router := lambdaext.NewRouter(lambdas)
	lambda.Start(router)
}

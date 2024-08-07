package main

import (
	"context"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/service/announcements"
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
)

func main() {
	// Create mongo client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	mongoURL := os.Getenv("MONGO_ENDPOINT")
	mongoClient, err := mongo.New(ctx, mongoURL)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to connect to mongo endpoint: %q", mongoURL)
	}

	defer func() {
		if err = mongoClient.Disconnect(ctx); err != nil {
			log.Fatal().Err(err).Msg("failed to disconnect from mongo")
		}
	}()

	// Create V1 Handlers
	handlerCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sendGridAPIKey := os.Getenv("SENDGRID_API_KEY")
	cognitoClientID := os.Getenv("COGNITO_CLIENT_ID")
	cognitoClientSecret := os.Getenv("COGNITO_CLIENT_SECRET")
	dynamoEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	log.Info().Msgf("Dynamo endpoint: %s", dynamoEndpoint)

	// Create handlers

	grh, err := handlers.NewGymRequestHandler(handlerCtx, dynamoEndpoint, sendGridAPIKey)
	if err != err {
		panic(err)
	}

	gvh, err := handlers.NewGymVideoHandler(handlerCtx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	gvsh, err := handlers.NewGymVideoSeriesHandler(handlerCtx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

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
		log.Fatal().Err(err).Msgf("failed to initialize new Gym Service")
	}
	announcements, err := announcements.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize new Gym Service")
	}
	techniques, err := techniques.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize new Gym Service")
	}
	profiles, err := profiles.NewService(ctx, mongoClient)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to initialize new Gym Service")
	}

	log.Info().Msgf("Base API URL: %v", os.Getenv("API_URL"))

	lambdas := map[string]lambdaext.Lambda{
		"gym-requests":     grh,
		"gym-videos":       gvh,
		"gym-video-series": gvsh,
		"s3-presign-url":   s3h,
		"cognito":          ch,
		"emails":           eh,
		"user-assets":      uah,

		// v2 endpoints are using mongodb
		"profiles":      profiles,
		"gyms":          gyms,
		"announcements": announcements,
		"techniques":    techniques,
	}

	router := lambdaext.NewRouter(lambdas)
	lambda.Start(router)
}

package main

import (
	"context"
	"os"
	"time"

	"github.com/Grapple-2024/backend/pkg/handlers"
	lambdaext "github.com/Grapple-2024/backend/pkg/lambda"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	sendGridAPIKey := os.Getenv("SENDGRID_API_KEY")
	cognitoClientID := os.Getenv("COGNITO_CLIENT_ID")
	cognitoClientSecret := os.Getenv("COGNITO_CLIENT_SECRET")
	dynamoEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	log.Info().Msgf("Dynamo endpoint: %s", dynamoEndpoint)

	// Create handlers
	gh, err := handlers.NewGymHandler(ctx, dynamoEndpoint, cognitoClientID, cognitoClientSecret)
	if err != err {
		panic(err)
	}

	grh, err := handlers.NewGymRequestHandler(ctx, dynamoEndpoint, sendGridAPIKey)
	if err != err {
		panic(err)
	}

	gas, err := handlers.NewGymAnnouncementHandler(ctx, dynamoEndpoint, sendGridAPIKey, cognitoClientID, cognitoClientSecret)
	if err != err {
		panic(err)
	}

	gvh, err := handlers.NewGymVideoHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	gvsh, err := handlers.NewGymVideoSeriesHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	s3h, err := handlers.NewS3Handler(ctx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	ch, err := handlers.NewCognitoHandler(ctx, dynamoEndpoint, cognitoClientID, cognitoClientSecret)
	if err != err {
		panic(err)
	}

	eh, err := handlers.NewEmailHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	uah, err := handlers.NewUserAssetHandler(ctx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	uph, err := handlers.NewUserProfileHandler(ctx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	lambdas := map[string]lambdaext.Lambda{
		"gyms":              gh,
		"gym-requests":      grh,
		"gym-announcements": gas,
		"gym-videos":        gvh,
		"gym-video-series":  gvsh,
		"s3-presign-url":    s3h,
		"cognito":           ch,
		"emails":            eh,
		"user-assets":       uah,
		"user-profiles":     uph,
	}

	router := lambdaext.NewRouter(lambdas)
	lambda.Start(router)
}

package main

import (
	"context"
	"os"
	"time"

	"github.com/Grapple-2024/backend/handlers"
	lambdaext "github.com/Grapple-2024/backend/lambda"
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

	dynamoEndpoint := os.Getenv("DYNAMODB_ENDPOINT")
	gh, err := handlers.NewGymHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	grh, err := handlers.NewGymRequestHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	gas, err := handlers.NewGymAnnouncementHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	gvs, err := handlers.NewGymVideoHandler(ctx, dynamoEndpoint)
	if err != err {
		panic(err)
	}

	s3h, err := handlers.NewS3Handler(ctx, dynamoEndpoint, region)
	if err != err {
		panic(err)
	}

	lambdas := map[string]lambdaext.Lambda{
		"gyms":              gh,
		"gym-requests":      grh,
		"gym-announcements": gas,
		"gym-videos":        gvs,
		"s3-presign-url":    s3h,
	}

	lambda.Start(
		lambdaext.NewRouter(lambdas),
	)
}

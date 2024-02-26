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

	lambdas := map[string]lambdaext.Lambda{
		"gyms":         gh,
		"gym-requests": grh,
	}

	lambda.Start(
		lambdaext.NewRouter(lambdas),
	)
}

package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/rs/zerolog/log"
)

const (
	region = "us-west-1"

	// env variable keys
	envCognitoUserPoolID     = "COGNITO_USER_POOL_ID"
	envCognitoClientID       = "COGNITO_CLIENT_ID"
	envCognitoClientSecretID = "COGNITO_CLIENT_SECRET"
)

func main() {
	// Cognito Env Vars
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
	mongoURL, ok := os.LookupEnv("MONGO_ENDPOINT")
	if !ok {
		log.Fatalf("required env var not set: MONGO_ENDPOINT")
	}

	// Create mongo context and client
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mc, err := mongo.New(ctx, mongoURL)
	if err != nil {
		log.Fatalf("failed to create mongo client: %v", err)
	}

	defer func() {
		if err = mc.Disconnect(ctx); err != nil {
			log.Fatalf("failed to disconnect from mongo: %v", err)
		}
	}()

	/** Create Cognito Client ***/
	cognitoClient, err := cognito.NewClient(
		region,
		cognito.WithUserPool(cognitoUserPoolID),
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create cognito client")
	}

	gymsCollection := mc.Client.Database("grapple").Collection("gyms")

}

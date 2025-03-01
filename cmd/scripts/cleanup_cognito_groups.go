package main

import (
	"context"
	"os"

	"github.com/Grapple-2024/backend/pkg/cognito"
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

	/** Create Cognito Client ***/
	cc, err := cognito.NewClient(
		region,
		cognito.WithUserPool(cognitoUserPoolID),
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create cognito client")
	}

	username := "8865a280-0ba2-4455-be19-a93d5813f269"
	out, err := cc.ListGroupsForUser(context.Background(), username)
	if err != nil {
		log.Fatal().Err(err).Msgf("failed to list groups for cognito user %s", username)
	}

	log.Info().Msgf("output: %+v", out)

}

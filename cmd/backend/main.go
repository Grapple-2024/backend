package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Grapple-2024/backend/cognito"
	"github.com/Grapple-2024/backend/handlers/auth"
	"github.com/Grapple-2024/backend/handlers/gym"

	"github.com/Grapple-2024/backend/mongo"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func init() {
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
}

func main() {
	// create mongo client
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	mongoUser := "root"
	mongoPass := "local"
	mongoHost := "localhost"
	mongoPort := 27017
	mongoURI := fmt.Sprintf("mongodb://%s:%s@%s:%d/", mongoUser, mongoPass, mongoHost, mongoPort)
	mongoClient, err := mongo.NewClient(ctx, mongoURI, "grapple")
	if err != nil {
		log.Fatal().Err(err).Msgf("error establishing connection to mongo")
	}
	log.Info().Msgf("mongoClient: %v", mongoClient)

	// Create Cognito Client
	clientID := "19e853hg83ddqvbq160fh6j8i6"
	clientSecret := "1fteublmrhdehckmva7u6lqf96hgcl4tgg4dt32iqvd7f00nru9"
	region := "us-west-1"
	cClient, err := cognito.NewClient(region, cognito.WithClientID(clientID), cognito.WithClientSecret(clientSecret))
	if err != nil {
		log.Fatal().Err(err).Msgf("error creating cognito client")
	}

	// Create handler(s)
	authHandler := auth.Handler{
		CognitoClient: cClient,
	}
	gymHandler := gym.Handler{
		MongoClient: mongoClient,
	}

	// Register routes and start web server
	r := gin.Default()
	r.POST("/login", authHandler.Login)
	r.POST("/register", authHandler.Register)

	// Students
	r.GET("/gyms/:id", gymHandler.GetGym)
	r.POST("/gyms", gymHandler.CreateGym)
	r.GET("/gyms", gymHandler.GetGyms)

	r.Run()
}

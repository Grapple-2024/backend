package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

var mongoClient *mongo.Client

func main() {
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
		if err = mongoClient.Disconnect(ctx); err != nil {
			log.Fatalf("failed to disconnect from mongo: %v", err)
		}
	}()

	mongoClient = mc

	lambda.Start(Handler)
}

func Handler(ctx context.Context, e events.CognitoEventUserPoolsPostConfirmation) (events.CognitoEventUserPoolsPostConfirmation, error) {
	log.Printf("Event: %+v", e)
	c := mongoClient.Database("grapple").Collection("profiles")
	attrs := e.Request.UserAttributes

	profile := dao.Profile{
		CognitoID:               attrs["sub"],
		Email:                   attrs["email"],
		FirstName:               attrs["given_name"],
		LastName:                attrs["family_name"],
		PhoneNumber:             attrs["phone_number"],
		NotifyOnRequestAccepted: true,
		Gyms:                    []dao.GymAssociation{},
		CreatedAt:               time.Now().Local().UTC(),
		UpdatedAt:               time.Now().Local().UTC(),
	}

	var result dao.Profile
	if err := mongo.Insert(ctx, c, profile, &result); err != nil {
		return e, fmt.Errorf("failed to insert profile: %v", err)
	}

	return e, nil
}

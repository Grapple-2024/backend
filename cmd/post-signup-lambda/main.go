package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/mitchellh/mapstructure"
)

var mongoClient *mongo.Client

// Handler function that will be invoked by AWS Lambda

type Event struct {
	Request Request `json:"request"`

	// TriggerSource should always be Post User Signup Confirmation Cognito Event
	TriggerSource string `json:"triggerSource"`
}

type Request struct {
	UserAttributes UserAttributes `json:"userAttributes"`
}

type UserAttributes struct {
	Email       string `json:"email"`
	FamilyName  string `json:"family_name" mapstructure:"family_name"`
	GivenName   string `json:"given_name" mapstructure:"given_name"`
	PhoneNumber string `json:"phone_number" mapstructure:"phone_number"`
	Sub         string `json:"sub"`
}

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
	// pc := mongoClient.Database("grapple").Collection("profiles")
	// pc.UpdateOne(ctx)

	lambda.Start(Handler)
}

func Handler(ctx context.Context, req map[string]any) (string, error) {
	event := Event{}
	if err := mapstructure.Decode(req, &event); err != nil {
		return "", fmt.Errorf("failed to decode event into struct: %v", err)
	}

	log.Printf("Event: %+v", event)
	c := mongoClient.Database("grapple").
		Collection("profiles")
	attrs := event.Request.UserAttributes

	profile := profiles.Profile{
		CognitoID:   attrs.Sub,
		Email:       attrs.Email,
		FirstName:   attrs.GivenName,
		LastName:    attrs.FamilyName,
		PhoneNumber: attrs.PhoneNumber,
		Gyms:        []profiles.GymAssociation{},
		CreatedAt:   time.Now().Local().UTC(),
		UpdatedAt:   time.Now().Local().UTC(),
	}

	var result profiles.Profile
	if err := mongo.Insert(ctx, c, profile, &result); err != nil {
		return "", err
	}

	log.Printf("Created new user profile: %+v", result)
	json, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	return string(json), nil
}

package subscriptions

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

type Service struct {
	Client         *mongoext.Client
	Collection     *mongo.Collection
	GymsCollection *mongo.Collection
}

func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("subscriptions")
	gymsCollection := mc.Database("grapple").Collection("gyms")

	svc := &Service{
		Client:         mc,
		Collection:     c,
		GymsCollection: gymsCollection,
	}

	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) ensureIndices(ctx context.Context) error {
	// Create a compound index on gym_id and profile_id
	_, err := s.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{
			{Key: "gym_id", Value: 1},
			{Key: "profile_id", Value: 1},
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var subscription dao.Subscription
	if err := json.Unmarshal([]byte(req.Body), &subscription); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	err := s.createSubscription(ctx, subscription)

	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) createSubscription(ctx context.Context, subscription dao.Subscription) error {
	subscription = dao.Subscription{
		GymId:                subscription.GymId,
		StripeCustomerId:     "",
		StripeSubscriptionId: "",
		SubscriptionStatus:   "active",
		PriceId:              "",
		CurrentPeriodEnd:     time.Now().Local().UTC(),
		CancelAtPeriodEnd:    false,
		CreatedAt:            time.Now().Local().UTC(),
		UpdatedAt:            time.Now().Local().UTC(),
	}

	if err := mongoext.Insert(ctx, s.Collection, subscription, &subscription); err != nil {
		return err
	}

	return nil
}

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var subscription dao.Subscription
	if err := json.Unmarshal([]byte(req.Body), &subscription); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	err := s.deleteSubscription(ctx, subscription)

	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

/**
* This function is used to get all the subscriptions for a user, it joins to the gym since
* the settings page will need to show the gym details for each subscription
**/
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters for filtering
	queryParams := req.QueryStringParameters
	matchStage := bson.M{}

	_, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("permission denied: %v", err))
	}

	// Add profile_id filter if provided
	if gymId, ok := queryParams["gym_id"]; ok && gymId != "" {
		// Convert the gym_id to an ObjectID
		gymIdObj, err := bson.ObjectIDFromHex(gymId)

		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid gym_id format: %v", err))
		}
		matchStage["gym_id"] = gymIdObj
	}

	// Set a reasonable default limit if not specified
	if limit <= 0 {
		limit = 100
	}

	// Build the aggregation pipeline
	pipeline := []bson.M{
		// Match stage (filtering)
		{
			"$match": matchStage,
		},
		// Lookup (join) with gyms collection
		{
			"$lookup": bson.M{
				"from":         "gyms",   // The collection to join with
				"localField":   "gym_id", // Field from subscriptions collection
				"foreignField": "_id",    // Field from gyms collection
				"as":           "gym",    // Output array field
			},
		},
		// Unwind the gym_details array to get a single gym object
		{
			"$unwind": bson.M{
				"path":                       "$gym",
				"preserveNullAndEmptyArrays": true, // Keep subscriptions even if no matching gym
			},
		},
		// Sort by creation date
		{
			"$sort": bson.M{
				"createdat": -1, // Newest first
			},
		},
		// Limit results
		{
			"$limit": limit,
		},
	}
	// Execute the aggregation
	cursor, err := s.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error executing aggregation: %v", err))
	}
	defer cursor.Close(ctx)

	// Decode the results into our combined structure
	var subscriptionsWithGyms []dao.Subscription
	if err = cursor.All(ctx, &subscriptionsWithGyms); err != nil {
		return lambda.ServerError(fmt.Errorf("error parsing aggregation results: %v", err))
	}

	// Return empty array instead of null if no subscriptions found
	if subscriptionsWithGyms == nil {
		subscriptionsWithGyms = []dao.Subscription{}
	}

	// Marshal response
	response, err := json.Marshal(subscriptionsWithGyms)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error serializing response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(response), nil), nil
}

func (s *Service) deleteSubscription(ctx context.Context, subscription dao.Subscription) error {
	filter := bson.M{"gym_id": subscription.GymId}

	_, err := s.Collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}

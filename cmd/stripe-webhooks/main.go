package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/stripe/stripe-go/v81"
	"github.com/stripe/stripe-go/v81/subscription"
	"github.com/stripe/stripe-go/v81/webhook"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
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

	stripeKey := os.Getenv("STRIPE_SECRET_KEY")

	if stripeKey == "" {
		log.Fatal("missing required env var")
	}

	stripe.Key = stripeKey

	lambda.Start(Handler)
}

// Lambda function to handle Stripe webhooks
func Handler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Verify Stripe signature for security
	endpointSecret := os.Getenv("STRIPE_WEBHOOK_SECRET")

	// Case-insensitive header lookup for Stripe-Signature
	signatureHeader := ""
	for key, value := range request.Headers {
		if strings.ToLower(key) == "stripe-signature" {
			signatureHeader = value
			break
		}
	}

	// Check if signature header exists
	if signatureHeader == "" {
		log.Printf("Error: Stripe-Signature header is missing")
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       "Missing Stripe-Signature header",
		}, nil
	}

	// Verify the webhook
	event, err := webhook.ConstructEvent([]byte(request.Body), signatureHeader, endpointSecret)
	if err != nil {
		log.Printf("Error verifying webhook signature: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 400,
			Body:       fmt.Sprintf("Invalid signature: %v", err),
		}, nil
	}

	// Log event
	log.Printf("Received event: %s", event.Type)

	// Handle different event types
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Error parsing session",
			}, nil
		}

		// Process the successful checkout
		err = handleCheckoutSessionCompleted(ctx, session)
		if err != nil {
			log.Printf("Error handling checkout session: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Internal server error",
			}, nil
		}

	case "customer.subscription.created", "customer.subscription.updated":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Error parsing subscription",
			}, nil
		}

		// Update the subscription in your database
		err = updateSubscriptionStatus(ctx, subscription)
		if err != nil {
			log.Printf("Error updating subscription: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Internal server error",
			}, nil
		}

	case "customer.subscription.deleted":
		var subscription stripe.Subscription
		err := json.Unmarshal(event.Data.Raw, &subscription)
		if err != nil {
			return events.APIGatewayProxyResponse{
				StatusCode: 400,
				Body:       "Error parsing subscription",
			}, nil
		}

		// Handle the subscription cancellation
		err = handleSubscriptionCancelled(ctx, subscription)
		if err != nil {
			log.Printf("Error handling subscription cancellation: %v", err)
			return events.APIGatewayProxyResponse{
				StatusCode: 500,
				Body:       "Internal server error",
			}, nil
		}
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       "Webhook processed successfully",
	}, nil
}

// Function to handle a completed checkout session
func handleCheckoutSessionCompleted(ctx context.Context, session stripe.CheckoutSession) error {
	// Get the customer ID
	customerID := session.Customer.ID

	// Get the subscription ID (if it was a subscription checkout)
	if session.Subscription == nil {
		return fmt.Errorf("no subscription found in checkout session")
	}

	subscriptionID := session.Subscription.ID

	// Retrieve full subscription details to get price ID and other metadata
	subscriptionObj, err := subscription.Get(subscriptionID, nil)
	if err != nil {
		return fmt.Errorf("failed to get subscription: %v", err)
	}

	// Get metadata from the checkout session which might include user ID or gym ID
	// This assumes you set these as metadata when creating the checkout session
	userID := session.Metadata["user_id"]

	if userID == "" {
		return fmt.Errorf("user_id not found in session metadata")
	}

	priceID := subscriptionObj.Items.Data[0].Price.ID

	gymID := session.Metadata["gym_id"]

	if gymID == "" {
		return fmt.Errorf("gym_id not found in session metadata")
	}

	newGymId, err := bson.ObjectIDFromHex(gymID)

	if err != nil {
		return fmt.Errorf("invalid gym_id format: %v", err)
	}

	// Create the subscription record in your database
	c := mongoClient.Database("grapple").Collection("subscriptions")
	filter := bson.M{"profile_id": userID}

	// Prepare the update with $set and $setOnInsert
	update := bson.M{
		"$set": bson.M{
			"stripe_subscription_id": subscriptionID,
			"subscription_status":    "active",
			"stripe_product_id":      subscriptionObj.Items.Data[0].Price.Product.ID,
			"gym_id":                 newGymId,
			"stripe_customer_id":     customerID,
			"price_id":               priceID,
			"current_period_end":     time.Unix(subscriptionObj.CurrentPeriodEnd, 0),
			"cancel_at_period_end":   subscriptionObj.CancelAtPeriodEnd,
			"updated_at":             time.Now().UTC(),
		},
		"$setOnInsert": bson.M{
			"profile_id": userID,
			"created_at": time.Now().UTC(),
		},
	}

	opts := options.UpdateOne().SetUpsert(true)
	result, err := c.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		return fmt.Errorf("failed to upsert subscription: %v", err)
	}

	// Log whether it was an insert or update
	if result.UpsertedCount > 0 {
		log.Printf("Created new subscription with ID: %v", result.UpsertedID)
	} else {
		log.Printf("Updated existing subscription for customer: %s", customerID)
	}

	gymCollection := mongoClient.Database("grapple").Collection("gyms")

	// Update the gym with the new subscription
	gymFilter := bson.M{"_id": newGymId}

	gymUpdate := bson.M{
		"$set": bson.M{
			"is_subscribed": true,
		},
	}

	_, err = gymCollection.UpdateOne(ctx, gymFilter, gymUpdate)

	if err != nil {
		return fmt.Errorf("failed to update gym: %v", err)
	}

	return nil
}

// Function to update subscription status based on webhook events
func updateSubscriptionStatus(ctx context.Context, subscription stripe.Subscription) error {
	customerID := subscription.Customer.ID
	subscriptionID := subscription.ID
	status := subscription.Status
	cancelAtPeriodEnd := subscription.CancelAtPeriodEnd

	// Map Stripe status to your application status
	var subscriptionStatus string
	switch status {
	case "active":
		if cancelAtPeriodEnd {
			// Subscription is active but scheduled to cancel at period end
			subscriptionStatus = "active_cancelling"
			log.Printf("Subscription %s is active but scheduled to be cancelled at period end", subscriptionID)
		} else {
			subscriptionStatus = "active"
		}
	case "past_due":
		subscriptionStatus = "past_due"
	case "unpaid":
		subscriptionStatus = "unpaid"
	case "canceled":
		subscriptionStatus = "canceled"
	case "trialing":
		subscriptionStatus = "trial"
	default:
		subscriptionStatus = "inactive"
	}

	c := mongoClient.Database("grapple").Collection("subscriptions")

	// Include additional fields if cancellation at period end is scheduled
	updateFields := bson.M{
		"stripe_subscription_id": subscriptionID,
		"stripe_customer_id":     customerID,
		"subscription_status":    subscriptionStatus,
		"current_period_end":     time.Unix(subscription.CurrentPeriodEnd, 0),
		"cancel_at_period_end":   cancelAtPeriodEnd,
		"updated_at":             time.Now().UTC(),
	}

	// Add cancellation date if it's scheduled to cancel
	if cancelAtPeriodEnd {
		// Store when the cancellation was requested
		updateFields["cancellation_requested_at"] = time.Now().UTC()
	}

	filter := bson.M{"stripe_subscription_id": subscriptionID}
	update := bson.M{
		"$set": updateFields,
	}

	result, err := c.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update subscription status: %v", err)
	}

	if result.MatchedCount == 0 {
		// Try with customer ID if subscription ID didn't match
		filter = bson.M{"stripe_customer_id": customerID}
		result, err = c.UpdateOne(ctx, filter, update)
		if err != nil {
			return fmt.Errorf("failed to update subscription status with customer ID: %v", err)
		}

		if result.MatchedCount == 0 {
			log.Printf("Warning: No subscription found for customer: %s with subscription ID: %s", customerID, subscriptionID)
		}
	}

	return nil
}

// Function to handle a cancelled subscription
func handleSubscriptionCancelled(ctx context.Context, subscription stripe.Subscription) error {
	subscriptionID := subscription.ID

	// Get the cancellation details
	cancelledAt := time.Unix(subscription.CanceledAt, 0)

	c := mongoClient.Database("grapple").Collection("subscriptions")

	// First find the subscription to get the gym_id
	var subRecord struct {
		GymID bson.ObjectID `bson:"gym_id"`
	}

	findFilter := bson.M{"stripe_subscription_id": subscriptionID}
	err := c.FindOne(ctx, findFilter).Decode(&subRecord)
	if err != nil {
		return fmt.Errorf("failed to find subscription record: %v", err)
	}

	// Update subscription in database
	updateFilter := bson.M{"stripe_subscription_id": subscriptionID}
	update := bson.M{
		"$set": bson.M{
			"subscription_status":  "cancelled",
			"cancelled_at":         cancelledAt,
			"current_period_end":   time.Unix(subscription.CurrentPeriodEnd, 0),
			"cancel_at_period_end": true,
			"updated_at":           time.Now().UTC(),
		},
	}

	result, err := c.UpdateOne(ctx, updateFilter, update)
	if err != nil {
		return fmt.Errorf("failed to update cancelled subscription: %v", err)
	}

	if result.MatchedCount == 0 {
		log.Printf("Warning: No subscription found with ID: %s", subscriptionID)
		return nil
	} else {
		log.Printf("Successfully marked subscription %s as cancelled", subscriptionID)
	}

	// Update the gym's subscription status
	gymCollection := mongoClient.Database("grapple").Collection("gyms")

	gymFilter := bson.M{"_id": subRecord.GymID}
	gymUpdate := bson.M{
		"$set": bson.M{
			"is_subscribed": false,
		},
	}

	gymResult, err := gymCollection.UpdateOne(ctx, gymFilter, gymUpdate)
	if err != nil {
		return fmt.Errorf("failed to update gym subscription status: %v", err)
	}

	if gymResult.MatchedCount == 0 {
		log.Printf("Warning: No gym found with ID: %s", subRecord.GymID.Hex())
	} else {
		log.Printf("Successfully updated gym %s subscription status to false", subRecord.GymID.Hex())
	}

	return nil
}

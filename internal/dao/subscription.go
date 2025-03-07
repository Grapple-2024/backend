package dao

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Subscription represents the "Subscription" mongodb entity
type Subscription struct {
	ID                   bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	ProfileId            string        `json:"profile_id,omitempty" bson:"profile_id,omitempty"`
	GymId                bson.ObjectID `json:"gym_id,omitempty" bson:"gym_id,omitempty"`
	StripeCustomerId     string        `json:"stripe_customer_id" bson:"stripe_customer_id,omitempty"`
	StripeSubscriptionId string        `json:"stripe_subscription_id" bson:"stripe_subscription_id,omitempty"`
	SubscriptionStatus   string        `json:"subscription_status" bson:"subscription_status,omitempty"`
	StripeProductId      string        `json:"stripe_product_id" bson:"stripe_product_id,omitempty"`
	PriceId              string        `json:"price_id" bson:"price_id,omitempty"`
	CurrentPeriodEnd     time.Time     `json:"current_period_end" bson:"current_period_end,omitempty"`
	CancelAtPeriodEnd    bool          `json:"cancel_at_period_end" bson:"cancel_at_period_end,omitempty"`
	CancelledAt          time.Time     `json:"cancelled_at" bson:"cancelled_at,omitempty"`
	Gym                  *Gym          `json:"gym,omitempty" bson:"gym,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

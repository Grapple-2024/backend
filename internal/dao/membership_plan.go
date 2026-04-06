package dao

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// MembershipPlan represents a plan that a gym offers to its members.
// BillingType is "recurring" or "one_time".
// Interval is "monthly", "yearly", or "weekly" (only applies when BillingType == "recurring").
// Price is stored in cents (e.g. 10000 = $100.00).
// ClassLimit nil means unlimited classes.
type MembershipPlan struct {
	ID          bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID       bson.ObjectID `json:"gym_id" bson:"gym_id"`
	Name        string        `json:"name" bson:"name" validate:"required"`
	Description string        `json:"description" bson:"description"`
	BillingType string        `json:"billing_type" bson:"billing_type" validate:"required,oneof=recurring one_time"`
	Interval    string        `json:"interval" bson:"interval"`
	Price       int64         `json:"price" bson:"price" validate:"required,min=0"`
	Currency    string        `json:"currency" bson:"currency"`
	ClassLimit  *int          `json:"class_limit" bson:"class_limit"`
	IsActive    bool          `json:"is_active" bson:"is_active"`
	CreatedAt   time.Time     `json:"created_at" bson:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at" bson:"updated_at"`
}

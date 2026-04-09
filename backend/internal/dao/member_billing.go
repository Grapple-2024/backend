package dao

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// MemberBilling links a gym member to a membership plan.
// StripeCustomerID and StripeSubscriptionID are reserved for future Stripe Connect integration.
type MemberBilling struct {
	ID       bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID    bson.ObjectID `json:"gym_id" bson:"gym_id"`
	MemberID string        `json:"member_id" bson:"member_id" validate:"required"` // Clerk user ID
	PlanID   bson.ObjectID `json:"plan_id" bson:"plan_id"`

	// Denormalized for display (avoids join on listing)
	PlanName   string `json:"plan_name" bson:"plan_name"`
	MemberName string `json:"member_name" bson:"member_name"`

	Status          string    `json:"status" bson:"status"`           // "active" | "paused" | "cancelled"
	StartDate       time.Time `json:"start_date" bson:"start_date"`
	NextPaymentDate time.Time `json:"next_payment_date" bson:"next_payment_date"`

	// Stripe-ready: populated when Stripe Connect is implemented
	StripeCustomerID     string `json:"stripe_customer_id,omitempty" bson:"stripe_customer_id,omitempty"`
	StripeSubscriptionID string `json:"stripe_subscription_id,omitempty" bson:"stripe_subscription_id,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// PaymentRecord tracks a single billing cycle payment for a member.
// StripePaymentIntentID and StripeInvoiceID are reserved for future Stripe Connect integration.
type PaymentRecord struct {
	ID        bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID     bson.ObjectID `json:"gym_id" bson:"gym_id"`
	MemberID  string        `json:"member_id" bson:"member_id"`
	BillingID bson.ObjectID `json:"billing_id" bson:"billing_id"`

	// Denormalized for display
	PlanID     bson.ObjectID `json:"plan_id" bson:"plan_id"`
	PlanName   string        `json:"plan_name" bson:"plan_name"`
	MemberName string        `json:"member_name" bson:"member_name"`

	Amount   int64  `json:"amount" bson:"amount"`     // cents
	Currency string `json:"currency" bson:"currency"` // "usd"

	Status  string     `json:"status" bson:"status"`              // "unpaid" | "paid" | "overdue"
	DueDate time.Time  `json:"due_date" bson:"due_date"`
	PaidAt  *time.Time `json:"paid_at,omitempty" bson:"paid_at,omitempty"` // nil until paid

	Notes string `json:"notes,omitempty" bson:"notes,omitempty"`

	// Stripe-ready: populated when Stripe Connect is implemented
	StripePaymentIntentID string `json:"stripe_payment_intent_id,omitempty" bson:"stripe_payment_intent_id,omitempty"`
	StripeInvoiceID       string `json:"stripe_invoice_id,omitempty" bson:"stripe_invoice_id,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

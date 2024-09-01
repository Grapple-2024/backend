package gym_requests

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status - Custom enum type to hold value for a request's status
type Status int

const (
	RequestPending  string = "Pending"
	RequestAccepted string = "Accepted"
	RequestDenied   string = "Denied"
)

// GymRequest represents the GymRequest document structure in MongoDB.
type GymRequest struct {
	ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID primitive.ObjectID `json:"gym_id" bson:"gym_id,omitempty" validate:"required"`

	RequestorID    string `json:"requestor_id" bson:"requestor_id,omitempty" validate:"required"`
	RequestorEmail string `json:"requestor_email" bson:"requestor_email,omitempty" validate:"required"`
	FirstName      string `json:"first_name" bson:"first_name,omitempty" validate:"required"`
	LastName       string `json:"last_name" bson:"last_name,omitempty" validate:"required"`

	// The status of the gym request either "Accepted", "Pending", or "Rejected"
	Status string `json:"status" bson:"status,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

// isValidStatus returns true if a status string is one of the three possible enums. It will return false if not.
func isValidStatus(s string) bool {
	if s == RequestAccepted || s == RequestDenied || s == RequestPending {
		return true
	}
	return false
}

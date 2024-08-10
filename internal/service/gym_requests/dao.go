package gym_requests

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status - Custom enum type to hold value for a request's status
type Status int

const (
	RequestPending  string = "Pending"
	RequestApproved string = "Approved"
	RequestDenied   string = "Denied"
)

// GymRequest represents the GymRequest document structure in MongoDB.
type GymRequest struct {
	ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID primitive.ObjectID `json:"gym_id" bson:"gym_id,omitempty"`

	RequestorID    string `json:"requestor_id" bson:"requestor_id,omitempty"`
	RequestorEmail string `json:"requestor_email" bson:"requestor_email,omitempty"`
	FirstName      string `json:"first_name" bson:"first_name,omitempty"`
	LastName       string `json:"last_name" bson:"last_name,omitempty"`

	// The status of the gym request either "Approved", "Pending", or "Rejected"
	Status string `json:"status" bson:"status,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

// isValidStatus returns true if a status string is one of the three possible enums. It will return false if not.
func isValidStatus(s string) bool {
	if s == RequestApproved || s == RequestDenied || s == RequestPending {
		return true
	}
	return false
}

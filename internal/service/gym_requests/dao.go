package gym_requests

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Status - Custom enum type to hold value for a request's status
type Status int

const (
	RequestPending  Status = iota // 0
	RequestApproved               // 1
	RequestDenied                 // 2
)

// GymRequest represents the GymRequest document structure in MongoDB.
type GymRequest struct {
	ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID primitive.ObjectID `json:"gym_id" dynamodbav:"gym_id,omitempty"`

	RequestorID    string `json:"requestor_id" dynamodbav:"requestor_id"`
	RequestorEmail string `json:"requestor_email" dynamodbav:"requestor_email"`
	FirstName      string `json:"first_name" dynamodbav:"first_name,omitempty"`
	LastName       string `json:"last_name" dynamodbav:"last_name,omitempty"`

	// The status of the gym request
	// either "Approved", "Pending", or "Rejected"
	Status Status `json:"status" dynamodbav:"status,omitempty"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

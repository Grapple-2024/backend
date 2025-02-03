package gym_requests

import (
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Status - Custom enum type to hold value for a request's status
type Status int

const (
	RequestPending  string = "Pending"
	RequestAccepted string = "Accepted"
	RequestDenied   string = "Denied"

	VirtualMembership  string = "VIRTUAL"
	InPersonMembership string = "IN-PERSON"
)

// GymRequest represents the GymRequest document structure in MongoDB.
type GymRequest struct {
	ID      bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Profile *dao.Profile  `json:"profile,omitempty" bson:"-"`
	GymID   bson.ObjectID `json:"gym_id" bson:"gym_id,omitempty" validate:"required"`

	RequestorID    string `json:"requestor_id" bson:"requestor_id,omitempty" validate:"required"`
	RequestorEmail string `json:"requestor_email" bson:"requestor_email,omitempty" validate:"required"`
	FirstName      string `json:"first_name" bson:"first_name,omitempty" validate:"required"`
	LastName       string `json:"last_name" bson:"last_name,omitempty" validate:"required"`
	MembershipType string `json:"membership_type" bson:"membership_type,omitempty" validate:"required"`

	// The status of the gym request either "Accepted", "Pending", or "Rejected"
	Status string `json:"status" bson:"status,omitempty"`

	// The role being requested in the gym, ie "coach" or "student".
	Role string `json:"role" bson:"role,omitempty" validate:"required"`

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

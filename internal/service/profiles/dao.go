package profiles

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Role enum
var (
	OwnerRole   = "Owner"
	CoachRole   = "Coach"
	StudentRole = "Student"
)

// Profile represents the Profile mongodb entity
type Profile struct {
	ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`

	CognitoID string `json:"cognito_id,omitempty" bson:"cognito_id,omitempty" validate:"required"`

	// Personal Data is duplicated from Cognito via post-signup-lambda function
	Email       string `json:"email" bson:"email,omitempty" validate:"required"`
	FirstName   string `json:"first_name" bson:"first_name,omitempty" validate:"required"`
	LastName    string `json:"last_name" bson:"last_name,omitempty" validate:"required"`
	PhoneNumber string `json:"phone_number" bson:"phone_number,omitempty" validate:"required"`

	AvatarURL string `json:"avatar_url" bson:"avatar_url,omitempty"`
	// AvatarS3ObjectKey string `json:"avatar_s3_object_key" bson:"avatar_s3_object_key,omitempty"`
	NotifyOnRequestAccepted bool             `json:"notify_on_request_accepted" bson:"notify_on_request_accepted,omitempty"`
	Gyms                    []GymAssociation `json:"gyms,omit" bson:"gyms,omitempty"`

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

// GymAssociation represents a user's association to a gym.
// A GymAssociation can either be for a student or a coach.
type GymAssociation struct {
	GymID            primitive.ObjectID `json:"gym_id" bson:"gym_id,omitempty"`
	Email            string             `json:"email" bson:"email,omitempty"`
	CoachName        string             `json:"coach_name,omitempty" bson:"coach_name,omitempty"`
	Role             string             `json:"role" bson:"role,omitempty"`
	EmailPreferences *EmailPreferences  `json:"email_preferences" bson:"email_preferences,omitempty"`
}

// EmailPreferences represent the email preferences for a specific Gym Association.
type EmailPreferences struct {
	NotifyOnAnnouncements bool `json:"notify_on_announcements" bson:"notify_on_announcements,omitempty"`
	NotifyOnRequests      bool `json:"notify_on_requests" bson:"notify_on_requests,omitempty"`
}

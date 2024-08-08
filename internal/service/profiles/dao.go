package profiles

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Profile represents the Profile mongodb entity
type Profile struct {
	ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`

	CognitoID string `json:"cognito_id,omitempty" bson:"cognito_id,omitempty" validate:"required"`

	// Personal Data is duplicated from Cognito via post-signup-lambda function
	Email       string `json:"email" bson:"email"`
	FirstName   string `json:"first_name" bson:"first_name"`
	LastName    string `json:"last_name" bson:"last_name"`
	PhoneNumber string `json:"phone_number" bson:"phone_number"`

	AvatarURL string           `json:"avatar_url" bson:"avatar_url"`
	Gyms      []GymAssociation `json:"gyms,omit" bson:"gyms"`

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

// GymAssociation represents a user's association to a gym.
// A GymAssociation can either be for a student or a coach.
type GymAssociation struct {
	GymID primitive.ObjectID `json:"gym_id,omitempty" bson:"gym_id,omitempty"`

	// The status of the gym association
	CoachName        string            `json:"coach_name,omitempty" bson:"coach_name,omitempty"`
	Role             string            `json:"role,omitempty" bson:"role,omitempty"`
	EmailPreferences *EmailPreferences `json:"email_preferences,omitempty" bson:"email_preferences"`
}

// EmailPreferences represent the email preferences for a specific Gym Association.
type EmailPreferences struct {
	NotifyOnAnnouncements bool `json:"notify_on_announcements,omitempty" bson:"notify_on_announcements,omitempty"`
	NotifyOnGymRequests   bool `json:"notify_on_requests,omitempty" bson:"notify_on_requests,omitempty"`
}

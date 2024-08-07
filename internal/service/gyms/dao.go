package gyms

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Gym represents the Gym document structure in MongoDB.
type Gym struct {

	// auto-computed field, not sent in request body
	Slug string `json:"slug" bson:"slug,omitempty"`

	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Name        string             `json:"name" bson:"name,omitempty" validate:"required,alphanumeric_and_spaces"`
	Description string             `json:"description" bson:"description,omitempty" validate:"required"`

	TechniquesOfTheWeek []primitive.ObjectID `json:"techniques_of_the_week,omitempty" bson:"techniques_of_the_week,omitempty"`

	// Cognito User ID for the creator of the gym
	Creator string `json:"creator" bson:"creator,omitempty" validate:"required"`

	// Address information for the gym
	AddressLine1 string `json:"address_line_1" bson:"address_line_1,omitempty" validate:"required"`
	AddressLine2 string `json:"address_line_2" bson:"address_line_2,omitempty"`
	City         string `json:"city" bson:"city,omitempty" validate:"required"`
	State        string `json:"state" bson:"state,omitempty" validate:"required,is_state"`
	ZIP          string `json:"zip" bson:"zip,omitempty" validate:"required"`
	Country      string `json:"country" bson:"country,omitempty" validate:"required"`

	// PublicEmail is the public email displayed to students for contacting the gym
	PublicEmail string `json:"public_email" bson:"public_email,omitempty" validate:"required"`

	// CoachEmail is the coach's personal email address for notifications, sign-in, etc
	CoachEmail string `json:"coach_email" bson:"coach_email,omitempty" validate:"required"`

	// s3 object uri of gym logo
	Logo string `json:"logo" bson:"logo,omitempty" validate:"required"`

	// Disciplines
	Disciplines []string           `json:"disciplines" bson:"disciplines,omitempty" validate:"required"`
	Schedule    map[string][]Event `json:"schedule" bson:"schedule,omitempty" validate:"required"`

	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at"`
}

type Event struct {
	Title string `json:"title,omitempty" bson:"title,omitempty"`
	Start string `json:"start,omitempty" bson:"start,omitempty"`
	End   string `json:"end,omitempty" bson:"end,omitempty"`
}

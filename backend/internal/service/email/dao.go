package email

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Gym represents the Gym document structure in MongoDB.
type Email struct {
	ID        bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	Email     string        `json:"email,omitempty" bson:"email,omitempty" validate:"required"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at,omitempty"`
}

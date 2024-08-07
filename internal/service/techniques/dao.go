package techniques

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Technique represents the "Technique of the Week" mongodb entity
type Technique struct {
	ID          primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID       primitive.ObjectID `json:"gym_id,omitempty" bson:"gym_id,omitempty" validate:"required"`
	Title       string             `json:"title" bson:"title,omitempty" validate:"required"`
	Description string             `json:"description" bson:"description,omitempty" validate:"required"`
	Disciplines []string           `json:"disciplines" bson:"disciplines,omitempty" validate:"required"`

	// metadata
	CreatedAtWeek int       `json:"created_at_week" bson:"created_at_week"`
	CreatedAtYear int       `json:"created_at_year" bson:"created_at_year"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at"`
}

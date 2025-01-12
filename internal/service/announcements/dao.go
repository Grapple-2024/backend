package announcements

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// Gym represents the Gym document structure in MongoDB.
type Announcement struct {
	ID          bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID       bson.ObjectID `json:"gym_id,omitempty" bson:"gym_id,omitempty" validate:"required"`
	CoachName   string        `json:"coach_name,omitempty" bson:"coach_name,omitempty" validate:"required"`
	CoachAvatar string        `json:"coach_avatar,omitempty" bson:"coach_avatar,omitempty" validate:"required"`
	Title       string        `json:"title" bson:"title,omitempty" validate:"required"`
	Content     string        `json:"content" bson:"description,omitempty" validate:"required"`

	CreatedAtWeek int       `json:"created_at_week" bson:"created_at_week,omitempty"`
	CreatedAtYear int       `json:"created_at_year" bson:"created_at_year,omitempty"`
	CreatedAt     time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt     time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

package gym_series

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GymSeries represents a Gym's Series document in MongoDB.
type GymSeries struct {
	// keys
	ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID primitive.ObjectID `json:"gym_id" bson:"gym_id,omitempty" validate:"required"`

	// attributes
	Title       string  `json:"title,omitempty" bson:"title,omitempty" validate:"required"`
	Description string  `json:"description,omitempty" bson:"description,omitempty" validate:"required"`
	Videos      []Video `json:"videos" bson:"videos,omitempty" validate:"required"`

	// computed fields
	Disciplines  []string `json:"disciplines,omitempty" bson:"disciplines,stringsets,omitempty"`
	Difficulties []string `json:"difficulties,omitempty" bson:"difficulties,stringsets,omitempty"`

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

type Video struct {
	ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`

	Title       string   `json:"title,omitempty" bson:"title,omitempty" validate:"required"`
	Description string   `json:"description,omitempty" bson:"description,omitempty" validate:"required"`
	Difficulty  string   `json:"difficulty,omitempty" bson:"difficulty,omitempty" validate:"required"`
	Disciplines []string `json:"disciplines,omitempty" bson:"disciplines,stringsets,omitempty" validate:"required"`
	SortOrder   int32    `json:"sort_order,omitempty" bson:"sort_order,omitempty" validate:"required"`
	S3ObjectKey string   `json:"s3_object_key,omitempty" bson:"s3_object_key,omitempty" validate:"required"`

	// Computed fields
	PresignedURL string `json:"presigned_url,omitempty" bson:"presigned_url,omitempty"` // computed by requesting a presigned URL given the S3ObjectURI

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

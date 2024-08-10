package gym_series

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// GymSeries represents a Gym's Series document in MongoDB.
type GymSeries struct {
	// keys
	ID    primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID primitive.ObjectID `json:"gym_id" bson:"gym_id,omitempty"`

	// attributes
	Title       string   `validator:"nonzero" json:"title,omitempty" bson:"title,omitempty"`
	Description string   `validator:"nonzero" json:"description,omitempty" bson:"description,omitempty"`
	Difficulty  string   `validator:"nonzero" json:"difficulty,omitempty" bson:"difficulty,omitempty"`
	Disciplines []string `validator:"nonzero" json:"disciplines,omitempty" bson:"disciplines,stringsets,omitempty"`
	Videos      []Video  `json:"videos" bson:"videos,omitempty"`

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

type Video struct {
	ID primitive.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`

	Title       string   `validator:"nonzero" json:"title,omitempty" bson:"title,omitempty" validate:"required"`
	Description string   `validator:"nonzero" json:"description,omitempty" bson:"description,omitempty" validate:"required"`
	Difficulty  string   `validator:"nonzero" json:"difficulty,omitempty" bson:"difficulty,omitempty"`
	Disciplines []string `validator:"nonzero" json:"disciplines,omitempty" bson:"disciplines,stringsets,omitempty"`
	SortOrder   int32    `json:"sort_order,omitempty" bson:"sort_order,omitempty" validate:"required"`
	S3ObjectKey string   `json:"s3_object_key,omitempty" bson:"s3_object_key,omitempty" validate:"required"`

	// Computed fields
	PresignedURL string `json:"presigned_url,omitempty" bson:"presigned_url,omitempty"` // computed by requesting a presigned URL given the S3ObjectURI

	// metadata
	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

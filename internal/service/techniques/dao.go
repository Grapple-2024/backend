package techniques

import (
	"time"

	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"go.mongodb.org/mongo-driver/v2/bson"
)

// Technique represents the "Technique" mongodb entity
type Technique struct {
	ID     bson.ObjectID         `json:"id,omitempty" bson:"_id,omitempty"`
	Series *gym_series.GymSeries `json:"series,omitempty" bson:"series,omitempty" validate:"-"`

	Title       string `json:"title" bson:"title,omitempty" validate:"required"`
	Description string `json:"description" bson:"description,omitempty" validate:"required"`

	// computed fields
	// Disciplines []string `json:"disciplines" bson:"disciplines,omitempty"`
	DisplayWeekNum int `json:"week_number" bson:"week_number"`
	DisplayYearNum int `json:"year_number" bson:"year_number"`

	// metadata
	DisplayOnWeek *time.Time `json:"display_on_week" bson:"display_on_week" validate:"required"`

	CreatedAt time.Time `json:"created_at" bson:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at" bson:"updated_at,omitempty"`
}

package dao

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

// CheckIn represents a single gym attendance record.
// Method is "manual" (coach-entered) or "qr" (member self check-in via QR code).
// Records are immutable — no update path, only create and delete.
type CheckIn struct {
	ID          bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID       bson.ObjectID `json:"gym_id" bson:"gym_id"`
	MemberID    string        `json:"member_id" bson:"member_id"`
	MemberName  string        `json:"member_name" bson:"member_name"`
	AvatarURL   string        `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	CheckedInAt time.Time     `json:"checked_in_at" bson:"checked_in_at"`
	Method      string        `json:"method" bson:"method"` // "manual" | "qr"
	Notes       string        `json:"notes,omitempty" bson:"notes,omitempty"`
	CreatedAt   time.Time     `json:"created_at" bson:"created_at"`
}

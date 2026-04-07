package dao

import (
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

const (
	SystemAdult = "adult"
	SystemKids  = "kids"
)

// Adult BJJ belt progression
var AdultBelts = []string{
	"white", "blue", "purple", "brown", "black",
	"coral", "red/white", "red",
}

// Kids BJJ belt progression (IBJJF standard)
var KidsBelts = []string{
	"white",
	"grey/white", "grey", "grey/black",
	"yellow/white", "yellow", "yellow/black",
	"orange/white", "orange", "orange/black",
	"green/white", "green", "green/black",
}

// Promotion records a single belt/stripe change for a gym member.
// The member's current belt is always derived from their most recent Promotion —
// there is no separate "current_belt" field to keep in sync.
type Promotion struct {
	ID         bson.ObjectID `json:"id,omitempty" bson:"_id,omitempty"`
	GymID      bson.ObjectID `json:"gym_id" bson:"gym_id"`
	MemberID   string        `json:"member_id" bson:"member_id"`
	MemberName string        `json:"member_name" bson:"member_name"`
	AvatarURL  string        `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	System     string        `json:"system" bson:"system"`       // "adult" | "kids"
	Belt       string        `json:"belt" bson:"belt"`           // e.g. "blue", "grey/black"
	Stripes    int           `json:"stripes" bson:"stripes"`     // 0–4
	Notes      string        `json:"notes,omitempty" bson:"notes,omitempty"`
	PromotedBy string        `json:"promoted_by,omitempty" bson:"promoted_by,omitempty"`
	PromotedAt time.Time     `json:"promoted_at" bson:"promoted_at"`
	CreatedAt  time.Time     `json:"created_at" bson:"created_at"`
}

// ValidateBelt returns an error if belt is not valid for the given system.
func ValidateBelt(system, belt string) error {
	var valid []string
	switch system {
	case SystemAdult:
		valid = AdultBelts
	case SystemKids:
		valid = KidsBelts
	default:
		return fmt.Errorf("system must be %q or %q", SystemAdult, SystemKids)
	}
	for _, b := range valid {
		if b == belt {
			return nil
		}
	}
	return fmt.Errorf("belt %q is not valid for system %q", belt, system)
}

// ValidateStripes returns an error if stripes is out of range.
func ValidateStripes(stripes int) error {
	if stripes < 0 || stripes > 4 {
		return fmt.Errorf("stripes must be 0–4, got %d", stripes)
	}
	return nil
}

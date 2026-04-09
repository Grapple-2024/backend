package members

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

// RichMember is the fully-joined member view returned by GET /members.
// It preserves the GymRequest envelope so existing frontend code keeps working,
// and adds enriched fields from profiles, billing, promotions, and check_ins.
type RichMember struct {
	// ── GymRequest core fields ───────────────────────────────────────────────
	ID             string `json:"id,omitempty"`
	GymID          string `json:"gym_id"`
	RequestorID    string `json:"requestor_id"`
	RequestorEmail string `json:"requestor_email"`
	FirstName      string `json:"first_name"`
	LastName       string `json:"last_name"`
	MembershipType string `json:"membership_type"`
	Role           string `json:"role"`
	Status         string `json:"status"`
	CreatedAt      string `json:"created_at,omitempty"`

	// ── Joined: Profile ──────────────────────────────────────────────────────
	Profile *MemberProfile `json:"profile,omitempty"`

	// ── Joined: Active billing record ────────────────────────────────────────
	Billing *MemberBilling `json:"billing,omitempty"`

	// ── Joined: Most recent promotion (current belt) ─────────────────────────
	CurrentBelt *CurrentBelt `json:"current_belt,omitempty"`

	// ── Joined: Last check-in timestamp ──────────────────────────────────────
	LastCheckIn *time.Time `json:"last_check_in,omitempty"`
}

// MemberProfile is the subset of Profile we expose.
type MemberProfile struct {
	AvatarURL   string `json:"avatar_url,omitempty" bson:"avatar_url,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty" bson:"phone_number,omitempty"`
}

// MemberBilling is the subset of MemberBilling we expose.
type MemberBilling struct {
	Status          string    `json:"status" bson:"status"`
	PlanName        string    `json:"plan_name" bson:"plan_name"`
	NextPaymentDate time.Time `json:"next_payment_date" bson:"next_payment_date"`
}

// CurrentBelt is the subset of Promotion we expose.
type CurrentBelt struct {
	System  string `json:"system" bson:"system"`
	Belt    string `json:"belt" bson:"belt"`
	Stripes int    `json:"stripes" bson:"stripes"`
}

// Service aggregates member data from multiple collections.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	Requests *mongo.Collection
}

// NewService creates and returns a new members.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	return &Service{
		RBAC:     rbac,
		Client:   mc,
		Requests: mc.Database("grapple").Collection("gymRequests"),
	}, nil
}

// ProcessGetAll handles GET /members?gym_id=X with optional filters.
// Query params: gym_id, status, role, membership_type, search, page, page_size, sort_column, sort_direction
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, _ int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, authErr := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:members", gymID), rbac.ActionRead)
	if authErr != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	// ── Pagination & sort params ──────────────────────────────────────────────
	page := 1
	if p := req.QueryStringParameters["page"]; p != "" {
		if v, e := strconv.Atoi(p); e == nil && v > 0 {
			page = v
		}
	}
	pageSize := 500 // default — return all members unless paginated
	if ps := req.QueryStringParameters["page_size"]; ps != "" {
		if v, e := strconv.Atoi(ps); e == nil && v > 0 {
			pageSize = v
		}
	}
	sortCol := req.QueryStringParameters["sort_column"]
	if sortCol == "" {
		sortCol = "first_name"
	}
	sortDir := 1
	if req.QueryStringParameters["sort_direction"] == "-1" {
		sortDir = -1
	}

	// ── Match stage ───────────────────────────────────────────────────────────
	match := bson.M{"gym_id": gymObjID}

	if v := req.QueryStringParameters["status"]; v != "" {
		match["status"] = v
	}
	if v := req.QueryStringParameters["role"]; v != "" {
		match["role"] = v
	}
	if v := req.QueryStringParameters["membership_type"]; v != "" {
		match["membership_type"] = v
	}
	if search := req.QueryStringParameters["search"]; search != "" {
		match["$or"] = bson.A{
			bson.M{"first_name": bson.M{"$regex": search, "$options": "i"}},
			bson.M{"last_name": bson.M{"$regex": search, "$options": "i"}},
			bson.M{"requestor_email": bson.M{"$regex": search, "$options": "i"}},
		}
	}

	skip := int64((page - 1) * pageSize)

	// ── Aggregation pipeline ──────────────────────────────────────────────────
	pipeline := mongo.Pipeline{
		// 1. Filter to this gym (+ optional filters)
		{{Key: "$match", Value: match}},

		// 2. Join profiles (string-keyed: requestor_id → cognito_id)
		{{Key: "$lookup", Value: bson.M{
			"from": "profiles",
			"let":  bson.M{"rid": "$requestor_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$eq": bson.A{"$cognito_id", "$$rid"}}}},
				bson.M{"$project": bson.M{"avatar_url": 1, "phone_number": 1}},
				bson.M{"$limit": 1},
			},
			"as": "profile_arr",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$profile_arr", "preserveNullAndEmptyArrays": true}}},

		// 3. Join member_billing (active record only; most recently created)
		{{Key: "$lookup", Value: bson.M{
			"from": "member_billing",
			"let":  bson.M{"rid": "$requestor_id", "gid": "$gym_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$member_id", "$$rid"}},
					bson.M{"$eq": bson.A{"$gym_id", "$$gid"}},
					bson.M{"$eq": bson.A{"$status", "active"}},
				}}}},
				bson.M{"$sort": bson.M{"created_at": -1}},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{"status": 1, "plan_name": 1, "next_payment_date": 1}},
			},
			"as": "billing_arr",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$billing_arr", "preserveNullAndEmptyArrays": true}}},

		// 4. Join promotions (most recent = current belt)
		{{Key: "$lookup", Value: bson.M{
			"from": "promotions",
			"let":  bson.M{"rid": "$requestor_id", "gid": "$gym_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$member_id", "$$rid"}},
					bson.M{"$eq": bson.A{"$gym_id", "$$gid"}},
				}}}},
				bson.M{"$sort": bson.M{"promoted_at": -1}},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{"system": 1, "belt": 1, "stripes": 1}},
			},
			"as": "belt_arr",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$belt_arr", "preserveNullAndEmptyArrays": true}}},

		// 5. Join check_ins (most recent)
		{{Key: "$lookup", Value: bson.M{
			"from": "check_ins",
			"let":  bson.M{"rid": "$requestor_id", "gid": "$gym_id"},
			"pipeline": bson.A{
				bson.M{"$match": bson.M{"$expr": bson.M{"$and": bson.A{
					bson.M{"$eq": bson.A{"$member_id", "$$rid"}},
					bson.M{"$eq": bson.A{"$gym_id", "$$gid"}},
				}}}},
				bson.M{"$sort": bson.M{"checked_in_at": -1}},
				bson.M{"$limit": 1},
				bson.M{"$project": bson.M{"checked_in_at": 1}},
			},
			"as": "checkin_arr",
		}}},
		{{Key: "$unwind", Value: bson.M{"path": "$checkin_arr", "preserveNullAndEmptyArrays": true}}},

		// 6. Project into RichMember shape
		{{Key: "$project", Value: bson.M{
			"_id":             1,
			"gym_id":          1,
			"requestor_id":    1,
			"requestor_email": 1,
			"first_name":      1,
			"last_name":       1,
			"membership_type": 1,
			"role":            1,
			"status":          1,
			"created_at":      1,
			// flatten joined docs — nil-safe (field absent if array was empty)
			"profile":      "$profile_arr",
			"billing":      "$billing_arr",
			"current_belt": "$belt_arr",
			"last_check_in": bson.M{"$ifNull": bson.A{"$checkin_arr.checked_in_at", nil}},
		}}},

		// 7. Sort + paginate
		{{Key: "$sort", Value: bson.D{{Key: sortCol, Value: sortDir}}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: int64(pageSize)}},
	}

	// ── Execute ───────────────────────────────────────────────────────────────
	cursor, err := s.Requests.Aggregate(ctx, pipeline)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("members aggregation failed: %w", err))
	}
	defer cursor.Close(ctx)

	// Decode into raw bson.M so we can re-key "_id" → "id" for JSON.
	var rawRows []bson.M
	if err := cursor.All(ctx, &rawRows); err != nil {
		return lambda.ServerError(fmt.Errorf("members decode failed: %w", err))
	}

	// Re-shape: convert ObjectID "_id" and "gym_id" to hex strings.
	members := make([]map[string]any, 0, len(rawRows))
	for _, row := range rawRows {
		m := make(map[string]any, len(row))
		for k, v := range row {
			switch k {
			case "_id":
				if oid, ok := v.(bson.ObjectID); ok {
					m["id"] = oid.Hex()
				}
			case "gym_id":
				if oid, ok := v.(bson.ObjectID); ok {
					m["gym_id"] = oid.Hex()
				} else {
					m[k] = v
				}
			default:
				m[k] = v
			}
		}
		members = append(members, m)
	}

	// Total count (for pagination metadata)
	totalCount, err := s.Requests.CountDocuments(ctx, match)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("count failed: %w", err))
	}

	resp, err := service.NewGetAllResponse("members", members, totalCount, len(members), page, pageSize)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID, ProcessPost, ProcessPut, ProcessDelete are not supported —
// mutations stay on their original service endpoints.
func (s *Service) ProcessGetByID(_ context.Context, _ events.APIGatewayProxyRequest, _ string) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "use GET /members?gym_id= instead")
}
func (s *Service) ProcessPost(_ context.Context, _ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "not supported")
}
func (s *Service) ProcessPut(_ context.Context, _ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "not supported")
}
func (s *Service) ProcessDelete(_ context.Context, _ events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "not supported")
}

// unused import guard
var _ = json.Marshal

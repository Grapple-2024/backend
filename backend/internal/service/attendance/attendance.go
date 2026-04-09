package attendance

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Service handles gym attendance / check-in records.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	*mongo.Collection
}

// NewService creates and returns a new attendance.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("check_ins")

	svc := &Service{
		RBAC:       rbac,
		Client:     mc,
		Collection: c,
	}

	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) ensureIndices(ctx context.Context) error {
	_, err := s.Collection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "gym_id", Value: 1}}},
		{Keys: bson.D{{Key: "member_id", Value: 1}}},
		{Keys: bson.D{{Key: "checked_in_at", Value: -1}}},
		// Compound index for daily uniqueness check (QR duplicate prevention)
		{Keys: bson.D{
			{Key: "gym_id", Value: 1},
			{Key: "member_id", Value: 1},
			{Key: "checked_in_at", Value: -1},
		}},
	})
	return err
}

// pathSegments returns path segments after the leading service name.
// /attendance/gymID/checkInID → ["gymID", "checkInID"]
func pathSegments(req events.APIGatewayProxyRequest) []string {
	parts := strings.Split(strings.TrimPrefix(req.Path, "/"), "/")
	var segs []string
	for _, p := range parts[1:] {
		if p != "" {
			segs = append(segs, p)
		}
	}
	return segs
}

// ProcessGetAll returns check-ins for a gym, optionally filtered by date and/or member.
// Query params: gym_id (required), date (YYYY-MM-DD, optional), member_id (optional)
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:attendance", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	filter := bson.M{"gym_id": gymObjID}

	// Filter by specific member
	if memberID := req.QueryStringParameters["member_id"]; memberID != "" {
		filter["member_id"] = memberID
	}

	// Filter by calendar day (UTC)
	if dateStr := req.QueryStringParameters["date"]; dateStr != "" {
		day, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid date format %q: use YYYY-MM-DD", dateStr))
		}
		start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
		end := start.Add(24 * time.Hour)
		filter["checked_in_at"] = bson.M{"$gte": start, "$lt": end}
	} else if req.QueryStringParameters["member_id"] == "" {
		// No date and no member filter — default to last 30 days
		filter["checked_in_at"] = bson.M{"$gte": time.Now().UTC().AddDate(0, 0, -30)}
	}

	findOpts := options.Find().SetSort(bson.M{"checked_in_at": -1})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := s.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query check-ins: %w", err))
	}
	defer cursor.Close(ctx)

	var checkIns []dao.CheckIn
	if err := cursor.All(ctx, &checkIns); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to decode check-ins: %w", err))
	}
	if checkIns == nil {
		checkIns = []dao.CheckIn{}
	}

	resp, err := json.Marshal(checkIns)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID is unused — required by Lambda interface.
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "use GET /attendance?gym_id= instead")
}

// ProcessPost records a check-in.
// Coaches/owners can check in any member (method: "manual").
// Students can check themselves in (method: "qr") — token.Sub must match member_id.
// Prevents duplicate check-ins within the same calendar day.
// Body: { gym_id, member_id, member_name, avatar_url, method, notes }
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	var payload struct {
		GymID      string `json:"gym_id"`
		MemberID   string `json:"member_id"`
		MemberName string `json:"member_name"`
		AvatarURL  string `json:"avatar_url"`
		Method     string `json:"method"` // "manual" | "qr"
		Notes      string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}
	if payload.GymID == "" || payload.MemberID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id and member_id are required")
	}
	if payload.Method != "manual" && payload.Method != "qr" {
		payload.Method = "manual"
	}

	// Students may only check themselves in
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:attendance", payload.GymID), rbac.ActionCreate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}
	if payload.Method == "qr" && token.Sub != payload.MemberID {
		return lambda.ClientError(http.StatusForbidden, "students may only check themselves in")
	}

	gymObjID, err := bson.ObjectIDFromHex(payload.GymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	now := time.Now().UTC()

	// Duplicate check: same member, same gym, same calendar day
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	existingFilter := bson.M{
		"gym_id":        gymObjID,
		"member_id":     payload.MemberID,
		"checked_in_at": bson.M{"$gte": dayStart, "$lt": dayEnd},
	}
	count, err := s.Collection.CountDocuments(ctx, existingFilter)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed duplicate check: %w", err))
	}
	if count > 0 {
		return lambda.ClientError(http.StatusConflict, "member already checked in today")
	}

	checkIn := dao.CheckIn{
		GymID:       gymObjID,
		MemberID:    payload.MemberID,
		MemberName:  payload.MemberName,
		AvatarURL:   payload.AvatarURL,
		CheckedInAt: now,
		Method:      payload.Method,
		Notes:       payload.Notes,
		CreatedAt:   now,
	}

	var created dao.CheckIn
	if err := mongoext.Insert(ctx, s.Collection, checkIn, &created); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to record check-in: %w", err))
	}

	resp, err := json.Marshal(created)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut is unused — check-ins are immutable.
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "check-ins are immutable")
}

// ProcessDelete removes a check-in record (erroneous entry correction).
// Path: /attendance/{gymID}/{checkInID}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)
	if len(segs) < 2 {
		return lambda.ClientError(http.StatusBadRequest, "path must be /attendance/{gymID}/{checkInID}")
	}
	gymID, checkInID := segs[0], segs[1]

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:attendance", gymID), rbac.ActionDelete)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	if err := mongoext.DeleteOne(ctx, s.Collection, checkInID); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to delete check-in: %w", err))
	}

	return lambda.NewResponse(http.StatusOK, `{"message":"check-in deleted"}`, nil), nil
}

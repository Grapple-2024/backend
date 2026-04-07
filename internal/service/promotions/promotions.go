package promotions

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

// Service handles belt promotion records.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	*mongo.Collection
}

// NewService creates and returns a new promotions.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("promotions")

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
		{Keys: bson.D{{Key: "promoted_at", Value: -1}}},
		// Compound for per-member history queries
		{Keys: bson.D{
			{Key: "gym_id", Value: 1},
			{Key: "member_id", Value: 1},
			{Key: "promoted_at", Value: -1},
		}},
	})
	return err
}

// pathSegments returns path segments after the leading service name.
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

// ProcessGetAll dispatches on path:
//   - GET /promotions?gym_id=&member_id=        → full history for a member
//   - GET /promotions/current?gym_id=            → latest promotion per member (map)
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)
	if len(segs) > 0 && segs[0] == "current" {
		return s.getCurrentBelts(ctx, req, token)
	}
	return s.listHistory(ctx, req, token, limit)
}

// listHistory returns the full promotion history for a single member.
func (s *Service) listHistory(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token, limit int32) (events.APIGatewayProxyResponse, error) {
	gymID := req.QueryStringParameters["gym_id"]
	memberID := req.QueryStringParameters["member_id"]
	if gymID == "" || memberID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id and member_id query params are required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:promotions", gymID), rbac.ActionRead)
	if err != nil || !ok {
		// Students may read their own history
		if token.Sub != memberID {
			return lambda.ClientError(http.StatusForbidden, "forbidden")
		}
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	filter := bson.M{"gym_id": gymObjID, "member_id": memberID}
	findOpts := options.Find().SetSort(bson.M{"promoted_at": -1})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := s.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query promotions: %w", err))
	}
	defer cursor.Close(ctx)

	var promotions []dao.Promotion
	if err := cursor.All(ctx, &promotions); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to decode promotions: %w", err))
	}
	if promotions == nil {
		promotions = []dao.Promotion{}
	}

	resp, err := json.Marshal(promotions)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// getCurrentBelts returns the latest promotion for every member in a gym.
// Response: map[memberID]Promotion
func (s *Service) getCurrentBelts(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token) (events.APIGatewayProxyResponse, error) {
	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:promotions", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	// Aggregation: for each member, pick the most recent promotion
	pipeline := []bson.M{
		{"$match": bson.M{"gym_id": gymObjID}},
		{"$sort": bson.M{"promoted_at": -1}},
		{"$group": bson.M{
			"_id":       "$member_id",
			"promotion": bson.M{"$first": "$$ROOT"},
		}},
	}

	cursor, err := s.Collection.Aggregate(ctx, pipeline)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to aggregate current belts: %w", err))
	}
	defer cursor.Close(ctx)

	type row struct {
		MemberID  string        `bson:"_id"`
		Promotion dao.Promotion `bson:"promotion"`
	}

	result := make(map[string]dao.Promotion)
	for cursor.Next(ctx) {
		var r row
		if err := cursor.Decode(&r); err != nil {
			continue
		}
		result[r.MemberID] = r.Promotion
	}
	if err := cursor.Err(); err != nil {
		return lambda.ServerError(fmt.Errorf("cursor error: %w", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID is unused — required by Lambda interface.
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "use GET /promotions?gym_id=&member_id= instead")
}

// ProcessPost records a new promotion.
// Body: { gym_id, member_id, member_name, avatar_url, system, belt, stripes, notes, promoted_by, promoted_at }
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	var payload dao.Promotion
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	gymID := payload.GymID.Hex()
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:promotions", gymID), rbac.ActionCreate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	if payload.MemberID == "" {
		return lambda.ClientError(http.StatusBadRequest, "member_id is required")
	}
	if err := dao.ValidateBelt(payload.System, payload.Belt); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}
	if err := dao.ValidateStripes(payload.Stripes); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	now := time.Now().UTC()
	if payload.PromotedAt.IsZero() {
		payload.PromotedAt = now
	}
	payload.CreatedAt = now

	var created dao.Promotion
	if err := mongoext.Insert(ctx, s.Collection, payload, &created); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to record promotion: %w", err))
	}

	resp, err := json.Marshal(created)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut is unused — promotions are immutable records.
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "promotions are immutable")
}

// ProcessDelete removes a promotion record (erroneous entry correction).
// Path: /promotions/{gymID}/{promotionID}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)
	if len(segs) < 2 {
		return lambda.ClientError(http.StatusBadRequest, "path must be /promotions/{gymID}/{promotionID}")
	}
	gymID, promotionID := segs[0], segs[1]

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:promotions", gymID), rbac.ActionDelete)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	if err := mongoext.DeleteOne(ctx, s.Collection, promotionID); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to delete promotion: %w", err))
	}

	return lambda.NewResponse(http.StatusOK, `{"message":"promotion deleted"}`, nil), nil
}

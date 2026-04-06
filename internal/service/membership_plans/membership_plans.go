package membership_plans

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Service handles CRUD for gym membership plans.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	*mongo.Collection
	validate *validator.Validate
}

// NewService creates and returns a new membership_plans.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("membership_plans")

	svc := &Service{
		RBAC:       rbac,
		Client:     mc,
		Collection: c,
		validate:   validator.New(),
	}

	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) ensureIndices(ctx context.Context) error {
	_, err := s.Collection.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "gym_id", Value: 1}},
	})
	return err
}

// ProcessGetAll returns all membership plans for a gym.
// Required query param: gym_id
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid gym_id: %v", err))
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:plans", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	filter := bson.M{"gym_id": gymObjID}

	findOpts := options.Find()
	findOpts.SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := s.Collection.Find(ctx, filter, findOpts)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query membership plans: %w", err))
	}
	defer cursor.Close(ctx)

	var plans []dao.MembershipPlan
	if err := cursor.All(ctx, &plans); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to decode membership plans: %w", err))
	}

	if plans == nil {
		plans = []dao.MembershipPlan{}
	}

	resp, err := json.Marshal(plans)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID returns a single membership plan by ID.
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	var plan dao.MembershipPlan
	if err := mongoext.FindByID(ctx, s.Collection, id, &plan); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("plan not found: %v", err))
	}

	gymID := plan.GymID.Hex()
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:plans", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	resp, err := json.Marshal(plan)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPost creates a new membership plan.
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	var plan dao.MembershipPlan
	if err := json.Unmarshal([]byte(req.Body), &plan); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	gymID := plan.GymID.Hex()
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:plans", gymID), rbac.ActionCreate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	if err := s.validate.Struct(plan); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("validation failed: %v", err))
	}

	if plan.Currency == "" {
		plan.Currency = "usd"
	}
	plan.IsActive = true
	plan.CreatedAt = time.Now().UTC()
	plan.UpdatedAt = time.Now().UTC()

	var created dao.MembershipPlan
	if err := mongoext.Insert(ctx, s.Collection, plan, &created); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to create membership plan: %w", err))
	}

	resp, err := json.Marshal(created)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut updates an existing membership plan.
// Path: /membership-plans/{gymID}/{planID}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	// Path: /membership-plans/{gymID}/{planID}
	gymID := req.PathParameters["gymID"]
	planID := req.PathParameters["planID"]
	if gymID == "" || planID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gymID and planID path params are required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:plans", gymID), rbac.ActionUpdate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	var updates dao.MembershipPlan
	if err := json.Unmarshal([]byte(req.Body), &updates); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	updates.UpdatedAt = time.Now().UTC()

	var updated dao.MembershipPlan
	if err := mongoext.UpdateByID(ctx, s.Collection, planID, updates, &updated, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update membership plan: %w", err))
	}

	resp, err := json.Marshal(updated)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete soft-deletes a membership plan (sets is_active = false).
// Path: /membership-plans/{gymID}/{planID}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	gymID := req.PathParameters["gymID"]
	planID := req.PathParameters["planID"]
	if gymID == "" || planID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gymID and planID path params are required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:plans", gymID), rbac.ActionDelete)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	deactivated := struct {
		IsActive  bool      `bson:"is_active"`
		UpdatedAt time.Time `bson:"updated_at"`
	}{
		IsActive:  false,
		UpdatedAt: time.Now().UTC(),
	}

	var updated dao.MembershipPlan
	if err := mongoext.UpdateByID(ctx, s.Collection, planID, deactivated, &updated, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to deactivate membership plan: %w", err))
	}

	return lambda.NewResponse(http.StatusOK, `{"message":"plan deactivated"}`, nil), nil
}

package member_billing

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

// Service manages member billing assignments and payment records.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	BillingCollection  *mongo.Collection
	PaymentsCollection *mongo.Collection
	PlansCollection    *mongo.Collection
}

// NewService creates and returns a new member_billing.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	svc := &Service{
		RBAC:               rbac,
		Client:             mc,
		BillingCollection:  mc.Database("grapple").Collection("member_billing"),
		PaymentsCollection: mc.Database("grapple").Collection("payment_records"),
		PlansCollection:    mc.Database("grapple").Collection("membership_plans"),
	}

	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) ensureIndices(ctx context.Context) error {
	_, err := s.BillingCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "gym_id", Value: 1}}},
		{Keys: bson.D{{Key: "member_id", Value: 1}}},
		{Keys: bson.D{{Key: "gym_id", Value: 1}, {Key: "member_id", Value: 1}}},
	})
	if err != nil {
		return err
	}

	_, err = s.PaymentsCollection.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "gym_id", Value: 1}}},
		{Keys: bson.D{{Key: "member_id", Value: 1}}},
		{Keys: bson.D{{Key: "billing_id", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	})
	return err
}

// pathSegments splits req.Path into non-empty segments, dropping the leading service name.
// /member-billing/payments → ["payments"]
// /member-billing/gymID/recordID → ["gymID", "recordID"]
func pathSegments(req events.APIGatewayProxyRequest) []string {
	parts := strings.Split(strings.TrimPrefix(req.Path, "/"), "/")
	// parts[0] is "member-billing" — skip it
	var segs []string
	for _, p := range parts[1:] {
		if p != "" {
			segs = append(segs, p)
		}
	}
	return segs
}

// ProcessGetAll dispatches on the path:
//   - GET /member-billing?gym_id=  → list billing assignments
//   - GET /member-billing/payments?gym_id= → list payment records
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)
	if len(segs) > 0 && segs[0] == "payments" {
		return s.listPayments(ctx, req, token, limit)
	}
	return s.listBilling(ctx, req, token, limit)
}

func (s *Service) listBilling(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token, limit int32) (events.APIGatewayProxyResponse, error) {
	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	filter := bson.M{"gym_id": gymObjID}
	if memberID := req.QueryStringParameters["member_id"]; memberID != "" {
		filter["member_id"] = memberID
	}

	findOpts := options.Find().SetSort(bson.M{"created_at": -1})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := s.BillingCollection.Find(ctx, filter, findOpts)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query billing: %w", err))
	}
	defer cursor.Close(ctx)

	var records []dao.MemberBilling
	if err := cursor.All(ctx, &records); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to decode billing: %w", err))
	}
	if records == nil {
		records = []dao.MemberBilling{}
	}

	resp, err := json.Marshal(records)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

func (s *Service) listPayments(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token, limit int32) (events.APIGatewayProxyResponse, error) {
	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	filter := bson.M{"gym_id": gymObjID}
	if memberID := req.QueryStringParameters["member_id"]; memberID != "" {
		filter["member_id"] = memberID
	}
	if status := req.QueryStringParameters["status"]; status != "" {
		filter["status"] = status
	}

	findOpts := options.Find().SetSort(bson.M{"due_date": -1})
	if limit > 0 {
		findOpts.SetLimit(int64(limit))
	}

	cursor, err := s.PaymentsCollection.Find(ctx, filter, findOpts)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query payments: %w", err))
	}
	defer cursor.Close(ctx)

	var records []dao.PaymentRecord
	if err := cursor.All(ctx, &records); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to decode payments: %w", err))
	}
	if records == nil {
		records = []dao.PaymentRecord{}
	}

	resp, err := json.Marshal(records)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID is unused but required by the Lambda interface.
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "use GET /member-billing?gym_id= instead")
}

// ProcessPost assigns a membership plan to a member.
// Creates a MemberBilling record and the first PaymentRecord.
// Body: { gym_id, member_id, plan_id, member_name, start_date }
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	var payload struct {
		GymID      string    `json:"gym_id"`
		MemberID   string    `json:"member_id"`
		PlanID     string    `json:"plan_id"`
		MemberName string    `json:"member_name"`
		StartDate  time.Time `json:"start_date"`
	}
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}
	if payload.GymID == "" || payload.MemberID == "" || payload.PlanID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id, member_id, and plan_id are required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", payload.GymID), rbac.ActionCreate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(payload.GymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}
	planObjID, err := bson.ObjectIDFromHex(payload.PlanID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid plan_id")
	}

	// Load the plan to get price + interval details
	var plan dao.MembershipPlan
	if err := mongoext.FindByID(ctx, s.PlansCollection, payload.PlanID, &plan); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("plan not found: %v", err))
	}

	now := time.Now().UTC()
	startDate := payload.StartDate
	if startDate.IsZero() {
		startDate = now
	}

	nextPayment := nextPaymentDate(startDate, plan.BillingType, plan.Interval)

	billing := dao.MemberBilling{
		GymID:           gymObjID,
		MemberID:        payload.MemberID,
		PlanID:          planObjID,
		PlanName:        plan.Name,
		MemberName:      payload.MemberName,
		Status:          "active",
		StartDate:       startDate,
		NextPaymentDate: nextPayment,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	var createdBilling dao.MemberBilling
	if err := mongoext.Insert(ctx, s.BillingCollection, billing, &createdBilling); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to create billing record: %w", err))
	}

	// Create first payment record
	payment := dao.PaymentRecord{
		GymID:      gymObjID,
		MemberID:   payload.MemberID,
		BillingID:  createdBilling.ID,
		PlanID:     planObjID,
		PlanName:   plan.Name,
		MemberName: payload.MemberName,
		Amount:     plan.Price,
		Currency:   plan.Currency,
		Status:     "unpaid",
		DueDate:    startDate,
		CreatedAt:  now,
		UpdatedAt:  now,
	}

	var createdPayment dao.PaymentRecord
	if err := mongoext.Insert(ctx, s.PaymentsCollection, payment, &createdPayment); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to create payment record: %w", err))
	}

	result := map[string]any{
		"billing": createdBilling,
		"payment": createdPayment,
	}
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut dispatches on path segments:
//   - PUT /member-billing/{gymID}/{billingID}        → update billing status
//   - PUT /member-billing/payments/{gymID}/{recordID} → update payment status
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)

	// /member-billing/payments/{gymID}/{recordID}
	if len(segs) >= 3 && segs[0] == "payments" {
		return s.updatePaymentRecord(ctx, req, token, segs[1], segs[2])
	}

	// /member-billing/{gymID}/{billingID}
	if len(segs) >= 2 {
		return s.updateBillingRecord(ctx, req, token, segs[0], segs[1])
	}

	return lambda.ClientError(http.StatusBadRequest, "path must be /member-billing/{gymID}/{billingID} or /member-billing/payments/{gymID}/{recordID}")
}

func (s *Service) updateBillingRecord(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token, gymID, billingID string) (events.APIGatewayProxyResponse, error) {
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", gymID), rbac.ActionUpdate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	var payload struct {
		Status string `json:"status"` // "active" | "paused" | "cancelled"
	}
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	validStatuses := map[string]bool{"active": true, "paused": true, "cancelled": true}
	if !validStatuses[payload.Status] {
		return lambda.ClientError(http.StatusBadRequest, "status must be one of: active, paused, cancelled")
	}

	update := struct {
		Status    string    `bson:"status"`
		UpdatedAt time.Time `bson:"updated_at"`
	}{
		Status:    payload.Status,
		UpdatedAt: time.Now().UTC(),
	}

	var updated dao.MemberBilling
	if err := mongoext.UpdateByID(ctx, s.BillingCollection, billingID, update, &updated, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update billing record: %w", err))
	}

	resp, err := json.Marshal(updated)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

func (s *Service) updatePaymentRecord(ctx context.Context, req events.APIGatewayProxyRequest, token *service.Token, gymID, recordID string) (events.APIGatewayProxyResponse, error) {
	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", gymID), rbac.ActionUpdate)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	var payload struct {
		Status string `json:"status"` // "paid" | "unpaid" | "overdue"
		Notes  string `json:"notes"`
	}
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	validStatuses := map[string]bool{"paid": true, "unpaid": true, "overdue": true}
	if !validStatuses[payload.Status] {
		return lambda.ClientError(http.StatusBadRequest, "status must be one of: paid, unpaid, overdue")
	}

	now := time.Now().UTC()
	update := bson.M{
		"status":     payload.Status,
		"updated_at": now,
	}
	if payload.Notes != "" {
		update["notes"] = payload.Notes
	}
	if payload.Status == "paid" {
		update["paid_at"] = now
	} else {
		update["paid_at"] = nil
	}

	objID, err := bson.ObjectIDFromHex(recordID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid record ID")
	}

	filter := bson.M{"_id": objID}
	_, err = s.PaymentsCollection.UpdateOne(ctx, filter, bson.M{"$set": update})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update payment record: %w", err))
	}

	var updated dao.PaymentRecord
	if err := mongoext.FindByID(ctx, s.PaymentsCollection, recordID, &updated); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to fetch updated record: %w", err))
	}

	resp, err := json.Marshal(updated)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete removes a billing assignment.
// Path: /member-billing/{gymID}/{billingID}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	segs := pathSegments(req)
	if len(segs) < 2 {
		return lambda.ClientError(http.StatusBadRequest, "path must be /member-billing/{gymID}/{billingID}")
	}
	gymID, billingID := segs[0], segs[1]

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:billing", gymID), rbac.ActionDelete)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	if err := mongoext.DeleteOne(ctx, s.BillingCollection, billingID); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to delete billing record: %w", err))
	}

	return lambda.NewResponse(http.StatusOK, `{"message":"billing record deleted"}`, nil), nil
}

// nextPaymentDate computes the next payment due date based on billing type and interval.
func nextPaymentDate(start time.Time, billingType, interval string) time.Time {
	if billingType == "one_time" {
		return start
	}
	switch interval {
	case "weekly":
		return start.AddDate(0, 0, 7)
	case "yearly":
		return start.AddDate(1, 0, 0)
	default: // monthly
		return start.AddDate(0, 1, 0)
	}
}

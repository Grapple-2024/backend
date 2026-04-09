package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/sync/errgroup"
)

const adminUserID = "user_3BdfDoTV1og0ttLXHCLLNjk0EXJ"

// ── Types ─────────────────────────────────────────────────────────────────────

type AdminMetrics struct {
	TotalMRR          int64           `json:"total_mrr"`
	MRRByMonth        []MonthMRR      `json:"mrr_by_month"`
	ActiveGyms        int64           `json:"active_gyms"`
	NewGymsThisMonth  int64           `json:"new_gyms_this_month"`
	TotalStudents     int64           `json:"total_students"`
	ChurnRate         float64         `json:"churn_rate"`
	AvgStudentsPerGym float64         `json:"avg_students_per_gym"`
	FeatureAdoption   FeatureAdoption `json:"feature_adoption"`
}

type MonthMRR struct {
	Month string `json:"month"`
	MRR   int64  `json:"mrr"`
}

type FeatureAdoption struct {
	BillingPct      float64 `json:"billing_pct"`
	AttendancePct   float64 `json:"attendance_pct"`
	BeltTrackingPct float64 `json:"belt_tracking_pct"`
}

type AdminGym struct {
	ID           string     `json:"id"`
	Name         string     `json:"name"`
	OwnerName    string     `json:"owner_name"`
	OwnerEmail   string     `json:"owner_email"`
	Address      string     `json:"address"`
	State        string     `json:"state"`
	StudentCount int        `json:"student_count"`
	Tier         int        `json:"tier"`
	HasBilling   bool       `json:"has_billing"`
	LastActivity *time.Time `json:"last_activity"`
	CreatedAt    time.Time  `json:"created_at"`
}

type AdminGymRosterResponse struct {
	Data       []AdminGym `json:"data"`
	Count      int        `json:"count"`
	TotalCount int64      `json:"total_count"`
}

type AdminGymDetail struct {
	Gym         bson.M     `json:"gym"`
	AdminNotes  []AdminLog `json:"admin_notes"`
	ActivityLog []AdminLog `json:"activity_log"`
	MemberCount int64      `json:"member_count"`
	Revenue30d  int64      `json:"revenue_30d"`
}

type AdminLog struct {
	ID         string         `json:"id"`
	Action     string         `json:"action"`
	TargetName string         `json:"target_name"`
	Timestamp  time.Time      `json:"timestamp"`
	Metadata   map[string]any `json:"metadata"`
}

// ── Service ───────────────────────────────────────────────────────────────────

type Service struct {
	*mongoext.Client
	Gyms       *mongo.Collection
	Requests   *mongo.Collection
	Billing    *mongo.Collection
	Payments   *mongo.Collection
	CheckIns   *mongo.Collection
	Promotions *mongo.Collection
	AdminLogs  *mongo.Collection
}

func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	db := mc.Database("grapple")
	return &Service{
		Client:     mc,
		Gyms:       db.Collection("gyms"),
		Requests:   db.Collection("gymRequests"),
		Billing:    db.Collection("member_billing"),
		Payments:   db.Collection("payment_records"),
		CheckIns:   db.Collection("check_ins"),
		Promotions: db.Collection("promotions"),
		AdminLogs:  db.Collection("admin_logs"),
	}, nil
}

func (s *Service) requireAdmin(req events.APIGatewayProxyRequest) (*service.Token, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return nil, fmt.Errorf("unauthorized")
	}
	if token.Sub != adminUserID {
		return nil, fmt.Errorf("not found")
	}
	return token, nil
}

func (s *Service) logAction(ctx context.Context, action, targetID, targetName string, metadata map[string]any) {
	doc := bson.M{
		"action":      action,
		"actor_id":    adminUserID,
		"target_id":   targetID,
		"target_name": targetName,
		"timestamp":   time.Now().UTC(),
		"metadata":    metadata,
	}
	_, _ = s.AdminLogs.InsertOne(ctx, doc)
}

// ── ProcessGetAll — business metrics ──────────────────────────────────────────

func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, _ int32) (events.APIGatewayProxyResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return lambda.ClientError(http.StatusNotFound, "not found")
	}

	now := time.Now().UTC()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	thirtyDaysAgo := now.AddDate(0, 0, -30)
	twelveMonthsAgo := now.AddDate(-1, 0, 0)

	var data AdminMetrics
	data.MRRByMonth = make([]MonthMRR, 12)

	g, gctx := errgroup.WithContext(ctx)

	// 1. Total MRR (paid this calendar month, all gyms)
	g.Go(func() error {
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"status":  "paid",
				"paid_at": bson.M{"$gte": startOfMonth},
			}}},
			{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}}}},
		}
		cursor, err := s.Payments.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("total_mrr: %w", err)
		}
		defer cursor.Close(gctx)
		var rows []struct {
			Total int64 `bson:"total"`
		}
		if err := cursor.All(gctx, &rows); err != nil {
			return err
		}
		if len(rows) > 0 {
			data.TotalMRR = rows[0].Total
		}
		return nil
	})

	// 2. MRR by month (last 12 months, all gyms)
	g.Go(func() error {
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"status":  "paid",
				"paid_at": bson.M{"$gte": twelveMonthsAgo},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$paid_at"},
					"month": bson.M{"$month": "$paid_at"},
				},
				"mrr": bson.M{"$sum": "$amount"},
			}}},
			{{Key: "$sort", Value: bson.D{
				{Key: "_id.year", Value: 1},
				{Key: "_id.month", Value: 1},
			}}},
		}
		cursor, err := s.Payments.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("mrr_by_month: %w", err)
		}
		defer cursor.Close(gctx)
		type row struct {
			ID  struct{ Year, Month int } `bson:"_id"`
			MRR int64                    `bson:"mrr"`
		}
		var rows []row
		if err := cursor.All(gctx, &rows); err != nil {
			return err
		}
		lookup := make(map[string]int64, len(rows))
		for _, r := range rows {
			lookup[fmt.Sprintf("%d-%02d", r.ID.Year, r.ID.Month)] = r.MRR
		}
		for i := 11; i >= 0; i-- {
			t := now.AddDate(0, -i, 0)
			key := fmt.Sprintf("%d-%02d", t.Year(), int(t.Month()))
			data.MRRByMonth[11-i] = MonthMRR{Month: key, MRR: lookup[key]}
		}
		return nil
	})

	// 3. Active gyms + new this month
	g.Go(func() error {
		total, err := s.Gyms.CountDocuments(gctx, bson.M{})
		if err != nil {
			return fmt.Errorf("active_gyms: %w", err)
		}
		newThisMonth, err := s.Gyms.CountDocuments(gctx, bson.M{
			"created_at": bson.M{"$gte": startOfMonth},
		})
		if err != nil {
			return fmt.Errorf("new_gyms: %w", err)
		}
		data.ActiveGyms = total
		data.NewGymsThisMonth = newThisMonth
		return nil
	})

	// 4. Total students (all accepted requests globally)
	g.Go(func() error {
		count, err := s.Requests.CountDocuments(gctx, bson.M{"status": "Accepted"})
		if err != nil {
			return fmt.Errorf("total_students: %w", err)
		}
		data.TotalStudents = count
		return nil
	})

	// 5. Churn rate (cancelled in last 30d / active + cancelled in last 30d)
	g.Go(func() error {
		active, err := s.Billing.CountDocuments(gctx, bson.M{"status": "active"})
		if err != nil {
			return fmt.Errorf("churn_active: %w", err)
		}
		cancelled, err := s.Billing.CountDocuments(gctx, bson.M{
			"status":     "cancelled",
			"updated_at": bson.M{"$gte": thirtyDaysAgo},
		})
		if err != nil {
			return fmt.Errorf("churn_cancelled: %w", err)
		}
		if denom := active + cancelled; denom > 0 {
			data.ChurnRate = float64(cancelled) / float64(denom) * 100
		}
		return nil
	})

	// 6. Feature adoption (% of gyms using billing / attendance / belt tracking)
	g.Go(func() error {
		countDistinctGyms := func(col *mongo.Collection, filter bson.M) (int64, error) {
			pipeline := mongo.Pipeline{
				{{Key: "$match", Value: filter}},
				{{Key: "$group", Value: bson.M{"_id": "$gym_id"}}},
				{{Key: "$count", Value: "n"}},
			}
			cur, err := col.Aggregate(gctx, pipeline)
			if err != nil {
				return 0, err
			}
			defer cur.Close(gctx)
			var rows []struct {
				N int64 `bson:"n"`
			}
			if err := cur.All(gctx, &rows); err != nil {
				return 0, err
			}
			if len(rows) > 0 {
				return rows[0].N, nil
			}
			return 0, nil
		}

		billingCount, err := countDistinctGyms(s.Billing, bson.M{})
		if err != nil {
			return fmt.Errorf("adoption_billing: %w", err)
		}
		checkInCount, err := countDistinctGyms(s.CheckIns, bson.M{
			"checked_in_at": bson.M{"$gte": thirtyDaysAgo},
		})
		if err != nil {
			return fmt.Errorf("adoption_checkins: %w", err)
		}
		beltCount, err := countDistinctGyms(s.Promotions, bson.M{})
		if err != nil {
			return fmt.Errorf("adoption_belt: %w", err)
		}
		totalGyms, err := s.Gyms.CountDocuments(gctx, bson.M{})
		if err != nil {
			return fmt.Errorf("adoption_gyms_count: %w", err)
		}
		if totalGyms > 0 {
			f := float64(totalGyms)
			data.FeatureAdoption.BillingPct = float64(billingCount) / f * 100
			data.FeatureAdoption.AttendancePct = float64(checkInCount) / f * 100
			data.FeatureAdoption.BeltTrackingPct = float64(beltCount) / f * 100
		}
		return nil
	})

	if err := g.Wait(); err != nil {
		return lambda.ServerError(fmt.Errorf("admin metrics: %w", err))
	}

	// Derived: avg students per gym (computed after goroutines complete)
	if data.ActiveGyms > 0 {
		data.AvgStudentsPerGym = float64(data.TotalStudents) / float64(data.ActiveGyms)
	}

	resp, err := json.Marshal(data)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ── ProcessPost — gym roster with search + pagination ─────────────────────────

func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return lambda.ClientError(http.StatusNotFound, "not found")
	}

	var body struct {
		Search   string `json:"search"`
		Page     int    `json:"page"`
		PageSize int    `json:"page_size"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid body")
	}
	if body.Page <= 0 {
		body.Page = 1
	}
	if body.PageSize <= 0 || body.PageSize > 100 {
		body.PageSize = 25
	}

	filter := bson.M{}
	if body.Search != "" {
		filter["$or"] = []bson.M{
			{"name": bson.M{"$regex": body.Search, "$options": "i"}},
			{"coach_email": bson.M{"$regex": body.Search, "$options": "i"}},
			{"state": bson.M{"$regex": body.Search, "$options": "i"}},
		}
	}

	totalCount, err := s.Gyms.CountDocuments(ctx, filter)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("roster count: %w", err))
	}

	skip := int64((body.Page - 1) * body.PageSize)
	limit := int64(body.PageSize)

	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: filter}},
		// Count accepted members per gym
		{{Key: "$lookup", Value: bson.M{
			"from": "gymRequests",
			"let":  bson.M{"gid": "$_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{"$and": []any{
						bson.M{"$eq": []any{"$gym_id", "$$gid"}},
						bson.M{"$eq": []any{"$status", "Accepted"}},
					}},
				}}},
				{{Key: "$count", Value: "n"}},
			},
			"as": "student_count_arr",
		}}},
		// Most recent check-in (any member)
		{{Key: "$lookup", Value: bson.M{
			"from": "check_ins",
			"let":  bson.M{"gid": "$_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{"$eq": []any{"$gym_id", "$$gid"}},
				}}},
				{{Key: "$sort", Value: bson.M{"checked_in_at": -1}}},
				{{Key: "$limit", Value: 1}},
			},
			"as": "checkin_arr",
		}}},
		{{Key: "$unwind", Value: bson.M{
			"path":                       "$checkin_arr",
			"preserveNullAndEmptyArrays": true,
		}}},
		// Any active billing record
		{{Key: "$lookup", Value: bson.M{
			"from": "member_billing",
			"let":  bson.M{"gid": "$_id"},
			"pipeline": mongo.Pipeline{
				{{Key: "$match", Value: bson.M{
					"$expr": bson.M{"$and": []any{
						bson.M{"$eq": []any{"$gym_id", "$$gid"}},
						bson.M{"$eq": []any{"$status", "active"}},
					}},
				}}},
				{{Key: "$limit", Value: 1}},
			},
			"as": "billing_arr",
		}}},
		// Project final shape
		{{Key: "$project", Value: bson.M{
			"_id":              1,
			"name":             1,
			"address_line_1":   1,
			"city":             1,
			"state":            1,
			"tier":             bson.M{"$ifNull": []any{"$tier", 1}},
			"created_at":       1,
			"coach_first_name": 1,
			"coach_last_name":  1,
			"coach_email":      1,
			"student_count": bson.M{"$ifNull": []any{
				bson.M{"$arrayElemAt": []any{"$student_count_arr.n", 0}},
				0,
			}},
			"last_activity": "$checkin_arr.checked_in_at",
			"has_billing":   bson.M{"$gt": []any{bson.M{"$size": "$billing_arr"}, 0}},
		}}},
		{{Key: "$sort", Value: bson.M{"created_at": -1}}},
		{{Key: "$skip", Value: skip}},
		{{Key: "$limit", Value: limit}},
	}

	cursor, err := s.Gyms.Aggregate(ctx, pipeline)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("roster aggregate: %w", err))
	}
	defer cursor.Close(ctx)

	var raw []bson.M
	if err := cursor.All(ctx, &raw); err != nil {
		return lambda.ServerError(fmt.Errorf("roster decode: %w", err))
	}

	gyms := make([]AdminGym, 0, len(raw))
	for _, doc := range raw {
		g := docToAdminGym(doc)
		gyms = append(gyms, g)
	}

	resp, err := json.Marshal(AdminGymRosterResponse{
		Data:       gyms,
		Count:      len(gyms),
		TotalCount: totalCount,
	})
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ── ProcessGetByID — gym detail ───────────────────────────────────────────────

func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return lambda.ClientError(http.StatusNotFound, "not found")
	}

	gymObjID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid id")
	}

	var gym bson.M
	if err := s.Gyms.FindOne(ctx, bson.M{"_id": gymObjID}).Decode(&gym); err != nil {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}
	if oid, ok := gym["_id"].(bson.ObjectID); ok {
		gym["id"] = oid.Hex()
		delete(gym, "_id")
	}

	// Member count
	memberCount, err := s.Requests.CountDocuments(ctx, bson.M{
		"gym_id": gymObjID,
		"status": "Accepted",
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("member count: %w", err))
	}

	// Revenue last 30 days
	thirtyDaysAgo := time.Now().UTC().AddDate(0, 0, -30)
	revPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"gym_id":  gymObjID,
			"status":  "paid",
			"paid_at": bson.M{"$gte": thirtyDaysAgo},
		}}},
		{{Key: "$group", Value: bson.M{"_id": nil, "total": bson.M{"$sum": "$amount"}}}},
	}
	revCursor, err := s.Payments.Aggregate(ctx, revPipeline)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("revenue_30d: %w", err))
	}
	defer revCursor.Close(ctx)
	var revRows []struct {
		Total int64 `bson:"total"`
	}
	_ = revCursor.All(ctx, &revRows)
	var revenue30d int64
	if len(revRows) > 0 {
		revenue30d = revRows[0].Total
	}

	// Admin logs for this gym (most recent 20)
	logCursor, err := s.AdminLogs.Find(ctx,
		bson.M{"target_id": id},
		options.Find().SetSort(bson.M{"timestamp": -1}).SetLimit(20),
	)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("admin_logs: %w", err))
	}
	defer logCursor.Close(ctx)

	type rawLog struct {
		ID         bson.ObjectID  `bson:"_id"`
		Action     string         `bson:"action"`
		TargetName string         `bson:"target_name"`
		Timestamp  time.Time      `bson:"timestamp"`
		Metadata   map[string]any `bson:"metadata"`
	}
	var rawLogs []rawLog
	if err := logCursor.All(ctx, &rawLogs); err != nil {
		return lambda.ServerError(fmt.Errorf("log decode: %w", err))
	}

	var adminNotes []AdminLog
	var activityLog []AdminLog
	for _, l := range rawLogs {
		entry := AdminLog{
			ID:         l.ID.Hex(),
			Action:     l.Action,
			TargetName: l.TargetName,
			Timestamp:  l.Timestamp,
			Metadata:   l.Metadata,
		}
		if l.Action == "add_note" {
			adminNotes = append(adminNotes, entry)
		} else {
			activityLog = append(activityLog, entry)
		}
	}
	if adminNotes == nil {
		adminNotes = []AdminLog{}
	}
	if activityLog == nil {
		activityLog = []AdminLog{}
	}

	// Log the view action
	gymName, _ := gym["name"].(string)
	s.logAction(ctx, "view_gym", id, gymName, nil)

	detail := AdminGymDetail{
		Gym:         gym,
		AdminNotes:  adminNotes,
		ActivityLog: activityLog,
		MemberCount: memberCount,
		Revenue30d:  revenue30d,
	}
	resp, err := json.Marshal(detail)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ── ProcessPut — update tier or add note ──────────────────────────────────────

func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return lambda.ClientError(http.StatusNotFound, "not found")
	}

	id := req.PathParameters["id"]
	if id == "" {
		return lambda.ClientError(http.StatusBadRequest, "id is required")
	}
	gymObjID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid id")
	}

	var body struct {
		Action string `json:"action"`
		Tier   int    `json:"tier"`
		Note   string `json:"note"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid body")
	}

	var gym bson.M
	_ = s.Gyms.FindOne(ctx, bson.M{"_id": gymObjID}).Decode(&gym)
	gymName, _ := gym["name"].(string)

	switch body.Action {
	case "update_tier":
		if body.Tier < 1 || body.Tier > 3 {
			return lambda.ClientError(http.StatusBadRequest, "tier must be 1, 2, or 3")
		}
		oldTier := 1
		if t, ok := gym["tier"].(int32); ok {
			oldTier = int(t)
		}
		if _, err := s.Gyms.UpdateOne(ctx,
			bson.M{"_id": gymObjID},
			bson.M{"$set": bson.M{"tier": body.Tier, "updated_at": time.Now().UTC()}},
		); err != nil {
			return lambda.ServerError(fmt.Errorf("update_tier: %w", err))
		}
		s.logAction(ctx, "update_tier", id, gymName, map[string]any{
			"old_tier": oldTier,
			"new_tier": body.Tier,
		})

	case "add_note":
		if body.Note == "" {
			return lambda.ClientError(http.StatusBadRequest, "note cannot be empty")
		}
		s.logAction(ctx, "add_note", id, gymName, map[string]any{"note": body.Note})

	default:
		return lambda.ClientError(http.StatusBadRequest, "unknown action: "+body.Action)
	}

	return lambda.NewResponse(http.StatusOK, `{"ok":true}`, nil), nil
}

// ── ProcessDelete — force delete gym and all related data ─────────────────────

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if _, err := s.requireAdmin(req); err != nil {
		return lambda.ClientError(http.StatusNotFound, "not found")
	}

	id := req.PathParameters["id"]
	if id == "" {
		return lambda.ClientError(http.StatusBadRequest, "id is required")
	}
	gymObjID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid id")
	}

	var body struct {
		Password string `json:"password"`
	}
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid body")
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" || body.Password != adminPassword {
		return lambda.ClientError(http.StatusForbidden, "invalid password")
	}

	// Capture gym name before deletion for the audit log
	var gym bson.M
	_ = s.Gyms.FindOne(ctx, bson.M{"_id": gymObjID}).Decode(&gym)
	gymName, _ := gym["name"].(string)

	// Delete all gym-related data
	collections := []struct {
		col    *mongo.Collection
		filter bson.M
	}{
		{s.Gyms, bson.M{"_id": gymObjID}},
		{s.Requests, bson.M{"gym_id": gymObjID}},
		{s.Billing, bson.M{"gym_id": gymObjID}},
		{s.Payments, bson.M{"gym_id": gymObjID}},
		{s.CheckIns, bson.M{"gym_id": gymObjID}},
		{s.Promotions, bson.M{"gym_id": gymObjID}},
	}
	for _, c := range collections {
		if _, err := c.col.DeleteMany(ctx, c.filter); err != nil {
			return lambda.ServerError(fmt.Errorf("delete %s: %w", c.col.Name(), err))
		}
	}

	s.logAction(ctx, "delete_gym", id, gymName, nil)

	return lambda.NewResponse(http.StatusOK, `{"ok":true}`, nil), nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func docToAdminGym(doc bson.M) AdminGym {
	g := AdminGym{Tier: 1}

	if id, ok := doc["_id"].(bson.ObjectID); ok {
		g.ID = id.Hex()
	}
	g.Name = strVal(doc, "name")
	g.OwnerName = strVal(doc, "coach_first_name") + " " + strVal(doc, "coach_last_name")
	g.OwnerEmail = strVal(doc, "coach_email")
	g.State = strVal(doc, "state")
	g.Address = fmt.Sprintf("%s, %s, %s",
		strVal(doc, "address_line_1"),
		strVal(doc, "city"),
		strVal(doc, "state"),
	)

	switch v := doc["tier"].(type) {
	case int32:
		g.Tier = int(v)
	case int64:
		g.Tier = int(v)
	case int:
		g.Tier = v
	}

	switch v := doc["student_count"].(type) {
	case int32:
		g.StudentCount = int(v)
	case int64:
		g.StudentCount = int(v)
	case int:
		g.StudentCount = v
	}

	if hb, ok := doc["has_billing"].(bool); ok {
		g.HasBilling = hb
	}

	if la, ok := doc["last_activity"].(bson.DateTime); ok {
		t := la.Time()
		g.LastActivity = &t
	}

	if ca, ok := doc["created_at"].(bson.DateTime); ok {
		g.CreatedAt = ca.Time()
	} else if ca, ok := doc["created_at"].(time.Time); ok {
		g.CreatedAt = ca
	}

	return g
}

func strVal(doc bson.M, key string) string {
	if v, ok := doc[key].(string); ok {
		return v
	}
	return ""
}

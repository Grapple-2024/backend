package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"golang.org/x/sync/errgroup"
)

// MonthRevenue is a single data point for the 12-month revenue chart.
type MonthRevenue struct {
	Month   string `json:"month"`   // "2025-04"
	Revenue int64  `json:"revenue"` // cents
}

// WeekAttendance is a single data point for the 12-week attendance chart.
type WeekAttendance struct {
	Week  string `json:"week"`  // "2025-W14"
	Count int64  `json:"count"`
}

// OverdueMember is a member with at least one overdue payment (earliest due date).
type OverdueMember struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	Amount     int64  `json:"amount"`
	DueDate    string `json:"due_date"` // "2025-03-01"
}

// UpcomingRenewal is an unpaid payment record due within the next 7 days.
type UpcomingRenewal struct {
	MemberID   string `json:"member_id"`
	MemberName string `json:"member_name"`
	PlanName   string `json:"plan_name"`
	Amount     int64  `json:"amount"`
	DueDate    string `json:"due_date"`
}

// DashboardData is the full aggregated payload returned by GET /dashboard.
type DashboardData struct {
	ActiveMembers    int64             `json:"active_members"`
	MonthlyRevenue   int64             `json:"monthly_revenue"`
	TodayAttendance  int64             `json:"today_attendance"`
	OverdueCount     int64             `json:"overdue_count"`
	RevenueByMonth   []MonthRevenue    `json:"revenue_by_month"`
	AttendanceByWeek []WeekAttendance  `json:"attendance_by_week"`
	OverdueList      []OverdueMember   `json:"overdue_list"`
	PendingRequests  int64             `json:"pending_requests"`
	UpcomingRenewals []UpcomingRenewal `json:"upcoming_renewals"`
}

// Service aggregates data from multiple collections to power the coach dashboard.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	CheckIns *mongo.Collection
	Payments *mongo.Collection
	Billing  *mongo.Collection
	Requests *mongo.Collection
}

// NewService creates and returns a new dashboard.Service.
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	db := mc.Database("grapple")
	return &Service{
		RBAC:     rbac,
		Client:   mc,
		CheckIns: db.Collection("check_ins"),
		Payments: db.Collection("payment_records"),
		Billing:  db.Collection("member_billing"),
		Requests: db.Collection("gym_requests"),
	}, nil
}

// ProcessGetAll handles GET /dashboard?gym_id=<id>
// All aggregations run in parallel via errgroup.
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, _ int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, "unauthorized")
	}

	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "gym_id query param is required")
	}

	ok, err := s.RBAC.IsAuthorized(ctx, token.Sub, fmt.Sprintf("gym:%s:dashboard", gymID), rbac.ActionRead)
	if err != nil || !ok {
		return lambda.ClientError(http.StatusForbidden, "forbidden")
	}

	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid gym_id")
	}

	now := time.Now().UTC()
	var data DashboardData

	// Initialize slices so JSON never returns null.
	data.RevenueByMonth = make([]MonthRevenue, 0, 12)
	data.AttendanceByWeek = make([]WeekAttendance, 0, 12)
	data.OverdueList = []OverdueMember{}
	data.UpcomingRenewals = []UpcomingRenewal{}

	g, gctx := errgroup.WithContext(ctx)

	// ── 1. Active members ─────────────────────────────────────────────────────
	g.Go(func() error {
		count, err := s.Billing.CountDocuments(gctx, bson.M{
			"gym_id": gymObjID,
			"status": "active",
		})
		if err != nil {
			return fmt.Errorf("active_members: %w", err)
		}
		data.ActiveMembers = count
		return nil
	})

	// ── 2. Monthly revenue (current calendar month) ───────────────────────────
	g.Go(func() error {
		startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"gym_id": gymObjID,
				"status": "paid",
				"paid_at": bson.M{"$gte": startOfMonth},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id":   nil,
				"total": bson.M{"$sum": "$amount"},
			}}},
		}
		cursor, err := s.Payments.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("monthly_revenue: %w", err)
		}
		defer cursor.Close(gctx)
		var rows []struct {
			Total int64 `bson:"total"`
		}
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("monthly_revenue decode: %w", err)
		}
		if len(rows) > 0 {
			data.MonthlyRevenue = rows[0].Total
		}
		return nil
	})

	// ── 3. Today's attendance ─────────────────────────────────────────────────
	g.Go(func() error {
		dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		count, err := s.CheckIns.CountDocuments(gctx, bson.M{
			"gym_id":        gymObjID,
			"checked_in_at": bson.M{"$gte": dayStart},
		})
		if err != nil {
			return fmt.Errorf("today_attendance: %w", err)
		}
		data.TodayAttendance = count
		return nil
	})

	// ── 4. Overdue count (distinct members) ───────────────────────────────────
	g.Go(func() error {
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{"gym_id": gymObjID, "status": "overdue"}}},
			{{Key: "$group", Value: bson.M{"_id": "$member_id"}}},
			{{Key: "$count", Value: "total"}},
		}
		cursor, err := s.Payments.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("overdue_count: %w", err)
		}
		defer cursor.Close(gctx)
		var rows []struct {
			Total int64 `bson:"total"`
		}
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("overdue_count decode: %w", err)
		}
		if len(rows) > 0 {
			data.OverdueCount = rows[0].Total
		}
		return nil
	})

	// ── 5. Revenue by month (last 12 months) ──────────────────────────────────
	g.Go(func() error {
		twelveMonthsAgo := now.AddDate(-1, 0, 0)
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"gym_id": gymObjID,
				"status": "paid",
				"paid_at": bson.M{"$gte": twelveMonthsAgo},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id": bson.M{
					"year":  bson.M{"$year": "$paid_at"},
					"month": bson.M{"$month": "$paid_at"},
				},
				"revenue": bson.M{"$sum": "$amount"},
			}}},
			{{Key: "$sort", Value: bson.D{
				{Key: "_id.year", Value: 1},
				{Key: "_id.month", Value: 1},
			}}},
		}
		cursor, err := s.Payments.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("revenue_by_month: %w", err)
		}
		defer cursor.Close(gctx)
		type monthRow struct {
			ID      struct{ Year, Month int } `bson:"_id"`
			Revenue int64                     `bson:"revenue"`
		}
		var rows []monthRow
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("revenue_by_month decode: %w", err)
		}

		// Build a lookup map and fill all 12 months (including zeros).
		lookup := make(map[string]int64, len(rows))
		for _, r := range rows {
			lookup[fmt.Sprintf("%d-%02d", r.ID.Year, r.ID.Month)] = r.Revenue
		}
		result := make([]MonthRevenue, 12)
		for i := 11; i >= 0; i-- {
			t := now.AddDate(0, -i, 0)
			key := fmt.Sprintf("%d-%02d", t.Year(), int(t.Month()))
			result[11-i] = MonthRevenue{Month: key, Revenue: lookup[key]}
		}
		data.RevenueByMonth = result
		return nil
	})

	// ── 6. Attendance by week (last 12 weeks) ─────────────────────────────────
	g.Go(func() error {
		twelveWeeksAgo := now.AddDate(0, 0, -84)
		pipeline := mongo.Pipeline{
			{{Key: "$match", Value: bson.M{
				"gym_id":        gymObjID,
				"checked_in_at": bson.M{"$gte": twelveWeeksAgo},
			}}},
			{{Key: "$group", Value: bson.M{
				"_id": bson.M{
					"year": bson.M{"$isoWeekYear": "$checked_in_at"},
					"week": bson.M{"$isoWeek": "$checked_in_at"},
				},
				"count": bson.M{"$sum": 1},
			}}},
			{{Key: "$sort", Value: bson.D{
				{Key: "_id.year", Value: 1},
				{Key: "_id.week", Value: 1},
			}}},
		}
		cursor, err := s.CheckIns.Aggregate(gctx, pipeline)
		if err != nil {
			return fmt.Errorf("attendance_by_week: %w", err)
		}
		defer cursor.Close(gctx)
		type weekRow struct {
			ID    struct{ Year, Week int } `bson:"_id"`
			Count int64                   `bson:"count"`
		}
		var rows []weekRow
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("attendance_by_week decode: %w", err)
		}

		// Build lookup then fill all 12 weeks (including zeros).
		lookup := make(map[string]int64, len(rows))
		for _, r := range rows {
			lookup[fmt.Sprintf("%d-W%02d", r.ID.Year, r.ID.Week)] = r.Count
		}
		result := make([]WeekAttendance, 12)
		for i := 11; i >= 0; i-- {
			t := now.AddDate(0, 0, -(i * 7))
			y, w := t.ISOWeek()
			key := fmt.Sprintf("%d-W%02d", y, w)
			result[11-i] = WeekAttendance{Week: key, Count: lookup[key]}
		}
		data.AttendanceByWeek = result
		return nil
	})

	// ── 7. Overdue list (up to 10, one per member) ────────────────────────────
	g.Go(func() error {
		findOpts := options.Find().
			SetSort(bson.M{"due_date": 1}).
			SetLimit(20)
		cursor, err := s.Payments.Find(gctx, bson.M{
			"gym_id": gymObjID,
			"status": "overdue",
		}, findOpts)
		if err != nil {
			return fmt.Errorf("overdue_list: %w", err)
		}
		defer cursor.Close(gctx)
		type payRow struct {
			MemberID   string    `bson:"member_id"`
			MemberName string    `bson:"member_name"`
			Amount     int64     `bson:"amount"`
			DueDate    time.Time `bson:"due_date"`
		}
		var rows []payRow
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("overdue_list decode: %w", err)
		}
		seen := make(map[string]bool)
		var list []OverdueMember
		for _, r := range rows {
			if seen[r.MemberID] {
				continue
			}
			seen[r.MemberID] = true
			list = append(list, OverdueMember{
				MemberID:   r.MemberID,
				MemberName: r.MemberName,
				Amount:     r.Amount,
				DueDate:    r.DueDate.Format("2006-01-02"),
			})
			if len(list) == 10 {
				break
			}
		}
		if list == nil {
			list = []OverdueMember{}
		}
		data.OverdueList = list
		return nil
	})

	// ── 8. Pending join requests ──────────────────────────────────────────────
	g.Go(func() error {
		count, err := s.Requests.CountDocuments(gctx, bson.M{
			"gym_id": gymObjID,
			"status": "Pending",
		})
		if err != nil {
			return fmt.Errorf("pending_requests: %w", err)
		}
		data.PendingRequests = count
		return nil
	})

	// ── 9. Upcoming renewals (unpaid, due ≤ 7 days) ───────────────────────────
	g.Go(func() error {
		weekFromNow := now.AddDate(0, 0, 7)
		findOpts := options.Find().
			SetSort(bson.M{"due_date": 1}).
			SetLimit(10)
		cursor, err := s.Payments.Find(gctx, bson.M{
			"gym_id": gymObjID,
			"status": "unpaid",
			"due_date": bson.M{
				"$gte": now,
				"$lte": weekFromNow,
			},
		}, findOpts)
		if err != nil {
			return fmt.Errorf("upcoming_renewals: %w", err)
		}
		defer cursor.Close(gctx)
		type payRow struct {
			MemberID   string    `bson:"member_id"`
			MemberName string    `bson:"member_name"`
			PlanName   string    `bson:"plan_name"`
			Amount     int64     `bson:"amount"`
			DueDate    time.Time `bson:"due_date"`
		}
		var rows []payRow
		if err := cursor.All(gctx, &rows); err != nil {
			return fmt.Errorf("upcoming_renewals decode: %w", err)
		}
		list := make([]UpcomingRenewal, 0, len(rows))
		for _, r := range rows {
			list = append(list, UpcomingRenewal{
				MemberID:   r.MemberID,
				MemberName: r.MemberName,
				PlanName:   r.PlanName,
				Amount:     r.Amount,
				DueDate:    r.DueDate.Format("2006-01-02"),
			})
		}
		data.UpcomingRenewals = list
		return nil
	})

	if err := g.Wait(); err != nil {
		return lambda.ServerError(fmt.Errorf("dashboard aggregation failed: %w", err))
	}

	resp, err := json.Marshal(data)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGetByID, ProcessPost, ProcessPut, ProcessDelete are unused — required by Lambda interface.
func (s *Service) ProcessGetByID(_ context.Context, _ events.APIGatewayProxyRequest, _ string) (events.APIGatewayProxyResponse, error) {
	return lambda.ClientError(http.StatusMethodNotAllowed, "not supported")
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

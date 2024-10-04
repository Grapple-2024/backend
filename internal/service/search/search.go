package search

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"gopkg.in/mgo.v2/bson"
)

type (
	// Service is the object that handles the business logic of all /search operations.
	// Service talks to the underlying Mongo Client (Data access layer) to CRUD announcement objects.
	Service struct {
		*mongoext.Client
		Gyms            *mongo.Collection
		Series          *mongo.Collection
		profilesService *profiles.Service
	}

	searchResponse struct {
		Gyms   []gyms.Gym             `json:"gyms"`
		Series []gym_series.GymSeries `json:"series"`

		TotalGyms   int64 `json:"total_gyms"`
		TotalSeries int64 `json:"total_series"`
		TotalCount  int64 `json:"total_count"`

		// URL to the next page
		NextPage *string `json:"next_page"`

		// URL to the previous page
		PreviousPage *string `json:"previous_page"`
	}

	queryParams struct {
		Page     int
		PageSize int
		GymID    primitive.ObjectID
		Query    string
	}
)

// NewService creates a new instance of a Announcement Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	series := mc.Database("grapple").Collection("series")
	gyms := mc.Database("grapple").Collection("gyms")

	// Create unique index for announcement names
	svc := &Service{
		Client: mc,
		Gyms:   gyms,
		Series: series,
	}
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func parseQueryParams(params map[string]string) (*queryParams, error) {
	// Parse filter query params
	gymID := params["gym_id"]

	// parse pagination query params
	page := params["page"]
	if page == "" {
		page = "1" // default to first page
	}
	pageSize := params["page_size"]
	if pageSize == "" {
		pageSize = "10" // default to 10 records per page
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil && pageSize != "" {
		return nil, err
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return nil, err
	}

	qp := &queryParams{
		Page:     pageInt,
		PageSize: pageSizeInt,
		Query:    params["query"],
	}

	if gymID != "" {
		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			return nil, err
		}
		qp.GymID = gymObjID
	}

	return qp, nil
}

// buildSeriesFilter
func buildSeriesFilter(params *queryParams) bson.M {
	title := params.Query
	// disciplines := params.Discipline
	// difficulties := req.MultiValueQueryStringParameters["difficulty"]
	gymID := params.GymID

	var and []bson.M
	var or []bson.M

	// Gym ID filter
	if gymID != primitive.NilObjectID {
		and = append(and, bson.M{
			"gym_id": params.GymID,
		})
	}

	// Title search with full-text and regex
	if title != "" {
		or = append(or, bson.M{
			"title": bson.M{
				"$regex":   title,
				"$options": "i",
			},
		}, bson.M{
			"videos.title": bson.M{
				"$regex":   title,
				"$options": "i",
			},
		})
	}

	// // Disciplines filter
	// if len(disciplines) > 0 {
	// 	and = append(and, bson.M{
	// 		"videos.disciplines": bson.M{
	// 			"$in": disciplines,
	// 		},
	// 	})
	// }

	// // Difficulties filter
	// if len(difficulties) > 0 {
	// 	and = append(and, bson.M{
	// 		"videos.difficulty": bson.M{
	// 			"$in": difficulties,
	// 		},
	// 	})
	// }

	// Combine filters
	filter := bson.M{}
	if len(and) > 0 {
		filter["$and"] = and
	}
	if len(or) > 0 {
		filter["$or"] = or
	}

	log.Debug().Msgf("Filter: %v", filter)
	return filter
}

// ProcessGetAll handles HTTP requests for GET /search/
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	// parse query params
	params, err := parseQueryParams(req.QueryStringParameters)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest,
			fmt.Sprintf("error parsing query params: %v ", err))
	}

	// build the filter
	gymsFilter := bson.M{}
	if params.Query != "" {
		gymsFilter["name"] = bson.M{
			"$regex":   params.Query,
			"$options": "i",
		}
	}
	if params.GymID != primitive.NilObjectID {
		gymsFilter["gym_id"] = params.GymID
	}
	seriesFilter := buildSeriesFilter(params)

	// Fetch gyms
	var gyms []gyms.Gym
	if err := mongoext.Paginate(ctx, s.Gyms, gymsFilter, params.Page, params.PageSize, true, &gyms); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	log.Info().Msgf("gyms: %v", gyms)
	log.Info().Msgf("gyms filter: %v", gymsFilter)

	// Fetch Series
	var series []gym_series.GymSeries
	if err := mongoext.Paginate(ctx, s.Series, seriesFilter, params.Page, params.PageSize, true, &series); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	log.Info().Msgf("series: %v", series)
	log.Info().Msgf("series filter: %v", seriesFilter)

	// Get the total count of gyms
	totalGyms, err := s.Gyms.CountDocuments(ctx, gymsFilter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	// Get the total count of series
	totalSeries, err := s.Series.CountDocuments(ctx, seriesFilter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp := searchResponse{
		Gyms:        gyms,
		Series:      series,
		TotalGyms:   totalGyms,
		TotalSeries: totalSeries,
		TotalCount:  totalGyms + totalSeries,
	}
	n := float64(resp.TotalCount) / float64(params.PageSize)
	totalPages := int64(math.Ceil(n))
	// if we're not on the last page, add the next page's URL to the response.
	if totalPages > int64(params.Page) {
		nextPageURL := fmt.Sprintf("%s/search/?pageSize=%d&page=%d", service.API_URL, params.PageSize, params.Page+1)
		resp.NextPage = &nextPageURL
	}

	// if we're not on the first page, add the previous page's URL to the response.
	if params.Page > 1 && totalPages >= int64(params.Page) {
		prevPageURL := fmt.Sprintf("%s/search/?pageSize=%d&page=%d", service.API_URL, params.PageSize, params.Page-1)
		resp.PreviousPage = &prevPageURL
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error marshalling response: %v", err))

	}
	return lambda.NewResponse(http.StatusOK, string(respBytes), nil), nil
}

func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPost handles HTTP requests for POST /announcements
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

// ensureIndices TODO.
func (s *Service) ensureIndices(ctx context.Context) error {
	return nil
}

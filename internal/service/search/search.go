package search

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type (
	// Service is the object that handles the business logic of all /search operations.
	// Service talks to the underlying Mongo Client (Data access layer) to CRUD announcement objects.
	Service struct {
		*rbac.RBAC
		*mongoext.Client
		Gyms            *mongo.Collection
		Series          *mongo.Collection
		profilesService *profiles.Service
	}

	searchResponse struct {
		Gyms   []dao.Gym              `json:"gyms"`
		Series []gym_series.GymSeries `json:"series"`

		// total counts across all pages
		TotalGyms  int64 `json:"total_gyms"`
		TotalCount int64 `json:"total_count"`

		// Per page counts
		SeriesCount int64 `json:"series_count"`
		GymsCount   int64 `json:"gyms_count"`

		// URL to the next page
		NextPage *string `json:"next_page"`

		// URL to the previous page
		PreviousPage *string `json:"previous_page"`
	}

	queryParams struct {
		Page     int
		PageSize int
		GymID    bson.ObjectID
		Query    string
	}
)

// NewService creates a new instance of a Announcement Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	series := mc.Database("grapple").Collection("series")
	gyms := mc.Database("grapple").Collection("gyms")

	// Create unique index for announcement names
	svc := &Service{
		Client: mc,
		Gyms:   gyms,
		Series: series,
		RBAC:   rbac,
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
		gymObjID, err := bson.ObjectIDFromHex(gymID)
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
	if gymID != bson.NilObjectID {
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

	return filter
}

// ProcessGetAll handles HTTP requests for GET /search/
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("permission denied: %v", err))
	}

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
	if params.GymID != bson.NilObjectID {
		gymsFilter["gym_id"] = params.GymID
	}
	seriesFilter := buildSeriesFilter(params)

	// Fetch gyms
	var gymsToReturn []dao.Gym
	if err := mongoext.Paginate(ctx, s.Gyms, gymsFilter, params.Page, params.PageSize, true, options.Find(), &gymsToReturn); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}

	// Fetch Series
	var series []gym_series.GymSeries
	if err := mongoext.Paginate(ctx, s.Series, seriesFilter, params.Page, params.PageSize, true, options.Find(), &series); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	log.Info().Msgf("Found total of %d series that match filter %+v", len(series), seriesFilter)

	// Filter only series that the user has permission to view
	seriesToReturn := []gym_series.GymSeries{}
	for _, gymSeries := range series {
		resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, gymSeries.GymID.Hex(), rbac.ResourceSeries)
		isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionCreate)
		if err != nil || !isAuthorized {
			if err != nil {
				log.Error().Err(err).Msgf("Error determining authorization of user %s on resource %s", token.Sub, resourceID)
			}
			continue
		}

		seriesToReturn = append(seriesToReturn, gymSeries)
	}

	// Get the total count of gyms
	totalGyms, err := s.Gyms.CountDocuments(ctx, gymsFilter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp := searchResponse{
		Gyms:        gymsToReturn,
		Series:      seriesToReturn,
		TotalGyms:   totalGyms,
		TotalCount:  totalGyms + int64(len(seriesToReturn)),
		SeriesCount: int64(len(seriesToReturn)),
		GymsCount:   int64(len(gymsToReturn)),
	}
	n := float64(resp.TotalCount) / float64(params.PageSize)
	totalPages := int64(math.Ceil(n))
	// if we're not on the last page, add the next page's URL to the response.
	if totalPages > int64(params.Page) {
		nextPageURL := fmt.Sprintf("%s/search/?page_size=%d&page=%d", service.API_URL, params.PageSize, params.Page+1)
		resp.NextPage = &nextPageURL
	}

	// if we're not on the first page, add the previous page's URL to the response.
	if params.Page > 1 && totalPages >= int64(params.Page) {
		prevPageURL := fmt.Sprintf("%s/search/?page_size=%d&page=%d", service.API_URL, params.PageSize, params.Page-1)
		resp.PreviousPage = &prevPageURL
	}

	if params.Query != "" {
		if resp.NextPage != nil {
			*resp.NextPage += fmt.Sprintf("&query=%s", params.Query)
		}
		if resp.PreviousPage != nil {
			*resp.PreviousPage += fmt.Sprintf("&query=%s", params.Query)

		}
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

package gym_series

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"gopkg.in/mgo.v2/bson"
)

// Service is the object that handles the business logic of all gymSeries related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gymSeries objects.
type Service struct {
	mongo.Session

	*mongoext.Client
	*mongo.Collection
}

// NewService creates a new instance of a GymSeries Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("series")

	// Create Mongo Session (needed for transactions)
	svc := &Service{Client: mc, Collection: c}
	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /gym-requests/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, _ map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// Parse filter query params
	showByWeek := req.QueryStringParameters["show_by_week"]
	gymID := req.QueryStringParameters["gym_id"]

	// parse pagination query params
	page := req.QueryStringParameters["page"]
	if page == "" {
		page = "1" // default to first page
	}
	pageSize := req.QueryStringParameters["page_size"]
	if pageSize == "" {
		pageSize = "10" // default to 10 records per page
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil && pageSize != "" {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &pageSize query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// create the filter based on query parameters in the request
	filter := bson.M{}
	if gymID != "" {
		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
		}
		filter["gym_id"] = gymObjID
	}

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
		}
		year, week := time.ISOWeek()
		filter["created_at_year"] = year
		filter["created_at_week"] = week
	}

	// Fetch records with pagination
	var records []GymSeries
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []GymSeries{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gymSeries", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gym-series/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gymSeries by ID
	var gymSeries GymSeries
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymSeries); err != nil {
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymSeries by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(gymSeries)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gym-series
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymSeries GymSeries
	if err := json.Unmarshal([]byte(req.Body), &gymSeries); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(gymSeries); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	gymSeries.CreatedAt = time.Now().Local().UTC()
	gymSeries.UpdatedAt = gymSeries.CreatedAt

	// insert the GymSeries, store the resulting record in 'result' variable
	var result GymSeries
	if err := mongoext.Insert(ctx, s.Collection, &gymSeries, &result); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /gym-series/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymSeries GymSeries
	if err := json.Unmarshal([]byte(req.Body), &gymSeries); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// update the record in mongo
	id := req.PathParameters["id"]
	var result GymSeries
	if err := mongoext.UpdateByID(ctx, s.Collection, id, gymSeries, &result, nil); err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gym-series/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	// create filter and options
	filter := bson.M{"_id": objID}
	opts := options.Delete().SetHint(bson.M{"_id": 1}) // use _id index to find object

	result, err := s.Collection.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	if result.DeletedCount == 0 {
		return lambda_v2.NewResponse(http.StatusNotFound, ``, nil), nil
	}

	return lambda_v2.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) updateSeriesTransaction(ctx context.Context, payload *GymSeries, id string) (*GymSeries, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var result GymSeries
		if err := mongoext.UpdateByID(ctx, s.Collection, id, payload, &result, nil); err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}
		log.Info().Msgf("Update GymSeries result: %v", result)

		return result, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("createProfile transaction completed successfully!")
	}

	return result.(*GymSeries), nil
}

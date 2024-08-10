package gyms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

// Service is the object that handles the business logic of all gym related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gym objects.
type Service struct {
	*mongoext.Client
	*mongo.Collection
}

// NewService creates a new instance of a Gym Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("gyms")

	// Create unique index for gyms collection
	svc := &Service{Client: mc, Collection: c}
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return &Service{Client: mc, Collection: c}, nil
}

// ProcessGetAll handles HTTP requests for GET /gyms/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, _ map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters
	gymSlug := req.QueryStringParameters["slug"]
	creatorID := req.QueryStringParameters["creator_id"]

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
	if gymSlug != "" {
		filter["slug"] = gymSlug
	}
	if creatorID != "" {
		filter["creator"] = creatorID
	}

	// Fetch records with pagination
	var records []Gym
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Gym{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gyms", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gyms/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gym by ID
	var gym Gym
	if err := mongoext.FindByID(ctx, s.Collection, id, &gym); err != nil {
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gym by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(gym)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gyms
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gym Gym
	if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	validate.RegisterValidation("alphanumeric_and_spaces", service.IsAlphaNumericAndSpaces)
	validate.RegisterValidation("is_state", service.IsState)

	if err := validate.Struct(gym); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	// Remove all special characters from the gym name (paranthesis mainly)
	citySlug := strings.ToLower(strings.ReplaceAll(gym.City, " ", "-"))
	stateSlug := strings.ToLower(strings.ReplaceAll(gym.State, " ", "-"))
	gymNameSlug := strings.ToLower(strings.ReplaceAll(gym.Name, " ", "-"))

	// Set computed fields for slug, created_at, and updated_at
	gym.Slug = fmt.Sprintf("%s/%s/%s", stateSlug, citySlug, gymNameSlug)
	gym.CreatedAt = time.Now().Local().UTC()
	gym.UpdatedAt = gym.CreatedAt

	// insert the announcement, store the resulting record in 'result' variable
	var result Gym
	if err := mongoext.Insert(ctx, s.Collection, &gym, &result); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert record: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /gyms/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gym Gym
	if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// update the record in mongo
	id := req.PathParameters["id"]
	var result Gym
	if err := mongoext.UpdateByID(ctx, s.Collection, id, gym, &result, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gyms/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	if err := mongoext.Delete(ctx, s.Collection, id); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to delete gym record: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

// ensureIndices ensures the proper indices are creatd for the 'gyms' collection.
func (s *Service) ensureIndices(ctx context.Context) error {
	// Gym name index
	_, err := s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"name": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}

	// Slug index
	_, err = s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"slug": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	return nil
}

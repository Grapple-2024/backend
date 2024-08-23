package gym_requests

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/profiles"
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

// Service is the object that handles the business logic of all gymRequest related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gymRequest objects.
type Service struct {
	mongo.Session

	*mongoext.Client
	*mongo.Collection
}

// NewService creates a new instance of a GymRequest Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("gymRequests")

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
	gymStatus := req.QueryStringParameters["status"]
	requestorID := req.QueryStringParameters["requestor"]

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

	if gymStatus != "" {
		filter["status"] = gymStatus
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

	if requestorID != "" {
		filter["requestor_id"] = requestorID
	}

	// Fetch records with pagination
	var records []GymRequest
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, true, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []GymRequest{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gymRequests", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gymRequests/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gymRequest by ID
	var gymRequest GymRequest
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymRequest); err != nil {
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymRequest by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(gymRequest)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gymRequests
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymRequest GymRequest
	if err := json.Unmarshal([]byte(req.Body), &gymRequest); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(gymRequest); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	gymRequest.CreatedAt = time.Now().Local().UTC()
	gymRequest.UpdatedAt = gymRequest.CreatedAt

	// insert the GymRequest, store the resulting record in 'result' variable
	var result GymRequest
	if err := mongoext.Insert(ctx, s.Collection, &gymRequest, &result); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /gymRequests/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymRequest GymRequest
	if err := json.Unmarshal([]byte(req.Body), &gymRequest); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// update the record in mongo
	id := req.PathParameters["id"]
	result, err := s.updateGymRequestTX(ctx, &gymRequest, id)
	if err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to finish updateGymRequest transaction: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gymRequests/{id}
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

func (s *Service) updateGymRequestTX(ctx context.Context, payload *GymRequest, id string) (*GymRequest, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var request GymRequest
		request.UpdatedAt = time.Now().Local().UTC()

		if err := mongoext.UpdateByID(ctx, s.Collection, id, payload, &request, nil); err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

		// return early if the request was not approved
		if payload.Status != RequestApproved {
			return request, nil
		}

		// The request was approved by the coach: update the student profile's gym_associations field.
		log.Debug().Msgf("a gym request was approved by coach for student %q (%s)", request.RequestorEmail, request.RequestorID)

		// create new profile
		gymAssociation := profiles.GymAssociation{
			CoachName: "TODO",
			GymID:     request.GymID,
			Role:      "Student",
			EmailPreferences: &profiles.EmailPreferences{
				NotifyOnAnnouncements: true,
			},
		}

		// create filter & update statements, send to mongodb to update the student's profile.
		filter := bson.M{
			"cognito_id": request.RequestorID,
		}
		update := bson.M{
			"$push": bson.M{
				"gyms": gymAssociation,
			},
		}

		// Update student profile with the new gym association
		var upsertResult profiles.Profile
		coll := s.Client.Database("grapple").Collection("profiles")
		if err := mongoext.Update(ctx, coll, update, filter, &upsertResult, nil); err != nil {
			return nil, fmt.Errorf("failed to upsert student's profile with filter %v after creating a gym request: %v", filter, err)
		}

		log.Info().Msgf("Successfully added gym association to user profile: %s", request.RequestorID)
		return request, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("createProfile transaction completed successfully!")
	}

	if request, ok := result.(GymRequest); ok {
		return &request, nil
	}

	return nil, err
}

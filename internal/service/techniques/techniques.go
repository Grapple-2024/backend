package techniques

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/gym_series"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"gopkg.in/mgo.v2/bson"
)

// Service is the object that handles the business logic of all technique related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD technique objects.
type Service struct {
	mongo.Session

	*mongoext.Client
	*mongo.Collection
	*s3.PresignClient

	videosBucketName string
}

// NewService creates a new instance of a Technique Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, videosBucketName, region string) (*Service, error) {
	c := mc.Database("grapple").Collection("techniques")

	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	svc := &Service{
		Client:           mc,
		Collection:       c,
		videosBucketName: videosBucketName,
		PresignClient:    s3.NewPresignClient(s3.NewFromConfig(cfg)),
	}

	// Create unique index for technique names
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	// Create Mongo Session (needed for transactions)
	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /techniques/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	// parse filter query parameters
	gymID := req.QueryStringParameters["gym_id"]
	showByWeek := req.QueryStringParameters["show_by_week"]

	// parse pagination query parameters
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
		return lambda.ClientError(http.StatusBadRequest, "invalid &page_size query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// create the filter based on query parameters in the request
	filter := bson.M{}
	if gymID != "" {
		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
		}
		filter["series.gym_id"] = gymObjID
	}

	loc, err := time.LoadLocation("America/Los_Angeles")
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to load location: %v", err))
	}

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
		}
		year, week := time.In(loc).ISOWeek()
		filter["year_number"] = year
		filter["week_number"] = week
	}

	// Fetch records with pagination
	var records []Technique
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Technique{}
	}

	if err := s.generatePresignedURLs(ctx, records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("techniques", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /techniques/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the technique by ID
	var technique Technique
	if err := mongoext.FindByID(ctx, s.Collection, id, &technique); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find technique by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(technique)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /techniques
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var technique Technique
	if err := json.Unmarshal([]byte(req.Body), &technique); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(technique); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	now := time.Now().Local().UTC()
	technique.CreatedAt, technique.UpdatedAt = now, now
	technique.DisplayYearNum, technique.DisplayWeekNum = technique.DisplayOnWeek.ISOWeek()

	// insert the technique, store the resulting record in 'result' variable
	result, err := s.createTechnique(ctx, &technique)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /techniques/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var technique Technique
	if err := json.Unmarshal([]byte(req.Body), &technique); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// update the record in mongo
	id := req.PathParameters["id"]
	var result Technique
	if err := mongoext.UpdateByID(ctx, s.Collection, id, technique, &result, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /techniques/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	// create filter and options
	filter := bson.M{"_id": objID}
	opts := options.Delete().SetHint(bson.M{"_id": 1}) // use _id index to find object

	result, err := s.Collection.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return lambda.ServerError(err)
	}

	if result.DeletedCount == 0 {
		return lambda.NewResponse(http.StatusNotFound, ``, nil), nil
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) createTechnique(ctx context.Context, t *Technique) (*Technique, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
		// Fetch the series that is being marked as a technique of the week
		var series gym_series.GymSeries
		seriesCollection := s.Client.Database("grapple").Collection("series")
		if err := mongoext.FindByID(sessCtx, seriesCollection, t.Series.ID.Hex(), &series); err != nil {
			return nil, err
		}

		// Insert the technique with the series nested within it
		var result Technique
		t.Series = &series
		if err := mongoext.Insert(sessCtx, s.Collection, t, &result); err != nil {
			return nil, err
		}
		return &result, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to run mongo transaction for technique creation")
		return nil, err
	}

	t, ok := result.(*Technique)
	if !ok {
		return nil, fmt.Errorf("result is not of *technique type: %+v", result)
	}
	return result.(*Technique), nil
}

// ensureIndices ensures the proper indices are created for the 'techniques' collection.
func (s *Service) ensureIndices(ctx context.Context) error {
	// Gym name index
	_, err := s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"gym_id": 1,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

// generatePresignedURLs generates presigned URL for each video in the records slice.
// It modifies the records slice by reference and returns an error
func (s *Service) generatePresignedURLs(ctx context.Context, records []Technique) error {
	for i, record := range records {
		if record.Series == nil {
			continue
		}
		for j, video := range record.Series.Videos {
			p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.videosBucketName, "download", video.S3ObjectKey)
			if err != nil {
				return fmt.Errorf("failed to generate presigned url: %v", err)
			}
			records[i].Series.Videos[j].PresignedURL = p.URL
		}
	}
	return nil
}

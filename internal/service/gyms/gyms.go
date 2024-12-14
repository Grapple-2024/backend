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
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"gopkg.in/mgo.v2/bson"
)

// Service is the object that handles the business logic of all gym related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gym objects.
type Service struct {
	mongo.Session

	*s3.PresignClient
	*mongoext.Client
	*mongo.Collection
	publicAssetsBucketName string
	region                 string
}

// NewService creates a new instance of a Gym Service given a mongo client
func NewService(ctx context.Context, publicAssetsBucketName, region string, mc *mongoext.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("gyms")

	// Create unique index for gyms collection
	svc := &Service{Client: mc, Collection: c}
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}

	return &Service{
		Session:                session,
		Client:                 mc,
		Collection:             c,
		PresignClient:          s3.NewPresignClient(s3.NewFromConfig(cfg)),
		region:                 region,
		publicAssetsBucketName: publicAssetsBucketName,
	}, nil
}

// ProcessGetAll handles HTTP requests for GET /gyms/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	// Parse query parameters
	gymSlug := req.QueryStringParameters["slug"]
	creatorID := req.QueryStringParameters["creator_id"]
	name := req.QueryStringParameters["name"]

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
	if gymSlug != "" {
		filter["slug"] = gymSlug
	}
	if name != "" {
		filter["name"] = bson.M{
			"$regex":   name,
			"$options": "i",
		}
	}
	if creatorID != "" {
		filter["creator"] = creatorID
	}

	// Fetch records with pagination
	var records []Gym
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Gym{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gyms", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gyms/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gym by ID
	var gym Gym
	if err := mongoext.FindByID(ctx, s.Collection, id, &gym); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gym by ID: %v", err))
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
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var gym Gym
	if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	result, err := s.createGymTX(ctx, token, &gym)
	if err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to finish updateGymRequest transaction: %v", err))
	}

	log.Info().Msgf("Create gym result: %v", result)
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for
// 1. PUT /gyms - insert/update a gym object
// 2. PUT /gyms/presign - generate presigned upload url for gym logo/banner/hero
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var result any

	// update the record in mongo
	id := req.PathParameters["id"]

	gymSubPath := fmt.Sprintf("/gyms/%s", id)
	switch req.Path {
	case gymSubPath:
		var gym Gym
		if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}

		// update the record in mongo
		if err := mongoext.UpdateByID(ctx, s.Collection, id, gym, &result, nil); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

	case fmt.Sprintf("%s/presign", gymSubPath):
		fileType := req.QueryStringParameters["type"] // should be banner or logo or hero, but can be anything
		file := req.QueryStringParameters["file"]
		_, err := service.GetToken(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
		}
		if file == "" {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("?file cannot be empty"))
		}

		// generate presigned avatar upload url
		key := fmt.Sprintf("gyms/%s/%s/%s", id, fileType, file)
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.publicAssetsBucketName, "upload", key)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to generate presigned upload url: %v", err))
		}

		s3ObjectURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.publicAssetsBucketName, s.region, key)
		resp := struct {
			*v4.PresignedHTTPRequest
			S3ObjectURL string `json:"s3_object_url"`
		}{
			PresignedHTTPRequest: p,
			S3ObjectURL:          s3ObjectURL,
		}

		result = resp
	default:
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid request path: %v", req.Path))
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

func (s *Service) createGym(ctx context.Context, gym *Gym, token *service.Token) (*Gym, error) {
	gym.CoachFirstName = token.GivenName
	gym.CoachLastName = token.FamilyName
	gym.Creator = token.Sub

	// Validate request body for required fields
	validate := validator.New()
	validate.RegisterValidation("alphanumeric_and_spaces", service.IsAlphaNumericAndSpaces)
	validate.RegisterValidation("is_state", service.IsState)

	if err := validate.Struct(gym); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return nil, err
	}

	// Remove all special characters from the gym name (paranthesis mainly)
	citySlug := strings.ToLower(strings.ReplaceAll(gym.City, " ", "-"))
	stateSlug := strings.ToLower(strings.ReplaceAll(gym.State, " ", "-"))
	gymNameSlug := strings.ToLower(strings.ReplaceAll(gym.Name, " ", "-"))

	// Set computed fields for slug, created_at, and updated_at
	gym.Slug = fmt.Sprintf("state/%s/city/%s/gym/%s", stateSlug, citySlug, gymNameSlug)
	gym.CreatedAt = time.Now().Local().UTC()
	gym.UpdatedAt = gym.CreatedAt

	// insert the gym, store the resulting record in 'result' variable
	var result Gym
	if err := mongoext.Insert(ctx, s.Collection, &gym, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *Service) createGymTX(ctx context.Context, token *service.Token, payload *Gym) (*Gym, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
		// create the gym
		gym, err := s.createGym(ctx, payload, token)
		if err != nil {
			return nil, fmt.Errorf("failed to create gym: %v", err)
		}

		// create new gym association for this Gym Owner
		gymAssociation := profiles.GymAssociation{
			CoachName: fmt.Sprintf("%s %s", gym.CoachFirstName, gym.CoachLastName),
			Email:     gym.CoachEmail,
			GymID:     gym.ID,
			Role:      profiles.OwnerRole,
			EmailPreferences: &profiles.EmailPreferences{
				NotifyOnAnnouncements: false,
				NotifyOnRequests:      true,
			},
		}

		log.Info().Msgf("Adding owner gym association to profile: %v", gymAssociation)

		// create filter & update statements, send to mongodb to update the student's profile.
		filter := bson.M{
			"cognito_id": gym.Creator, // find the profile with cognito_id equal to the creator of the Gym.
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
			return nil, fmt.Errorf("failed to upsert student's profile with filter %v: %v", filter, err)
		}

		log.Info().Msgf("Successfully added gym association to user profile: %s", payload.Creator)
		return *gym, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("failed to run mongo transaction for gym creation")
		return nil, err
	}
	log.Info().Msgf("createGym transaction completed successfully: %v", result)

	if request, ok := result.(Gym); ok {
		return &request, nil
	}

	return nil, err
}

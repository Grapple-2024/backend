package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"gopkg.in/mgo.v2/bson"
)

// Service is the object that handles the business logic of all Profile related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD Profile objects.
type Service struct {
	*mongoext.Client
	*mongo.Collection
	mongo.Session
	*s3.PresignClient

	publicAssetsBucketName string

	CognitoClient *cognito.Client
}

// NewService creates a new instance of a Profile Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, publicAssetsBucketName, region, cognitoClientID, cognitoClientSecret string) (*Service, error) {
	c := mc.Database("grapple").Collection("profiles")

	cc, err := cognito.NewClient(
		region,
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		return nil, err
	}
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	svc := &Service{
		Client:                 mc,
		Collection:             c,
		CognitoClient:          cc,
		PresignClient:          s3.NewPresignClient(s3.NewFromConfig(cfg)),
		publicAssetsBucketName: publicAssetsBucketName,
	}

	// Create Mongo Session (needed for transactions)
	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /profiles/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, _ map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// build query filter if current_user=true
	var filter bson.M
	currentUser := req.QueryStringParameters["current_user"]
	if currentUser == "true" {
		token, err := service.GetToken(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
		}
		// create the filter based on query parameters in the request
		filter = bson.M{
			"cognito_id": token.Sub,
		}
	}
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
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &pageSize query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// Fetch records with pagination
	var records []Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Profile{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	var result []byte
	if req.QueryStringParameters["current_user"] == "true" {
		// marshal response as single profile object for easier frontend consumption
		resp, err := json.Marshal(records[0])
		if err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to marshal current user profile to json: %v", err))
		}
		result = resp
	} else {
		// marshal response as array
		result, err = service.NewGetAllResponse("profiles", records, totalCount, len(records), pageInt, pageSizeInt)
		if err != nil {
			return lambda_v2.ServerError(err)
		}
	}
	return lambda.NewResponse(http.StatusOK, string(result), nil), nil
}

// ProcessGet handles HTTP requests for GET /profiles/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the profile by ID
	// var profile Profile
	// if err := mongoext.FindByID(ctx, s.Collection, id, &profile); err != nil {
	// 	return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find profile by ID: %v", err))
	// }

	// // Return record as JSON
	// json, err := json.Marshal(profile)
	// if err != nil {
	// 	return lambda.ServerError(err)
	// }
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPost is a no-operation. Updating and inserting a profile is handled via PUT /profiles
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPut handles HTTP requests for
// 1. PUT /profiles - insert/update a profile document
// 2. PUT /profiles/avatar - generate presigned upload url for profile avatar
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var result any
	switch req.Path {
	case "/profiles/avatar":
		// generate presigned avatar upload url
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.publicAssetsBucketName, "upload", "avatar.png")
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to generate presigned upload url: %v", err))
		}

		result = p
	case "/profiles":
		var profile Profile
		if err := json.Unmarshal([]byte(req.Body), &profile); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}
		log.Debug().Msgf("Updating user profile with payload: %v", profile)

		// If request body contains an ID: we are doing and update
		// Else: we are doing a create operation (generate a new object ID)
		var id string
		if profile.ID != primitive.NilObjectID {
			id = profile.ID.Hex()
			profile.ID = primitive.NilObjectID
		} else {
			id = primitive.NewObjectID().Hex()
			profile.CreatedAt = time.Now().Local().UTC()
		}

		// update the record in mongo, store the result in "result" variable.
		var p Profile
		opts := options.Update().SetUpsert(true) // allow for upserts
		if err := mongoext.UpdateByID(ctx, s.Collection, id, profile, &p, opts); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

		result = p
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

// ProcessDelete handles HTTP requests for DELETE /profiles/{id}
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

func (s *Service) createProfile(ctx context.Context, p *Profile) (*Profile, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var result Profile
		if err := mongoext.Insert(sessCtx, s.Collection, p, &result); err != nil {
			return nil, err
		}
		log.Info().Msgf("Insert result: %v", result)

		return result, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("createProfile transaction completed successfully!")
	}

	return result.(*Profile), nil
}

// ensureIndices ensures the proper indices are creatd for the 'gyms' collection.
func (s *Service) ensureIndices(ctx context.Context) error {
	// Cognito ID index
	_, err := s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"cognito_id": 1,
		},
		Options: options.Index().SetUnique(true),
	})
	if err != nil {
		return err
	}
	return nil
}

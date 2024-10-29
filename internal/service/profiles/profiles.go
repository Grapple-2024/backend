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
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/service/cognitoidentityprovider"
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
	awsRegion     string
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
		awsRegion:              region,
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
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
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
		return lambda.ClientError(http.StatusBadRequest, "invalid &page_size query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// Fetch records with pagination
	log.Info().Msgf("Filter: %v", filter)

	var records []Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Profile{}
		resp, err := json.Marshal(records)
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to marshal current user profiles to json: %v", err))
		}
		return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	log.Info().Msgf("Got records: %v", records)

	var result []byte
	if req.QueryStringParameters["current_user"] == "true" {
		// marshal response as single profile object for easier frontend consumption
		resp, err := json.Marshal(records[0])
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to marshal current user profile to json! %v", err))
		}
		result = resp
	} else {
		// marshal response as array
		result, err = service.NewGetAllResponse("profiles", records, totalCount, len(records), pageInt, pageSizeInt)
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to create response: %v", err))
		}
	}
	return lambda.NewResponse(http.StatusOK, string(result), nil), nil
}

// ProcessGet handles HTTP requests for GET /profiles/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the profile by ID
	// var profile Profile
	// if err := mongoext.FindByID(ctx, s.Collection, id, &profile); err != nil {
	// 	return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find profile by ID: %v", err))
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
		token, err := service.GetToken(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
		}

		file := req.QueryStringParameters["file"]
		// generate presigned avatar upload url
		key := fmt.Sprintf("%s/%s", token.Sub, file)
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.publicAssetsBucketName, "upload", key)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to generate presigned upload url: %v", err))
		}

		s3ObjectURL := fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s.publicAssetsBucketName, "us-west-1", key)
		resp := struct {
			*v4.PresignedHTTPRequest
			S3ObjectURL string `json:"s3_object_url"`
		}{
			PresignedHTTPRequest: p,
			S3ObjectURL:          s3ObjectURL,
		}

		result = resp
	case "/profiles":
		// Update user profile based on token (cognito ID)
		token, err := service.GetToken(req.Headers)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to authorize user: %v", err))
		}

		var payload Profile
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}

		payload.UpdatedAt = time.Now().Local().UTC()

		// update the record in mongo, store the result in "result" variable.
		filter := bson.M{
			"cognito_id": token.Sub,
		}
		update := bson.M{
			"$set": payload,
		}

		var p Profile
		if err := mongoext.Update(ctx, s.Collection, update, filter, &p, nil); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

		log.Info().Msgf("Profile: %v", p)
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

// ProcessDelete handles HTTP requests for DELETE /profiles/{id}.
// This endpoint will delete data in multiple collections to ensure full cleanup of a user's data. Use with caution.
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]

	if err := s.deleteUser(ctx, id); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to delete profile with ID %q: %v", id, err))
	}
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) deleteUser(ctx context.Context, profileID string) error {

	// get the profile
	var profile Profile
	if err := mongoext.FindByID(ctx, s.Collection, profileID, &profile); err != nil {
		return fmt.Errorf("failed to find profile with id %s: %v", profileID, err)
	}

	// find all gym requests for this profile and delete them
	filter := bson.M{
		"requestor_id": profile.CognitoID,
	}
	gymRequestsColl := s.Database().Collection("gymRequests")
	res, err := gymRequestsColl.DeleteMany(ctx, filter, nil)
	if err != nil {
		return fmt.Errorf("failed to delete gym requests for cognito user %q: %v", profile.CognitoID, err)
	}
	log.Info().Msgf("Delete gym requests count: %v", res.DeletedCount)

	out, err := s.CognitoClient.AdminDeleteUser(&cognitoidentityprovider.AdminDeleteUserInput{
		Username: &profile.Email,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user %q in cognito: %v", profile.Email, err)
	}
	log.Info().Msgf("Delete Cognito User Result: %v", out.String())

	return nil
}

// GetGymAssociationsBy returns all gym associations with the specified Gym ID and role (Student or Coach).
func (s *Service) GetGymAssociationsBy(ctx context.Context, gymID string, role string) ([]GymAssociation, error) {
	gymObjID, err := primitive.ObjectIDFromHex(gymID)
	if err != nil {
		return nil, err
	}

	// get all coaches for this gym
	filter := bson.M{
		"gyms": bson.M{
			"$elemMatch": bson.M{
				"gym_id": gymObjID,
				"role":   role,
			},
		},
	}
	var profiles []Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, 1, 1000, true, &profiles); err != nil {
		return nil, fmt.Errorf("could not find any profiles that have a %s role with gym id %q %v", role, gymID, err)
	}

	var gymAssociations []GymAssociation
	for _, p := range profiles {
		for _, g := range p.Gyms {
			if g.GymID != gymObjID || g.Role != role {
				continue
			}

			gymAssociations = append(gymAssociations, g)
		}
	}

	return gymAssociations, nil
}

// GetAllStudents returns a slice of gym IDs associated with the cognito token. The associations can either be Student or Coach.
func GetGymsOf(ctx context.Context, collection *mongo.Collection, cognitoID string) ([]primitive.ObjectID, error) {
	filter := bson.M{
		"cognito_id": cognitoID,
	}

	var profile Profile
	if err := mongoext.Find(ctx, collection, filter, &profile); err != nil {
		return nil, fmt.Errorf("could not find any profiles with cognito ID %q: %v", cognitoID, err)
	}
	log.Info().Msgf("Found profile, fetching gyms: %v", profile)

	var gymIDs []primitive.ObjectID
	for _, g := range profile.Gyms {
		gymIDs = append(gymIDs, g.GymID)
	}
	return gymIDs, nil
}

// GetAllStudents returns true if the Cognito ID is a student of the specified Gym ID.
func (s *Service) IsStudentOf(ctx context.Context, cognitoID, gymID string) (bool, error) {
	gymObjID, err := primitive.ObjectIDFromHex(gymID)
	if err != nil {
		return false, err
	}

	// find the profile if it exists
	filter := bson.M{
		"cognito_id": cognitoID,
		"gyms": bson.M{
			"$elemMatch": bson.M{
				"gym_id": gymObjID,
				"role":   StudentRole,
			},
		},
	}
	var profile Profile
	if err := mongoext.Find(ctx, s.Collection, filter, &profile); err != nil {
		return false, fmt.Errorf("could not find any profiles that have a student role with gym id %q %v", gymID, err)
	}

	if profile.CognitoID == cognitoID {
		return true, nil
	}
	return false, nil
}

// GetAllStudents returns all student profiles associated with a specific gym.
func (s *Service) GetStudentsOf(ctx context.Context, gymID string) ([]Profile, error) {
	gymObjID, err := primitive.ObjectIDFromHex(gymID)
	if err != nil {
		return nil, err
	}

	// get all coaches for this gym
	filter := bson.M{
		"gyms": bson.M{
			"$elemMatch": bson.M{
				"gym_id": gymObjID,
				"role":   StudentRole,
			},
		},
	}
	var profiles []Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, 1, 1000, true, &profiles); err != nil {
		return nil, fmt.Errorf("could not find any profiles that have a student role in gym id %q %v", gymID, err)
	}

	return profiles, nil
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

package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// Service is the object that handles the business logic of all Profile related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD Profile objects.
type Service struct {
	*mongoext.Client
	*mongo.Collection
	*mongo.Session
	*s3.PresignClient

	publicAssetsBucketName string

	CognitoClient *cognito.Client
	awsRegion     string
}

// NewService creates a new instance of a Profile Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, publicAssetsBucketName, region string, cognitoClient *cognito.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("profiles")

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
		CognitoClient:          cognitoClient,
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
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// build query filter if current_user=true
	var filter bson.M
	currentUser := req.QueryStringParameters["current_user"]
	if currentUser == "true" {
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

	var records []dao.Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, options.Find(), &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}

	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []dao.Profile{}
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

	var result []byte
	if req.QueryStringParameters["current_user"] == "true" {
		profile := records[0]

		// marshal response as single profile object for easier frontend consumption
		resp, err := json.Marshal(profile)
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

		var payload dao.Profile
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

		var p dao.Profile
		if err := mongoext.UpdateOne(ctx, s.Collection, update, filter, &p, nil); err != nil {
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
	var profile dao.Profile
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

	// out, err := s.CognitoClient.AdminDeleteUser(&cognitoidentityprovider.AdminDeleteUserInput{
	// 	Username: &profile.Email,
	// })
	// if err != nil {
	// 	return fmt.Errorf("failed to delete user %q in cognito: %v", profile.Email, err)
	// }
	// log.Info().Msgf("Delete Cognito User Result: %v", out.String())

	return nil
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

// UpsertGymAssociation upserts (inserts or updates) a gym association to a user profile object.
// It does not update any groups in Cognito or the RBAC framework.
func UpsertGymAssociation(ctx context.Context, mc *mongoext.Client, gym *dao.Gym, groupName, username string) error {
	gymAssociation := dao.GymAssociation{
		Gym:   gym,
		Email: username,
		Group: groupName,
		EmailPreferences: &dao.EmailPreferences{
			NotifyOnAnnouncements: true,
			NotifyOnRequests:      true,
		},
	}

	profiles := mc.Database("grapple").Collection("profiles")
	filter := bson.M{
		"email": username,
	}

	// remove the gym association first
	update := bson.M{
		"$pull": bson.M{
			"gyms": bson.M{"gym._id": gym.ID},
		},
	}
	var result dao.Profile
	if err := mongoext.UpdateOne(ctx, profiles, update, filter, &result, nil); err != nil {
		return fmt.Errorf("failed to upsert profile with filter %v: %v", filter, err)
	}

	// re-add it to ensure no duplication
	update = bson.M{
		"$push": bson.M{
			"gyms": gymAssociation,
		},
	}

	log.Info().Msgf("Upserting gym association %v to user %q", gymAssociation, username)

	// Update student profile with the new gym association
	if err := mongoext.UpdateOne(ctx, profiles, update, filter, &result, nil); err != nil {
		return fmt.Errorf("failed to upsert profile with filter %v: %v", filter, err)
	}

	return nil
}

package profiles

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/cognito"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/Grapple-2024/backend/pkg/utils"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentityprovider"
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
	gymsCollection         *mongo.Collection
	CognitoClient          *cognito.Client
	awsRegion              string
	userPoolID             string
}

// NewService creates a new instance of a Profile Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, publicAssetsBucketName, region, userPoolID string, cognitoClient *cognito.Client) (*Service, error) {
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
		gymsCollection:         c.Database().Collection("gyms"),
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
		return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("permission denied: %v", err))
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
	var profiles []dao.Profile
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, options.Find(), &profiles); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}

	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if profiles == nil {
		resp, err := json.Marshal([]dao.Profile{})
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to marshal current user profiles to json: %v", err))
		}
		return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
	}

	for i, profile := range profiles {
		for j, membership := range profile.Gyms {
			var gym dao.Gym
			if err := mongoext.FindByID(ctx, s.gymsCollection, membership.GymID.Hex(), &gym); err != nil {
				return lambda.ServerError(fmt.Errorf("failed to find gym for gym membership! DATA INCONSISTENCY ERROR: %v", err))
			}

			profiles[i].Gyms[j].Gym = &gym
		}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	var result []byte
	if req.QueryStringParameters["current_user"] == "true" {
		profile := profiles[0]

		// marshal response as single profile object for easier frontend consumption
		resp, err := json.Marshal(profile)
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to marshal current user profile to json! %v", err))
		}
		result = resp
	} else {
		// marshal response as array
		result, err = service.NewGetAllResponse("profiles", profiles, totalCount, len(profiles), pageInt, pageSizeInt)
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
			return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("permission denied: %v", err))
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
			return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("failed to authorize user: %v", err))
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
	profileID := req.PathParameters["id"]

	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("failed to authenticate: %v", err))
	}

	var profile dao.Profile
	if err := mongoext.FindByID(ctx, s.Collection, profileID, &profile); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("profile with ID %s not found: %v", profileID, err))
	}

	// Safeguard authorization check
	if token.Sub != profile.CognitoID {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("authorization failure: you must be logged in as the same user you wish to delete %v", err),
		)
	}

	if err := s.deleteAllDataForProfile(ctx, profileID); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to delete profile with ID %q: %v", profileID, err))
	}
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) deleteAllDataForProfile(ctx context.Context, profileID string) error {
	var profile dao.Profile
	if err := mongoext.FindByID(ctx, s.Collection, profileID, &profile); err != nil {
		return fmt.Errorf("failed to find profile with id %s: %v", profileID, err)
	}

	// Find all Gyms created by the user (if any)
	var gyms []dao.Gym
	f := bson.M{
		"creator": profile.CognitoID,
	}
	if err := mongoext.Paginate(ctx, s.gymsCollection, f, 1, 100, false, options.Find(), &gyms); err != nil {
		log.Error().Msgf("failed to find gyms created by cognito ID %s: %v", profileID, err)
		// return fmt.Errorf("failed to find gyms created by cognito ID %s: %v", profileID, err)
	}

	// Delete Gyms and GymRequest objects created by the user with this cognito ID
	if err := s.deleteRecordsByCognitoID(ctx, profile.CognitoID); err != nil {
		return err
	}

	for _, gym := range gyms {
		// Delete all Cognito Groups for each Gym the user created
		if err := s.deleteCognitoGroupsForGym(ctx, gym.ID.Hex()); err != nil {
			return err
		}

		// Delete announcements, techniques, and series for each gym the user created
		if err := s.deleteRecordsByGymID(ctx, gym.ID); err != nil {
			return err
		}
	}

	// Delete the user from Cognito
	if err := s.deleteCognitoUser(ctx, profile.CognitoID); err != nil {
		return err
	}

	// Finally, delete

	return nil
}
func (s *Service) deleteCognitoGroupsForGym(ctx context.Context, gymID string) error {
	result, err := s.CognitoClient.ListGroups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list cognito groups: %v", err)
	}

	gymGroupPrefix := fmt.Sprintf("gym::%s", gymID)
	for _, group := range result.Groups {
		if !strings.HasPrefix(*group.GroupName, gymGroupPrefix) {
			continue
		}

		res, err := s.CognitoClient.DeleteGroup(ctx, &cognitoidentityprovider.DeleteGroupInput{
			GroupName:  group.GroupName,
			UserPoolId: &s.userPoolID,
		})
		if err != nil {
			return err
		}
		log.Info().Msgf("Deleted Cognito Group %s: %+v", *group.GroupName, res.ResultMetadata)

	}

	return nil
}

func (s *Service) deleteCognitoUser(ctx context.Context, username string) error {
	result, err := s.CognitoClient.AdminDeleteUser(ctx, &cognitoidentityprovider.AdminDeleteUserInput{
		Username:   &username,
		UserPoolId: &s.userPoolID,
	})
	if err != nil {
		return fmt.Errorf("failed to delete user %q in cognito: %v", username, err)
	}
	log.Info().Msgf("Delete Cognito User Result: %v", result.ResultMetadata)

	return nil
}

// deleteRecordsByGymID deletes any record in announcements, techniques, or gym-series table associated with the specified gym ID
func (s *Service) deleteRecordsByGymID(ctx context.Context, gymID bson.ObjectID) error {
	collections := []string{"announcements", "series", "techniques"}

	filter := bson.M{
		"gym_id": gymID,
	}
	for _, c := range collections {
		res, err := s.Database().Collection(c).DeleteMany(ctx, filter, nil)
		if err != nil {
			return fmt.Errorf("failed to delete records in collection %s associated with gym ID %s: %v", c, gymID, err)
		}

		log.Info().Msgf("Deleted %d records in collection %s associated with gym id %s", res.DeletedCount, c, gymID)
	}

	return nil
}

// deleteRecordsByCognitoID deletes any record in gymRequests or gyms table created by the specified cognito ID
func (s *Service) deleteRecordsByCognitoID(ctx context.Context, cognitoID string) error {
	queries := map[string]string{
		"gymRequests": "requestor_id",
		"gyms":        "creator",
		"profiles":    "cognito_id",
	}

	for collection, column := range queries {
		filter := bson.M{
			column: cognitoID,
		}
		c := s.Database().Collection(collection)
		res, err := c.DeleteMany(ctx, filter, nil)
		if err != nil {
			return fmt.Errorf("failed to delete records for cognito user %s: %v", cognitoID, err)
		}

		log.Info().Msgf("Deleted %d records in collection %s associated with cognito id %s", res.DeletedCount, collection, cognitoID)
	}

	return nil
}
func (s *Service) deleteGymByCognitoID(ctx context.Context, cognitoID string) error {
	filter := bson.M{
		"requestor_id": cognitoID,
	}
	gymRequestsColl := s.Database().Collection("gymRequests")
	res, err := gymRequestsColl.DeleteMany(ctx, filter, nil)
	if err != nil {
		return fmt.Errorf("failed to delete gym requests for cognito user %q: %v", cognitoID, err)
	}

	log.Info().Msgf("Delete gym requests count: %v", res.DeletedCount)

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
func UpsertGymAssociation(ctx context.Context, mc *mongoext.Client, gym *dao.Gym, roleName string, request *dao.GymRequest) error {
	groupName := fmt.Sprintf("gym::%s::%s", gym.ID.Hex(), utils.PluralGroupNameFromRole(roleName))
	gymAssociation := dao.GymAssociation{
		GymID: gym.ID,
		// Gym:            gym,
		Email:          request.RequestorEmail,
		MembershipType: request.MembershipType,
		Group:          groupName,
		EmailPreferences: &dao.EmailPreferences{
			NotifyOnAnnouncements: true,
			NotifyOnRequests:      true,
		},
	}

	profiles := mc.Database("grapple").Collection("profiles")
	filter := bson.M{
		"cognito_id": request.RequestorID,
	}

	// remove the gym association first
	update := bson.M{
		"$pull": bson.M{
			"gyms": bson.M{"gym_id": gym.ID},
		},
	}
	var result dao.Profile
	log.Info().Msgf("Pulling current gym association filter: %+v, %+v", update, filter)
	if err := mongoext.UpdateOne(ctx, profiles, update, filter, &result, nil); err != nil {
		return fmt.Errorf("failed to upsert profile with filter %v: %v", filter, err)
	}

	// re-add it to ensure no duplication
	update = bson.M{
		"$push": bson.M{
			"gyms": gymAssociation,
		},
	}

	log.Info().Msgf("Upserting gym association %v to user %q", gymAssociation, request.RequestorID)

	// Update student profile with the new gym association
	if err := mongoext.UpdateOne(ctx, profiles, update, filter, &result, nil); err != nil {
		return fmt.Errorf("failed to upsert profile with filter %v: %v", filter, err)
	}

	return nil
}

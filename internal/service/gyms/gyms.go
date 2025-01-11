package gyms

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
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
	*rbac.RBAC
	mongo.Session

	*s3.PresignClient
	*mongoext.Client
	*mongo.Collection
	publicAssetsBucketName string
	region                 string
}

// NewService creates a new instance of a Gym Service given a mongo client
func NewService(ctx context.Context, publicAssetsBucketName, region string, mc *mongoext.Client, rbac *rbac.RBAC) (*Service, error) {
	svc := &Service{
		Client:                 mc,
		Collection:             mc.Database("grapple").Collection("gyms"),
		RBAC:                   rbac,
		publicAssetsBucketName: publicAssetsBucketName,
		region:                 region,
	}
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
	svc.Session = session
	svc.PresignClient = s3.NewPresignClient(s3.NewFromConfig(cfg))

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /gyms/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	gymSlug := req.QueryStringParameters["slug"]
	creatorID := req.QueryStringParameters["creator_id"]
	name := req.QueryStringParameters["name"]

	page := req.QueryStringParameters["page"]
	if page == "" {
		page = "1"
	}
	pageSize := req.QueryStringParameters["page_size"]
	if pageSize == "" {
		pageSize = "10"
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
	if records == nil {
		records = []Gym{}
	}

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

// ProcessPost handless the creation of a Gym
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	isAuthorized, err := s.IsAuthorized(ctx, token.Username, rbac.ResourceGym, rbac.ActionCreate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionCreate, rbac.ResourceGym),
		)
	}

	var gym Gym
	if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	result, err := s.createGymTX(ctx, token, &gym)
	if err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to finish createGymTX: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for
// 1. PUT /gyms/{id} - insert/update a gym object
// 2. PUT /gyms/{id}/presign - generate presigned upload url for gym logo/banner/hero
// 3. PUT /gyms/{id}/assign-role - assign a user's role in the gym
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// update the record in mongo
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	id := req.PathParameters["id"]
	gymResourceID := fmt.Sprintf("%s:%s", rbac.ResourceGym, id)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, gymResourceID, rbac.ActionUpdate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionUpdate, gymResourceID),
		)
	}

	var gym Gym
	if err := mongoext.FindByID(ctx, s.Collection, id, &gym); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gym by ID: %v", err))
	}

	var result any
	gymSubPath := fmt.Sprintf("/gyms/%s", id)
	switch req.Path {
	case gymSubPath:
		var gym Gym
		if err := json.Unmarshal([]byte(req.Body), &gym); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}
		if err := mongoext.UpdateByID(ctx, s.Collection, id, gym, &result, nil); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}
	case fmt.Sprintf("%s/assign-role", gymSubPath):
		gymRolesResource := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, id, rbac.ResourceRoles)
		isAuthorized, err := s.IsAuthorized(ctx, token.Username, gymRolesResource, rbac.ActionUpdate)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
		} else if !isAuthorized {
			return lambda.ClientError(http.StatusForbidden,
				fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionCreate, gymRolesResource),
			)
		}

		var payload map[string]string
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}
		email := payload["email"]
		role := payload["role"]
		if email == "" {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("must specify email of user to assign roles to in the 'email' body field"))
		}
		if role == "" {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("must specify role to assign in the 'role' body field"))
		} else if role != rbac.Owners && role != rbac.Students && role != rbac.Coaches {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("must specify a valid role name in the 'role' body field: [owners, coaches, students]"))
		}}
		if err := s.RBAC.AssignUserToGymGroup(ctx, email, id, role); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to assign user %s to coaches group for gym %s", token.Username, id))
		}

		roleName := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, id, rbac.Owners)
		coachName := fmt.Sprintf("%s %s", gym.CoachFirstName, gym.CoachLastName)
		if err := profiles.AddGymAssociation(ctx, s.Client, gym.ID, coachName, roleName, token); err != nil {
			return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to add gym association to profile: %v", err))
		}

		result = map[string]string{
			"message": "success",
		}
	case fmt.Sprintf("%s/coaches", gymSubPath):
		// check authorization
		resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, id, rbac.ResourceCoaches)
		isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionCreate)
		if err != nil {
			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
		} else if !isAuthorized {
			return lambda.ClientError(http.StatusForbidden,
				fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionCreate, resourceID),
			)
		}

		var payload map[string]string
		if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}
		if payload["email"] == "" {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("must specify email of user to add as coach in 'email' field:"))
		}

		// update the record in mongo
		if err := s.RBAC.AssignUserToGymGroup(ctx, payload["email"], id, rbac.Coaches); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to assign user %s to coaches group for gym %s", token.Username, id))
		}

		// create the owners association on the gym owner's profile
		roleName := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, id, rbac.Coaches)
		coachName := fmt.Sprintf("%s %s", gym.CoachFirstName, gym.CoachLastName)
		if err := profiles.AddGymAssociation(ctx, s.Client, gym.ID, coachName, roleName, token); err != nil {
			return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to add gym association to profile: %v", err))
		}

		result = map[string]string{
			"message": "success",
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

	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	resourceID := fmt.Sprintf("%s:%s", rbac.ResourceGym, id)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionDelete)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionDelete, resourceID),
		)
	}

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

		// create the owners association on the gym owner's profile
		roleName := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, gym.ID, rbac.Owners)
		coachName := fmt.Sprintf("%s %s", gym.CoachFirstName, gym.CoachLastName)
		if err := profiles.AddGymAssociation(ctx, s.Client, gym.ID, coachName, roleName, token); err != nil {
			return nil, err
		}

		gymID := gym.ID.Hex()
		if err := s.RBAC.CreateGymRBAC(ctx, gymID); err != nil {
			return lambda.ServerError(err)
		}

		if err := s.RBAC.AssignUserToGymGroup(ctx, token.Username, gymID, rbac.Owners); err != nil {
			return lambda.ServerError(err)
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

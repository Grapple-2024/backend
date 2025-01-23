package gym_series

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

// Service is the object that handles the business logic of all Gym Series related operations.
// Service talks to the underlying Mongo Client (Data access layer or DAO) to CRUD Gym Series objects.
type Service struct {
	*rbac.RBAC

	*mongo.Session
	*mongoext.Client
	*mongo.Collection
	*s3.PresignClient

	videosBucketName       string // S3 Bucket name to store gym videos in
	publicAssetsBucketName string
}

// NewService creates a new instance of a GymSeries Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, videosBucketName, publicAssetsBucketName, region string, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("series")

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	svc := &Service{
		RBAC:                   rbac,
		Client:                 mc,
		Collection:             c,
		videosBucketName:       videosBucketName,
		publicAssetsBucketName: publicAssetsBucketName,
		PresignClient:          s3.NewPresignClient(s3.NewFromConfig(cfg)),
	}

	// Create Mongo Session (needed for transactions)
	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	return svc, nil
}

// ensureIndices ensures the proper indices are created for the 'gymseries' collection.
func (s *Service) ensureIndices(ctx context.Context) error {
	// Full-text index on title, description, coach_name, and disciplines
	_, err := s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"title":        "text",
			"videos.title": "text",
		},
		Options: options.Index().SetName("TextIndex"), // Optional: Set a custom name for the index
	})
	if err != nil {
		return err
	}

	return nil
}

// ProcessGetAll handles HTTP requests for GET /gym-requests/
// It takes in a context and a list of the requesting entitie's gym associations (IDs). It will query mongodb for series that match those IDs.
// TODO: remove dynamodb map after switching off fully
func (s *Service) buildGetAllFilter(req *events.APIGatewayProxyRequest, gymID string) (bson.M, error) {
	title := req.QueryStringParameters["title"]
	disciplines := req.MultiValueQueryStringParameters["discipline"]
	difficulties := req.MultiValueQueryStringParameters["difficulty"]
	// showByWeek := req.QueryStringParameters["show_by_week"]
	var and []bson.M
	var or []bson.M

	// Gym ID filter
	gymObjID, err := bson.ObjectIDFromHex(gymID)
	if err != nil {
		return nil, fmt.Errorf("invalid object ID specified for gym_id query param: %s", gymID)
	}

	and = append(and, bson.M{
		"gym_id": gymObjID,
	})

	// Show by week filter
	// if showByWeek != "" {
	// 	time, err := time.Parse(time.RFC3339, showByWeek)
	// 	if err != nil {
	// 		return nil, fmt.Errorf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err)
	// 	}
	// 	year, week := time.ISOWeek()

	// 	and = append(and,
	// 		bson.M{
	// 			"created_at_week": week,
	// 		},
	// 		bson.M{
	// 			"created_at_year": year,
	// 		},
	// 	)
	// }

	// Title search with full-text and regex
	if title != "" {
		or = append(or, bson.M{
			"title": bson.M{
				"$regex":   title,
				"$options": "i",
			},
		}, bson.M{
			"videos.title": bson.M{
				"$regex":   title,
				"$options": "i",
			},
		})
	}

	// Disciplines filter
	if len(disciplines) > 0 {
		and = append(and, bson.M{
			"videos.disciplines": bson.M{
				"$in": disciplines,
			},
		})
	}

	// Difficulties filter
	if len(difficulties) > 0 {
		and = append(and, bson.M{
			"videos.difficulty": bson.M{
				"$in": difficulties,
			},
		})
	}

	// Combine filters
	filter := bson.M{}
	if len(and) > 0 {
		filter["$and"] = and
	}
	if len(or) > 0 {
		filter["$or"] = or
	}

	log.Debug().Msgf("Filter: %v", filter)
	return filter, nil
}

func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to authenticate:: %v", err))
	}

	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("required query param ?gym_id not present"))
	}

	// check permission to read series on this gym
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, gymID, rbac.ResourceSeries) // gym:<gym_id>:series
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionRead)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionRead, resourceID),
		)
	}

	// Parse filter query params
	filter, err := s.buildGetAllFilter(&req, gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid filter param: %v", err))
	}

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
		return lambda.ClientError(http.StatusBadRequest, "invalid &page_size query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// Fetch records with pagination
	var records []GymSeries
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, true, options.Find(), &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []GymSeries{}
	}

	// generate presigned url for each video in the series
	if err := s.generatePresignedURLs(ctx, records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gym-series", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gym-series/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("failed to authenticate:: %v", err))
	}

	// Get the gymSeries by ID
	var gymSeries GymSeries
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymSeries); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymSeries by ID: %v", err))
	}

	// check permission to read series on this gym
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, gymSeries.GymID, rbac.ResourceSeries) // gym:<gym_id>:series
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionRead)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionRead, resourceID),
		)
	}

	// generate presigned urls for each video in the series
	if err := s.generatePresignedURLs(ctx, []GymSeries{gymSeries}); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}
	// Return record as JSON
	json, err := json.Marshal(gymSeries)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gym-series
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var payload GymSeries
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, payload.GymID.Hex(), rbac.ResourceSeries) // gym:<gym_id>:series
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionCreate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionCreate, resourceID),
		)
	}

	// Validate request body for required fields
	validate, err := service.NewValidator()
	if err != nil {
		return lambda.ServerError(err)
	}
	if err := validate.Struct(payload); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	payload.CreatedAt = time.Now().Local().UTC()
	payload.UpdatedAt = payload.CreatedAt
	payload.Videos = []Video{}

	// insert the series (payload), store the resulting record in 'result' variable
	var result GymSeries
	if err := mongoext.Insert(ctx, s.Collection, payload, &result); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for three endpoints:
// 1. PUT /gym-series/{id} -- Series Update
// 2. PUT /gym-videos/{id}/video/{id} -- Video insert/update
// 3. PUT /gym-series/{id}/presign -- generate presigned upload url for a new video
// 5. PUT /gym-series/{id}/presign-thumbnail - generate presigned upload url for series and videos
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	seriesID := req.PathParameters["id"]
	if seriesID == "" {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("must specify series ID in request url, ie /gym-series/{id}"))
	}
	var series GymSeries
	if err := mongoext.FindByID(ctx, s.Collection, seriesID, &series); err != nil {
		return lambda.ClientError(http.StatusNotFound, err.Error())
	}

	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, series.GymID.Hex(), rbac.ResourceSeries)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionUpdate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionUpdate, resourceID),
		)
	}

	var result any
	switch req.Path {
	case fmt.Sprintf("/gym-series/%s/presign-thumbnail", seriesID):
		file := req.QueryStringParameters["file"]
		if file == "" {
			return lambda.ClientError(
				http.StatusBadRequest,
				fmt.Sprintf("you must specify the file name and extension in ?file parameter, ie ?file=thumbnail.png"),
			)
		}

		now := time.Now().Unix()
		key := fmt.Sprintf("gyms/%s/series/%s/thumbnails/%d_%s", series.GymID.Hex(), series.ID.Hex(), now, file)
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.publicAssetsBucketName, "upload", key)
		if err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to generate presigned upload url: %v", err))
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
	case fmt.Sprintf("/gym-series/%s", seriesID):
		var gymSeries GymSeries
		if err := json.Unmarshal([]byte(req.Body), &gymSeries); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}

		if gymSeries.Disciplines != nil || gymSeries.Difficulties != nil {
			log.Warn().Msgf("difficulties and disciplines are calculated fields, must not specify in update request body.")
			gymSeries.Disciplines = nil
			gymSeries.Difficulties = nil
		}
		if err := s.calculateDisciplines(&gymSeries); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add disciplines to series: %v", err))
		}
		if err := s.calculateDifficulties(&gymSeries); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add difficulties to series: %v", err))
		}

		if err := mongoext.UpdateByID(ctx, s.Collection, seriesID, gymSeries, &result, nil); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

	case fmt.Sprintf("/gym-series/%s/presign", seriesID):
		file := req.QueryStringParameters["file"]
		if file == "" {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("you must specify the file name and extension in ?file parameter, ie ?file=video1.mp4"))
		}
		fileType := req.QueryStringParameters["type"]
		if file == "" {
			return lambda.ClientError(
				http.StatusBadRequest,
				fmt.Sprintf("you must specify the file name and extension in ?file parameter, ie ?file=video.mp4&type=video OR ?file=thumbnail.png&type=thumbnail"),
			)
		}
		if fileType != "video" && fileType != "thumbnail" {
			return lambda.ClientError(
				http.StatusBadRequest,
				fmt.Sprintf("invalid file type specified in ?file query parameter, possible options are: [video, thumbnail]"),
			)
		}

		key := fmt.Sprintf("gyms/%s/series/%s/%s/%d_%s", series.GymID.Hex(), series.ID.Hex(), fileType, time.Now().UnixNano(), file)
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.videosBucketName, "upload", key)
		if err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to generate presigned upload url: %v", err))
		}
		resp := struct {
			*v4.PresignedHTTPRequest
			S3ObjectKey string `json:"s3_object_key"`
		}{
			PresignedHTTPRequest: p,
			S3ObjectKey:          key,
		}

		result = resp
	case fmt.Sprintf("/gym-series/%s/videos", seriesID):
		// Create or Update a Video in a series
		// TOOD: separate this logic out into func getUpdateVideoFilter()
		var video Video
		if err := json.Unmarshal([]byte(req.Body), &video); err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}

		seriesObjID, err := bson.ObjectIDFromHex(seriesID)
		if err != nil {
			return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid series ID: %v", err))
		}

		var filter bson.M
		var update bson.M
		if video.ID == bson.NilObjectID {
			// Create a new video in the series
			video.ID = bson.NewObjectIDFromTimestamp(time.Now())
			video.UpdatedAt = time.Now().Local().UTC()
			video.CreatedAt = video.UpdatedAt
			validate := validator.New()

			if err := validate.Struct(video); err != nil {
				var errMsgs []string
				for _, err := range err.(validator.ValidationErrors) {
					errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
				}
				return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
			}

			series.Videos = append(series.Videos, video)
			if err := s.calculateDisciplines(&series); err != nil {
				return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add disciplines to series: %v", err))
			}
			if err := s.calculateDifficulties(&series); err != nil {
				return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add difficulties to series: %v", err))
			}

			// create a new video
			filter = bson.M{
				"_id": seriesObjID,
			}
			update = bson.M{
				"$push": bson.M{
					"videos": video,
				},
				"$set": bson.M{
					"disciplines":  series.Disciplines,
					"difficulties": series.Difficulties,
				},
			}
		} else {
			// re-calculate the disciplines/difficulties on the series
			for i := 0; i < len(series.Videos); i++ {
				if series.Videos[i].ID != video.ID {
					continue
				}
				// update the series in-place
				series.Videos[i].Disciplines = video.Disciplines
				series.Videos[i].Difficulty = video.Difficulty
			}
			if err := s.calculateDisciplines(&series); err != nil {
				return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add disciplines to series: %v", err))
			}
			if err := s.calculateDifficulties(&series); err != nil {
				return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not add difficulties to series: %v", err))
			}

			// Update an existing video in the series
			video.UpdatedAt = time.Now().Local().UTC()
			filter = bson.M{
				"_id":        seriesObjID,
				"videos._id": video.ID,
			}
			update = bson.M{
				"$set": bson.M{
					"videos.$.title":       video.Title,
					"videos.$.description": video.Description,
					"videos.$.sort_order":  video.SortOrder,
					"videos.$.difficulty":  video.Difficulty,
					"videos.$.disciplines": video.Disciplines,
					"disciplines":          series.Disciplines,
					"difficulties":         series.Difficulties,
				},
			}
		}

		if err := mongoext.UpdateOne(ctx, s.Collection, update, filter, &result, nil); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update series with video: %v", err))
		}
	default:
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("invalid request url: %v", req.Path))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for
// 1. DELETE /gym-series/{id}
// 2. DELETE /gym-series/{id}/videos/{id}.
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	videoID := req.PathParameters["video_id"]

	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var series GymSeries
	if err := mongoext.FindByID(ctx, s.Collection, id, &series); err != nil {
		return lambda.ClientError(http.StatusNotFound, err.Error())
	}

	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, series.GymID.Hex(), rbac.ResourceSeries)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionDelete)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionDelete, resourceID),
		)
	}

	objID, err := bson.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	var result any
	var filter bson.M
	switch req.Path {
	case fmt.Sprintf("/gym-series/%s", id):
		if err = mongoext.DeleteOne(ctx, s.Collection, id); err != nil {
			return lambda.ServerError(err)
		}

	// Delete a Gym Series Video
	case fmt.Sprintf("/gym-series/%s/videos/%s", id, videoID):
		filter = bson.M{
			"_id": objID,
		}

		videoObjID, err := bson.ObjectIDFromHex(videoID)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid video ID %q: %v", videoID, err))
		}
		for i := 0; i < len(series.Videos); i++ {
			if series.Videos[i].ID == videoObjID {
				// remove the video from the series.Videos slice
				series.Videos = append(series.Videos[:i], series.Videos[i+1:]...)
				break
			}
		}

		// recalculate top-level disciplines and difficulties fields on the Series object.
		if err := s.calculateDisciplines(&series); err != nil {
			return lambda.ClientError(http.StatusBadRequest, err.Error())
		}
		if err := s.calculateDifficulties(&series); err != nil {
			return lambda.ClientError(http.StatusBadRequest, err.Error())
		}

		update := bson.M{
			"$pull": bson.M{
				"videos": bson.M{
					"_id": videoObjID,
				},
			},
			"$set": bson.M{
				"disciplines":  series.Disciplines,
				"difficulties": series.Difficulties,
			},
		}
		log.Debug().Msgf("Deleting video from series: update: %v\n filter: %v", update, filter)

		if err := mongoext.UpdateOne(ctx, s.Collection, update, filter, &result, nil); err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to delete video %q from series %q %v", videoID, id, err))
		}
		log.Info().Msgf("Update result: %+v", result)

	default:
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid request path %q: %v", req.Path, err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusAccepted, string(resp), nil), nil
}

func (s *Service) updateSeriesTransaction(ctx context.Context, payload *GymSeries, id string) (*GymSeries, error) {
	transactionOptions := options.Transaction().SetReadConcern(&readconcern.ReadConcern{Level: "local"}).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		var result GymSeries
		if err := mongoext.UpdateByID(ctx, s.Collection, id, payload, &result, nil); err != nil {
			return lambda.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}
		log.Info().Msgf("Update GymSeries result: %v", result)

		return result, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("updateSeries transaction completed successfully!")
	}

	return result.(*GymSeries), nil
}

// generatePresignedURLs generates presigned URL for each video in the records slice.
// It modifies the records slice by reference and returns an error
func (s *Service) generatePresignedURLs(ctx context.Context, records []GymSeries) error {
	for i, series := range records {
		for j, video := range series.Videos {
			p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.videosBucketName, "download", video.S3ObjectKey)
			if err != nil {
				return fmt.Errorf("failed to generate presigned url: %v", err)
			}
			records[i].Videos[j].PresignedURL = p.URL
		}
	}
	return nil
}

func (s *Service) calculateDisciplines(series *GymSeries) error {
	series.Disciplines = &[]string{}

	disciplineSet := make(map[string]bool)
	for _, v := range series.Videos {
		for _, d := range v.Disciplines {
			disciplineSet[d] = true
		}
	}

	// Convert the set to a slice
	for discipline := range disciplineSet {
		*series.Disciplines = append(*series.Disciplines, discipline)
	}

	return nil
}

func (s *Service) calculateDifficulties(series *GymSeries) error {
	series.Difficulties = &[]string{}

	difficultySet := make(map[string]bool)
	for _, v := range series.Videos {
		difficultySet[v.Difficulty] = true
	}

	// Convert the set to a slice
	for d := range difficultySet {
		*series.Difficulties = append(*series.Difficulties, d)
	}

	return nil
}

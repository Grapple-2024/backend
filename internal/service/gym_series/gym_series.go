package gym_series

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
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

// Service is the object that handles the business logic of all Gym Series related operations.
// Service talks to the underlying Mongo Client (Data access layer or DAO) to CRUD Gym Series objects.
type Service struct {
	mongo.Session

	*mongoext.Client
	*mongo.Collection
	*s3.PresignClient

	videosBucketName string // S3 Bucket name to store gym videos in
}

// NewService creates a new instance of a GymSeries Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, videosBucketName, region string) (*Service, error) {
	c := mc.Database("grapple").Collection("series")

	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	svc := &Service{
		Client:           mc,
		Collection:       c,
		videosBucketName: videosBucketName,
		PresignClient:    s3.NewPresignClient(s3.NewFromConfig(cfg)),
	}

	// Create Mongo Session (needed for transactions)
	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /gym-requests/
// TODO: remove dynamodb map after switching off fully
func (s *Service) buildGetAllFilter(ctx context.Context, req *events.APIGatewayProxyRequest) (bson.M, error) {
	title := req.QueryStringParameters["title"]
	disciplines := req.MultiValueQueryStringParameters["discipline"]
	difficulties := req.MultiValueQueryStringParameters["difficulty"]
	showByWeek := req.QueryStringParameters["show_by_week"]
	gymID := req.QueryStringParameters["gym_id"]

	filter := bson.M{}
	var and []bson.M
	if gymID != "" {
		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			return nil, fmt.Errorf("invalid object ID specified for gym_id query param: %s", gymID)
		}
		and = append(and, bson.M{
			"gym_id": gymObjID,
		})
	}

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return nil, fmt.Errorf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err)
		}
		year, week := time.ISOWeek()

		and = append(and,
			bson.M{
				"created_at_week": week,
			},
			bson.M{
				"created_at_year": year,
			},
		)

	}

	if title != "" {
		and = append(and, bson.M{
			"title": bson.M{
				"$regex": fmt.Sprintf("^%s", title),
			},
		})
	}

	if len(disciplines) > 0 {
		and = append(and, bson.M{
			"disciplines": bson.M{
				"$in": disciplines,
			},
		})
	}

	if len(difficulties) > 0 {
		and = append(and, bson.M{
			"difficulties": bson.M{
				"$in": difficulties,
			},
		})
	}

	if len(and) > 0 {
		filter = bson.M{
			"$and": and,
		}
	}

	log.Debug().Msgf("Filter: %v", filter)
	return filter, nil
}

func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, _ map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// Parse filter query params
	filter, err := s.buildGetAllFilter(ctx, &req)
	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid filter param: %v", err))
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
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &pageSize query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// create the filter based on the query parameters in the request
	// filter := bson.M{}
	// if gymID != "" {
	// 	gymObjID, err := primitive.ObjectIDFromHex(gymID)
	// 	if err != nil {
	// 		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
	// 	}
	// 	filter["gym_id"] = gymObjID
	// }

	// if showByWeek != "" {
	// 	time, err := time.Parse(time.RFC3339, showByWeek)
	// 	if err != nil {
	// 		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
	// 	}
	// 	year, week := time.ISOWeek()
	// 	filter["created_at_year"] = year
	// 	filter["created_at_week"] = week
	// }

	// Fetch records with pagination
	var records []GymSeries
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, true, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []GymSeries{}
	}

	// generate presigned url for each video in the series
	if err := s.generatePresignedURLs(ctx, records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, err.Error())
	}

	// add disciplines and difficulties to the response
	newSeries, err := s.addDisciplinesToTopLevel(records)

	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, err.Error())
	}

	newSeries, err = s.addDifficultiesToTopLevel(newSeries)

	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, err.Error())
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gym-series", newSeries, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gym-series/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gymSeries by ID
	var gymSeries GymSeries
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymSeries); err != nil {
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymSeries by ID: %v", err))
	}

	// generate presigned urls for each video in the series
	if err := s.generatePresignedURLs(ctx, []GymSeries{gymSeries}); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, err.Error())
	}

	// Return record as JSON
	json, err := json.Marshal(gymSeries)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gym-series
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymSeries GymSeries
	if err := json.Unmarshal([]byte(req.Body), &gymSeries); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(gymSeries); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	gymSeries.CreatedAt = time.Now().Local().UTC()
	gymSeries.UpdatedAt = gymSeries.CreatedAt

	// insert the GymSeries, store the resulting record in 'result' variable
	var result GymSeries
	if err := mongoext.Insert(ctx, s.Collection, &gymSeries, &result); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for three endpoints:
// 1. PUT /gym-series/{id} -- Series Update)
// 2. /gym-videos/{id}/video/{id} -- Video update
// 3. /gym-series/{id}/presign -- generate presigned upload url for a new video
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	if id == "" {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("must specify {id} in request url"))
	}

	var result any
	switch req.Path {
	case fmt.Sprintf("/gym-series/%s", id):
		var gymSeries GymSeries
		if err := json.Unmarshal([]byte(req.Body), &gymSeries); err != nil {
			return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}
		if gymSeries.Videos != nil {
			return lambda_v2.ClientError(http.StatusUnprocessableEntity, "must use /gym-series/{id}/videos/{id} to update a video in a series %v")
		}

		// update the record in mongo
		if err := mongoext.UpdateByID(ctx, s.Collection, id, gymSeries, &result, nil); err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}
		log.Debug().Msgf("Updated mongo document by ID: %v", result)

	case fmt.Sprintf("/gym-series/%s/presign", id):
		// uploading a video to the series, generate presigned upload URL
		file := req.QueryStringParameters["file"]
		if file == "" {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("you must specify the file name and extension in ?file parameter, ie ?file=video1.mp4"))
		}
		key := fmt.Sprintf("%s/%s", id, file)

		log.Debug().Msgf("Generating presigned upload url for a new series video %q in series %q", file, id)
		p, err := service.GeneratePresignedURL(ctx, s.PresignClient, s.videosBucketName, "upload", key)
		if err != nil {
			return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to generate presigned upload url: %v", err))
		}
		resp := struct {
			*v4.PresignedHTTPRequest
			S3ObjectKey string `json:"s3_object_key"`
		}{
			PresignedHTTPRequest: p,
			S3ObjectKey:          key,
		}

		result = resp
	case fmt.Sprintf("/gym-series/%s/videos", id):
		// Create or Update a Video in a series
		// TOOD: separate this logic out into func getUpdateVideoFilter()
		var video Video
		if err := json.Unmarshal([]byte(req.Body), &video); err != nil {
			return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
		}

		id, err := primitive.ObjectIDFromHex(id)
		if err != nil {
			return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid series ID: %v", err))
		}

		var filter bson.M
		var update bson.M
		log.Info().Msgf("Video ID: %v", video.ID)
		if video.ID == primitive.NilObjectID {
			// Create a new video in the series
			video.ID = primitive.NewObjectIDFromTimestamp(time.Now())
			video.UpdatedAt = time.Now().Local().UTC()
			video.CreatedAt = video.UpdatedAt
			validate := validator.New()

			// validate the struct
			if err := validate.Struct(video); err != nil {
				var errMsgs []string
				for _, err := range err.(validator.ValidationErrors) {
					errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
				}
				return lambda_v2.ClientError(http.StatusUnprocessableEntity, errMsgs...)
			}

			// create a new video
			filter = bson.M{
				"_id": id,
			}
			update = bson.M{
				"$push": bson.M{
					"videos": video,
				},
			}
		} else {
			// Update an existing video in the series
			video.UpdatedAt = time.Now().Local().UTC()
			filter = bson.M{
				"_id":        id,
				"videos._id": video.ID,
			}
			update = bson.M{
				"$set": bson.M{
					"videos.$": video,
				},
			}
		}

		log.Info().Msgf("Filter: %v", filter)
		log.Info().Msgf("Update: %v", update)

		if err := mongoext.Update(ctx, s.Collection, update, filter, &result, nil); err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update series with video: %v", err))
		}
	default:
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("invalid request url: %v", req.Path))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gym-series/{id} and /gym-series/{id}/videos/{id}.
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	videoID := req.PathParameters["video_id"]

	var result any
	var opts *options.DeleteOptions
	var filter bson.M
	switch req.Path {
	case fmt.Sprintf("/gym-series/%s", id):
		filter = bson.M{"_id": objID}
		opts = options.Delete().SetHint(bson.M{"_id": 1}) // use series ID index to delete the object

		result, err = s.Collection.DeleteOne(ctx, filter, opts)
		if err != nil {
			return lambda_v2.ServerError(err)
		}

	case fmt.Sprintf("/gym-series/%s/videos/%s", id, videoID):
		filter = bson.M{
			"_id": objID,
		}

		videoObjID, err := primitive.ObjectIDFromHex(videoID)
		if err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid video ID %q: %v", videoID, err))
		}
		update := bson.M{
			"$pull": bson.M{
				"videos": bson.M{
					"_id": videoObjID,
				},
			},
		}
		log.Debug().Msgf("Deleting video from series: update: %v\n filter: %v", update, filter)

		if err := mongoext.Update(ctx, s.Collection, update, filter, &result, nil); err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to delete video %q from series %q %v", videoID, id, err))
		}
		log.Info().Msgf("Update result: %+v", result)

	default:
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid request path %q: %v", req.Path, err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

func (s *Service) updateSeriesTransaction(ctx context.Context, payload *GymSeries, id string) (*GymSeries, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		var result GymSeries
		if err := mongoext.UpdateByID(ctx, s.Collection, id, payload, &result, nil); err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}
		log.Info().Msgf("Update GymSeries result: %v", result)

		return result, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("Failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("createProfile transaction completed successfully!")
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

func (s *Service) addDisciplinesToTopLevel(records []GymSeries) ([]GymSeries, error) {
	resp := []GymSeries{}

	for _, series := range records {
		disciplineSet := make(map[string]struct{}) // A set to avoid duplicates

		// Loop through each video in the series
		for _, video := range series.Videos {
			// Loop through each discipline in the video
			for _, discipline := range video.Disciplines {
				disciplineSet[discipline] = struct{}{} // Add discipline to the set
			}
		}

		// Convert the set to a slice
		for discipline := range disciplineSet {
			series.Disciplines = append(series.Disciplines, discipline)
		}

		resp = append(resp, series)
	}

	return resp, nil
}

func (s *Service) addDifficultiesToTopLevel(records []GymSeries) ([]GymSeries, error) {
	resp := []GymSeries{}

	for _, series := range records {
		difficultySet := make(map[string]struct{}) // A set to avoid duplicates

		// Loop through each video in the series
		for _, video := range series.Videos {
			// Add the difficulty of the video to the set
			difficultySet[video.Difficulty] = struct{}{}
		}

		// Convert the set to a slice
		for difficulty := range difficultySet {
			series.Difficulties = append(series.Difficulties, difficulty)
		}

		resp = append(resp, series)
	}

	return resp, nil
}

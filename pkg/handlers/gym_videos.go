package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/pkg/dynamodb"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type GymVideoHandler struct {
	*dynamodbsdk.Client
	*AuthService
	*s3.PresignClient
	videosTable string
}

type GymVideo struct {
	PK       string `json:"pk,omitempty" dynamodbav:"pk,omitempty"`     // primary key
	SeriesID string `json:"series_id" dynamodbav:"series_id,omitempty"` // foreign key

	// attributes
	GymID       string   `json:"gym_id,omitempty" dynamodbav:"gym_id,omitempty"`
	Title       string   `json:"title,omitempty" dynamodbav:"title,omitempty"`
	Description string   `json:"description,omitempty" dynamodbav:"description,omitempty"`
	Difficulty  string   `json:"difficulty,omitempty" dynamodbav:"difficulty,omitempty"`
	Disciplines []string `json:"disciplines,omitempty" dynamodbav:"disciplines,stringsets,omitempty"`
	S3Object    string   `json:"s3_object,omitempty" dynamodbav:"s3_object,omitempty"`
	SortOrder   int32    `json:"sort_order" dynamodbav:"sort_order"`

	// Computed fields on any GET:
	PresignedURL string `json:"presigned_url,omitempty" dynamodbav:"presigned_url,omitempty"`

	CreatedAt time.Time `json:"created_at,omitempty" dynamodbav:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty" dynamodbav:"updated_at,omitempty"`

	Dummy string `json:"-" dynamodbav:"dummy,omitempty"`
}

func NewGymVideoHandler(ctx context.Context, dynamoEndpoint string) (*GymVideoHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	// create AWS cfg
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	// create AWS s3 and pre-sign clients
	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)

	return &GymVideoHandler{
		Client:        db,
		AuthService:   authSVC,
		PresignClient: psc,
		videosTable:   os.Getenv("GYM_VIDEOS_TABLE_NAME"),
	}, nil
}

func (h *GymVideoHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *GymVideoHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	result, err := h.GetByID(ctx, h.videosTable, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	if len(result.Items) > 0 {
		gymID := result.Items[0]["gym_id"].(*types.AttributeValueMemberS).Value
		objectKey := result.Items[0]["s3_object"].(*types.AttributeValueMemberS).Value
		key := fmt.Sprintf("%s/%s", gymID, objectKey)
		url, err := h.getPresignedURL(key)
		if err != nil {
			return lambda.ServerError(fmt.Errorf("failed to get presigned url: %v", err))
		}

		result.Items[0]["presigned_url"] = url
	}

	var videos []GymVideo
	err = attributevalue.UnmarshalListOfMaps(result.Items, &videos)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(videos[0])
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil

}

func (h *GymVideoHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Unmarshal request body into GymVideo struct
	var video GymVideo
	if err := json.Unmarshal([]byte(req.Body), &video); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	// Validate the request
	if err := validate.Struct(&video); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("request body failed validation: %v", err))
	}

	// Insert the GymVideo object into DynamoDB
	video.CreatedAt = time.Now().UTC()
	video.UpdatedAt = video.CreatedAt
	video.Dummy = "dumb"
	video.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymVideo#%s/%s/%d", video.GymID, video.Title, video.CreatedAt.Unix())),
	)
	_, err := h.Insert(ctx, h.videosTable, &video, "pk")
	if err != nil {
		return lambda.ServerError(err)
	}

	// synchronize the difficulties and disciplines of the associated GymVideoSeries entity.
	if err := h.syncSeries(ctx, video.SeriesID); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to sync series: %w", err))
	}

	json, err := json.Marshal(&video)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymVideoHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in url path")
	}

	// Fetch the Video
	result, err := h.GetByID(ctx, h.videosTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to fetch video by ID: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "video not found, cannot delete it")
	}

	var videos []GymVideo
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &videos); err != nil {
		return lambda.ServerError(fmt.Errorf("cannot unmarshal dynamodb response to []GymVideo: %v", err))
	}
	videoToDelete := videos[0]

	// Delete the video from dynamodb
	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Delete(ctx, h.videosTable, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	// sync associated series
	if err := h.syncSeries(ctx, videoToDelete.SeriesID); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to sync the video's associated series: %w", err))
	}

	json, err := json.Marshal(resp.ResultMetadata)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}
func (h *GymVideoHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	// Unmarshal JSON http request body into GymVideo struct
	var video GymVideo
	if err := json.Unmarshal([]byte(req.Body), &video); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	r, err := h.updateGymVideo(ctx, id, &video)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update video in dynamo: %v", err))
	}

	// Sync the updated videos' series
	if err := h.syncSeries(ctx, video.SeriesID); err != nil {
		log.Warn().Msgf("failed to sync the video's associated series: %v", err)
		// return lambda.ServerError(fmt.Errorf("failed to sync the video's associated series: %w", err))
	}

	// Unmarshal Dynamodb response into GymVideo
	var returnVideo GymVideo
	if err := attributevalue.UnmarshalMap(r.Attributes, &returnVideo); err != nil {
		return lambda.ServerError(err)
	}

	// Marshal GymVideo into JSON and serve the response
	json, err := json.Marshal(returnVideo)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymVideoHandler) getPresignedURL(key string) (*types.AttributeValueMemberS, error) {
	// Get pre-signed URL
	bucketName := os.Getenv("GYM_VIDEOS_BUCKET_NAME")
	params := &s3.GetObjectInput{
		Bucket: &bucketName,
		Key:    aws.String(key),
	}

	r, err := h.PresignClient.PresignGetObject(context.TODO(), params, func(opts *s3.PresignOptions) {
		opts.Expires = time.Minute * 30
	})
	if err != nil {
		return nil, fmt.Errorf("couldn't get a presigned download url for object %v: %w", key, err)
	}

	return &types.AttributeValueMemberS{
		Value: r.URL,
	}, nil
}

func (h *GymVideoHandler) syncSeries(ctx context.Context, seriesID string) error {
	// Get the updated/deleted/created video's associated Series object from DynamodB
	r, err := h.GetByID(ctx, videoSeriesTableName, seriesID)
	if err != nil {
		return fmt.Errorf("failed to fetch series %q: %w", seriesID, err)
	}
	var seriess []GymVideoSeries
	if err := attributevalue.UnmarshalListOfMaps(r.Items, &seriess); err != nil {
		return err
	}
	if len(seriess) == 0 {
		return fmt.Errorf("could not find a series with PK %s", seriesID)
	}
	series := seriess[0]

	// Get all the associated videos for this series
	log.Info().Msgf("Series Sync:: current disciplines: %v\n", series.Disciplines)
	log.Info().Msgf("Series Sync:: current difficulties: %v\n\n", series.Difficulties)

	// fetch all associated videos for the series, will contain the latest updates
	if err := fetchVideosForSeries(ctx, h, h.PresignClient, &series, false); err != nil {
		return err
	}

	disciplines := []string{}
	difficulties := []string{}
	for _, v := range series.Videos {
		disciplines = append(disciplines, v.Disciplines...)
		difficulties = append(difficulties, v.Difficulty)

		if !slices.Contains(series.Difficulties, v.Difficulty) {
			series.Difficulties = append(series.Difficulties, v.Difficulty)
		}
		for _, d := range v.Disciplines {
			if !slices.Contains(series.Disciplines, d) {
				series.Disciplines = append(series.Disciplines, d)
			}
		}
	}

	// Clean up any disciplines / difficulties in the series that aren't reflected in any video (backwards compatible / safe guard feature)
	for i, discipline := range series.Disciplines {
		log.Info().Msgf("checking if series needs discipline %v", discipline)

		if !slices.Contains(disciplines, discipline) {
			log.Info().Msgf("found extraneous discipline in series, removing it: %v", discipline)
			series.Disciplines = append(series.Disciplines[:i], series.Disciplines[i+1:]...)
		}
	}
	for i, difficulty := range series.Difficulties {
		log.Info().Msgf("checking if series needs difficulty %v", difficulty)
		if !slices.Contains(difficulties, difficulty) {
			log.Info().Msgf("found extraneous difficulty in series, removing it: %v", difficulty)
			series.Difficulties = append(series.Difficulties[:i], series.Difficulties[i+1:]...)
			log.Info().Msgf("disciplines after index removal: %v", series.Difficulties)
		}
	}

	log.Info().Msgf("Series Sync:: new disciplines: %v\n", series.Disciplines)
	log.Info().Msgf("Series Sync:: new difficulties: %v\n\n", series.Difficulties)

	if _, err := h.updateGymVideoSeries(ctx, series.PK, &series); err != nil {
		return err
	}

	return nil
}

func (h GymVideoHandler) updateGymVideoSeries(ctx context.Context, id string, series *GymVideoSeries) (*dynamodb.UpdateItemOutput, error) {
	// Marshal request payload into map[string]types.AttributeValue
	av, err := attributevalue.MarshalMap(series)
	if err != nil {
		return nil, err
	}

	// Build update expression for dynamodb
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "gym_id" || k == "created_at" || k == "updated_at" {
			continue // continue on immutable fields
		}
		update = update.Set(expression.Name(k), expression.Value(v))
	}
	update = update.Set(expression.Name("updated_at"), expression.Value(time.Now().UTC()))
	builder := expression.NewBuilder().WithCondition(expression.Equal(
		expression.Name("pk"),
		expression.Value(id),
	),
	).WithUpdate(update)

	expr, err := builder.Build()
	if err != nil {
		return nil, err
	}

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	return h.Update(ctx, videoSeriesTableName, key, &expr, false)
}

func (h GymVideoHandler) updateGymVideo(ctx context.Context, id string, video *GymVideo) (*dynamodb.UpdateItemOutput, error) {
	// Marshal request payload into map[string]types.AttributeValue
	av, err := attributevalue.MarshalMap(video)
	if err != nil {
		return nil, err
	}

	// Build update expression for dynamodb
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "gym_id" || k == "created_at" || k == "updated_at" {
			continue // continue on immutable fields
		}
		update = update.Set(expression.Name(k), expression.Value(v))
	}
	update = update.Set(expression.Name("updated_at"), expression.Value(time.Now().UTC()))
	builder := expression.NewBuilder().WithCondition(expression.Equal(
		expression.Name("pk"),
		expression.Value(id),
	),
	).WithUpdate(update)

	expr, err := builder.Build()
	if err != nil {
		return nil, err
	}

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	return h.Update(ctx, h.videosTable, key, &expr, false)
}

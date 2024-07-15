package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
)

const (
	coachGroupARN = "arn:aws:iam::381491926210:role/us-west-1_HT5oR6AwO-coachGroupRole"

	operationDownload = "download"
	operationUpload   = "upload"
)

type S3Handler struct {
	*AuthService
	*s3.PresignClient
	S3Client         *s3.Client
	videosBucketName string
}

func NewS3Handler(ctx context.Context, dynamoEndpoint, region string) (*S3Handler, error) {
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &S3Handler{
		S3Client:         c,
		PresignClient:    psc,
		AuthService:      authSVC,
		videosBucketName: os.Getenv("GYM_VIDEOS_BUCKET_NAME"),
	}, nil
}

func (h *S3Handler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	gym := req.QueryStringParameters["gym"]
	ttl := req.QueryStringParameters["ttl"]
	keys := req.MultiValueQueryStringParameters["key"]
	log.Info().Msgf("Multi query string params: %v", req.MultiValueQueryStringParameters["key"])
	operation := req.QueryStringParameters["operation"]

	// check for empty required parameter
	if len(keys) == 0 || gym == "" || operation == "" {
		return lambda.ClientError(http.StatusNotFound, "must specify ?key=<file-name>&gym=<gym_pk>&operation=<download|upload> query string parameters")
	}

	// set default ttl
	if ttl == "" {
		ttl = "5m"
	}
	ttlDur, err := time.ParseDuration(ttl)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("ttl must be an integer: %v", err))
	}

	var resp any
	switch operation {
	case operationUpload:
		if len(keys) > 1 {
			return lambda.ClientError(http.StatusNotFound, "you can only specify one key query parameter during an upload operation")
		}

		// check to make sure the token is a coach of the gym
		if err := h.IsCoach(ctx, req.Headers, gym); err != nil {
			log.Error().Err(err).Msgf("security incident: user tried to upload file to a gym they are not a coach of!")

			return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("could not verify token is a coach: %v", err))
		}

		// create the presigned upload URL
		objectKey := fmt.Sprintf("%s/%s", gym, keys[0])
		r, err := h.createPresignedUploadURL(h.videosBucketName, objectKey, ttlDur)
		if err != nil {
			return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error creating presigned upload url: %v", err))
		}

		resp = map[string]any{
			"url":       r.URL,
			"s3_object": keys[0],
		}
	case operationDownload:
		// check to make sure the token is either a coach or student
		// check to make sure the token is a student / coach of the gym
		isNotCoach := h.IsCoach(ctx, req.Headers, gym)
		isNotStudent := h.IsStudent(ctx, req.Headers, gym)
		if isNotCoach != nil && isNotStudent != nil {
			log.Error().Err(err).Msgf("security incident! user tried to download a file from a gym they are neither a student or coach of!")
			return lambda.ClientError(http.StatusForbidden, "user is neither a coach or student of this gym")
		}

		presignedURLs := []map[string]any{}
		for _, key := range keys {
			objectKey := fmt.Sprintf("%s/%s", gym, key)
			r, err := h.createPresignedDownloadURL(h.videosBucketName, objectKey, ttlDur)
			if err != nil {
				return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error creating presigned download url: %v", err))
			}
			log.Info().Msgf("Processing key: %v", key)
			log.Info().Msgf("Presigned URL: %v", r)

			presignedURLs = append(presignedURLs, map[string]any{
				"url":       r.URL,
				"s3_object": key,
			})
		}
		resp = presignedURLs

	default:
		return lambda.ClientError(http.StatusNotFound, "invalid opeation value. valid values for ?operation are either 'download' or 'upload'")
	}

	bytes, err := json.Marshal(resp)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error marshaling presigned url response to json: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(bytes), nil), nil
}

func (h *S3Handler) createPresignedUploadURL(bucketName string, objectKey string, ttl time.Duration) (*v4.PresignedHTTPRequest, error) {
	params := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	request, err := h.PresignClient.PresignPutObject(context.TODO(), params, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		log.Info().Msgf("Couldn't get a presigned request to put %v:%v. Here's why: %v", bucketName, objectKey, err)
		return nil, err
	}

	return request, nil
}

func (h *S3Handler) createPresignedDownloadURL(bucketName string, objectKey string, ttl time.Duration) (*v4.PresignedHTTPRequest, error) {
	params := &s3.GetObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
	}

	request, err := h.PresignClient.PresignGetObject(context.TODO(), params, func(opts *s3.PresignOptions) {
		opts.Expires = ttl
	})
	if err != nil {
		log.Info().Msgf("Couldn't get a presigned download url for object %v:%v. Here's why: %v", bucketName, objectKey, err)
		return nil, err
	}

	return request, nil
}

// Needed to satisfy interface, but not implemented

func (h *S3Handler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}
func (h *S3Handler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

func (h *S3Handler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

func (h *S3Handler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

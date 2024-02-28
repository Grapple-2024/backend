package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
)

const (
	gymVideosBucket = "grapple-gym-videos"
	coachGroupARN   = "arn:aws:iam::381491926210:role/us-west-1_HT5oR6AwO-coachGroupRole"
)

type S3Handler struct {
	*s3.PresignClient
}

func NewS3Handler(ctx context.Context, region string) (*S3Handler, error) {
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)
	return &S3Handler{
		PresignClient: psc,
	}, nil
}

func (h *S3Handler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error) {
	token, err := ValidateJWT(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("could not validate token: %v", err))
	}
	if !slices.Contains(token.Roles, coachGroupARN) {
		return lambda.ClientError(http.StatusForbidden, "permission denied, user is not a coach")
	}

	gym := req.QueryStringParameters["gym"]
	ttl := req.QueryStringParameters["ttl"]
	key := req.QueryStringParameters["key"]

	// check for empty required parameter
	if key == "" || gym == "" {
		return lambda.ClientError(http.StatusNotFound, "must specify ?key&gym query string parameters")
	}

	// set default ttl
	if ttl == "" {
		ttl = "5m"
	}

	ttlDur, err := time.ParseDuration(ttl)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("ttl must be an integer: %v", err))
	}

	finalKey := fmt.Sprintf("%s/%s", gym, key)
	r, err := h.createPresignedUploadURL(gymVideosBucket, finalKey, ttlDur)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error creating presigned url: %v", err))
	}

	bytes, err := json.Marshal(r)
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

// Needed to satisfy interface, but not implemented

func (h *S3Handler) ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error) {
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

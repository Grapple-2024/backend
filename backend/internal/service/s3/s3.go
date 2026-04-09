package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/pkg/aws/s3"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"
)

// Service is the object that handles the business logic of all S3 multi-part upload related operations.
type Service struct {
	*s3.Client
	videosBucketName string
}

// NewService creates a new instance of s3 HTTP handler
func NewService(ctx context.Context, s3Client *s3.Client, videosBucketName string) (*Service, error) {
	svc := &Service{
		Client:           s3Client,
		videosBucketName: videosBucketName,
	}

	return svc, nil
}

// ProcessGetAll handles starting a multi-part upload request
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	if !strings.HasSuffix(req.Path, "/start-upload") {
		return lambda.ClientError(400, fmt.Sprintf("Endpoint not %s found, usage: GET /s3/start-upload", req.Path))
	}

	file := req.QueryStringParameters["file"]
	gymID := req.QueryStringParameters["gym_id"]
	seriesID := req.QueryStringParameters["series_id"]
	contentType := req.QueryStringParameters["content_type"]

	uploadPath := fmt.Sprintf("gyms/%s/series/%s/video/%d_%s", gymID, seriesID, time.Now().UnixNano(), file)

	log.Info().Msgf("Upload path generated %s", uploadPath)
	if contentType == "" {
		return lambda.ClientError(400, "must specify content_type query param: ?uploadPath=gyms/videos/{id}&content_type=video/mp4", req.Path)
	}

	resp, err := s.Client.StartMultipartUpload(ctx, s.videosBucketName, uploadPath, contentType)
	if err != nil {
		return lambda.ClientError(400, err.Error())
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return lambda.ClientError(400, err.Error())
	}

	return lambda.NewResponse(http.StatusOK, string(respBytes), nil), nil
}

// ProcessPost completes a multi-part upload given an upload ID
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if !strings.HasSuffix(req.Path, "/complete-upload") {
		return lambda.ClientError(400, fmt.Sprintf("Endpoint not %s found, usage: POST /s3/complete-upload", req.Path))
	}

	uploadID := req.QueryStringParameters["upload_id"]
	uploadPath := req.QueryStringParameters["upload_path"]
	resp, err := s.Client.CompleteUpload(ctx, &s3.CompleteUploadRequest{
		BucketName: s.videosBucketName,
		UploadID:   uploadID,
		UploadPath: uploadPath,
	})
	if err != nil {
		return lambda.ClientError(400, err.Error())
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return lambda.ClientError(400, err.Error())
	}

	return lambda.NewResponse(http.StatusOK, string(respBytes), nil), nil
}

// Unimplemented methods
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

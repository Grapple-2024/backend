package mapbox

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"
)

// Service is the object that handles the business logic of all Profile related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD Profile objects.
type Service struct {
	http.Client

	mapboxAPIKey string
}

// NewService creates a new instance of a Profile Service given a mongo client
func NewService(ctx context.Context, mapboxAPIKey string) (*Service, error) {
	svc := &Service{
		mapboxAPIKey: mapboxAPIKey,
	}

	log.Info().Msgf("map box api key: %v", mapboxAPIKey)

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /mapbox
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	resp, err := s.forwardGeocode(req.QueryStringParameters["q"])
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to calculate coordinates from search query: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

func (s *Service) forwardGeocode(searchQuery string) ([]byte, error) {
	url, err := url.Parse(`https://api.mapbox.com/search/geocode/v6/forward`)
	if err != nil {
		return nil, err
	}
	query := url.Query()
	query.Set("q", searchQuery)
	query.Set("access_token", s.mapboxAPIKey)
	url.RawQuery = query.Encode()

	req, err := http.NewRequest(http.MethodGet, url.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := s.Do(req)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(resp.Body)
}

// Unused but needed for interface implementation
// ProcessGet handles HTTP requests for GET /mapbox/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPost is a no-operation. Updating and inserting a profile is handled via PUT /profiles
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// ProcessPut handles HTTP requests for
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

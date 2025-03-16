package lambda

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"
)

const (
	exclusiveStartKeyKey = "exclusiveStartKey"
	// exclusiveStartKeySK = "exclusiveStartKeySK"
	limit = "limit"
)

type Lambda interface {
	ProcessGetByID(context.Context, events.APIGatewayProxyRequest, string) (events.APIGatewayProxyResponse, error)
	ProcessGetAll(context.Context, events.APIGatewayProxyRequest, int32) (events.APIGatewayProxyResponse, error)
	ProcessPost(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
	ProcessPut(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
	ProcessDelete(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
}

// NewRouter creates the main HTTP listener for Grapple backend, given a slice of Lambda endpoints to be registered.
// Each lambda represents a subpath on the backend API, ie /users, /gyms, /announcements, etc.
func NewRouter(lambdas map[string]Lambda) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		s := strings.Split(strings.TrimPrefix(req.Path, "/"), "/")
		if len(s) == 0 {
			return ClientError(http.StatusBadRequest, "bad request path: "+req.Path)
		}
		endpoint := s[0]

		handler := lambdas[endpoint]
		if handler == nil {
			return ClientError(http.StatusNotFound, fmt.Sprintf("endpoint not registered: %s", endpoint))
		}

		switch req.HTTPMethod {
		case http.MethodGet:
			return ProcessGet(ctx, handler, req)
		case http.MethodPost:
			return handler.ProcessPost(ctx, req)
		case http.MethodDelete:
			return handler.ProcessDelete(ctx, req)
		case http.MethodPut:
			return handler.ProcessPut(ctx, req)
		default:
			return ClientError(http.StatusMethodNotAllowed, http.StatusText(http.StatusMethodNotAllowed))
		}
	}
}

func ProcessGet(ctx context.Context, l Lambda, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if ok {
		return l.ProcessGetByID(ctx, req, id)
	}

	limitInt := 50
	limit, ok := req.QueryStringParameters[limit]
	if ok {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for limit parameter: %q is not an integer", limit))
		}
	}

	// TODO: remove the dynamodb from the function after switching off dynamo
	return l.ProcessGetAll(ctx, req, int32(limitInt))
}

// helper functions below this point
func NewResponse(statusCode int, body string, additionalHeaders map[string]string) events.APIGatewayProxyResponse {
	resp := events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST, GET, DELETE, PUT, HEAD, OPTIONS",
			"Access-Control-Allow-Headers": "*",
		},
		Body:       body,
		StatusCode: statusCode,
	}

	for k, v := range additionalHeaders {
		if _, ok := resp.Headers[k]; ok {
			continue // do not allow the lambda to overwrite CORS headers
		}
		resp.Headers[k] = v
	}

	return resp
}

func ClientError(status int, msgs ...string) (events.APIGatewayProxyResponse, error) {
	respBytes, err := json.Marshal(map[string]any{
		"error": msgs,
	})
	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}
	return NewResponse(status, string(respBytes), nil), nil
}

func ServerError(err error) (events.APIGatewayProxyResponse, error) {
	log.Error().Err(err).Msgf("Server error: %v", err.Error())

	// TODO: We shouldn't return server-sided error messages to the consumer of Grapple backend.
	// Return user-friendly error message like "A system error has occured, please try again or contact your system admin."
	resp := map[string]any{
		"error": err.Error(),
	}

	respBytes, err := json.Marshal(resp)
	if err != nil {
		return events.APIGatewayProxyResponse{}, nil
	}

	return NewResponse(http.StatusInternalServerError, string(respBytes), nil), nil
}

package lambda

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/rs/zerolog/log"
)

const (
	exclusiveStartKey = "exclusiveStartKey"
	limit             = "limit"
)

type Lambda interface {
	ProcessGetByID(ctx context.Context, id string) (events.APIGatewayProxyResponse, error)
	ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, exclusiveStartKey *string) (events.APIGatewayProxyResponse, error)
	ProcessPost(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
	ProcessPut(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
	ProcessDelete(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
}

func NewRouter(lambdas map[string]Lambda) func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Main handler function for all HTTP requests on this Lambda API.
	return func(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
		base := strings.Split(strings.TrimPrefix(req.Path, "/"), "/")[0]
		log.Info().Msgf("base: %v", base)

		handler := lambdas[base]

		log.Info().Msgf("Handler: %v", handler)
		if handler == nil {
			return ClientError(http.StatusNotFound, http.StatusText(http.StatusNotFound))
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
		return l.ProcessGetByID(ctx, id)
	}

	exclusiveStartKey, _ := req.QueryStringParameters[exclusiveStartKey]
	limitInt := 50
	limit, ok := req.QueryStringParameters[limit]
	if ok {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for limit parameter: %q is not an integer", limit))
		}
	}

	return l.ProcessGetAll(ctx, req, int32(limitInt), &exclusiveStartKey)
}

// helper functions below this point
func NewResponse(statusCode int, body string, additionalHeaders map[string]string) events.APIGatewayProxyResponse {
	resp := events.APIGatewayProxyResponse{
		Headers: map[string]string{
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "POST,OPTIONS,GET,DELETE",
			"Access-Control-Allow-Headers": "X-Amz-Date,X-Api-Key,X-Amz-Security-Token,X-Requested-With,X-Auth-Token,Referer,User-Agent,Origin,Content-Type,Authorization,Accept,Access-Control-Allow-Methods,Access-Control-Allow-Origin,Access-Control-Allow-Headers",
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

func ClientError(status int, msg string) (events.APIGatewayProxyResponse, error) {
	return NewResponse(status, msg, nil), nil
}

func ServerError(err error) (events.APIGatewayProxyResponse, error) {
	log.Error().Err(err).Msgf("Server error: %v", err.Error())
	return NewResponse(http.StatusInternalServerError, fmt.Sprintf("Internal Server Error: %v", err), nil), nil
}

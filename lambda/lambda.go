package lambda

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
)

const (
	exclusiveStartKeyPK = "exclusiveStartKeyPK"
	exclusiveStartKeySK = "exclusiveStartKeySK"
	limit               = "limit"
)

type Lambda interface {
	ProcessGetByID(context.Context, events.APIGatewayProxyRequest, string) (events.APIGatewayProxyResponse, error)
	ProcessGetAll(context.Context, events.APIGatewayProxyRequest, int32, map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error)
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
		return l.ProcessGetByID(ctx, req, id)
	}

	startKeyPK := req.QueryStringParameters[exclusiveStartKeyPK]
	startKeySK := req.QueryStringParameters[exclusiveStartKeySK]

	limitInt := 50
	limit, ok := req.QueryStringParameters[limit]
	if ok {
		var err error
		limitInt, err = strconv.Atoi(limit)
		if err != nil {
			return ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for limit parameter: %q is not an integer", limit))
		}
	}

	var startKey *dynamodbsdk.Key
	if startKeyPK != "" {
		startKey = &dynamodbsdk.Key{
			PK: startKeyPK,
		}
	}
	if startKeySK != "" {
		startKey.SK = startKeySK
	}

	fmt.Printf("Start Key: %v\n", startKey)
	var av map[string]types.AttributeValue
	var err error
	if startKey != nil {
		av, err = attributevalue.MarshalMap(startKey)
		if err != nil {
			return ClientError(http.StatusBadRequest, fmt.Sprintf("invalid exclusive start key: %v", err))
		}
	}

	fmt.Printf("AV: %+v\n", av)

	return l.ProcessGetAll(ctx, req, int32(limitInt), av)
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

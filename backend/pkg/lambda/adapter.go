package lambda

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
)

// Adapter wraps a Lambda router function as a standard net/http handler.
// It translates between net/http requests/responses and APIGatewayProxyRequest/Response.
type Adapter struct {
	handler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)
}

// NewAdapter wraps the Lambda router returned by NewRouter into an http.Handler.
func NewAdapter(handler func(context.Context, events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error)) *Adapter {
	return &Adapter{handler: handler}
}

func (a *Adapter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, _ := io.ReadAll(r.Body)

	headers := make(map[string]string, len(r.Header))
	for k, vs := range r.Header {
		headers[k] = vs[0]
	}

	queryParams := make(map[string]string, len(r.URL.Query()))
	for k, vs := range r.URL.Query() {
		queryParams[k] = vs[0]
	}

	// Extract path parameters: /endpoint/id → PathParameters["id"] = id
	pathParams := map[string]string{}
	segments := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(segments) >= 2 && segments[1] != "" {
		pathParams["id"] = segments[1]
	}

	req := events.APIGatewayProxyRequest{
		HTTPMethod:            r.Method,
		Path:                  r.URL.Path,
		Headers:               headers,
		QueryStringParameters: queryParams,
		PathParameters:        pathParams,
		Body:                  string(body),
	}

	resp, err := a.handler(r.Context(), req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for k, v := range resp.Headers {
		w.Header().Set(k, v)
	}
	w.WriteHeader(resp.StatusCode)
	w.Write([]byte(resp.Body)) //nolint:errcheck
}

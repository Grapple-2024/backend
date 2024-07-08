package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type EmailHandler struct {
	*dynamodbsdk.Client
	emailsTable string
}

type Email struct {
	PK        string    `json:"-" dynamodbav:"pk"`
	Email     string    `json:"email" dynamodbav:"email"`
	Dummy     string    `json:"-" dynamodbav:"dummy"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func NewEmailHandler(ctx context.Context, dynamoEndpoint string) (*EmailHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}
	return &EmailHandler{
		Client:      db,
		emailsTable: os.Getenv("EMAILS_TABLE_NAME"),
	}, nil
}

func (h *EmailHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// ascending := parseBool(req.QueryStringParameters["ascending"], true)

	// Send Query request to DynamoDB
	scanLimit := limit + 1000
	result, err := h.Scan(ctx, &dynamodb.ScanInput{
		TableName: &h.emailsTable,
		Limit:     &scanLimit,
		// ScanIndexForward: &ascending,
		// IndexName:        aws.String("CreatedAtIndex"),
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error querying table: %v", err))
	}

	var emails []Email
	resp, err := dynamodbsdk.MarshalResponse(
		aws.String("created_at"), limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &emails,
	)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("bad request: %v", err))
	}

	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *EmailHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (h *EmailHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Unmarshal request body
	var email Email
	if err := json.Unmarshal([]byte(req.Body), &email); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	// Validate request body
	if err := validate.Struct(&email); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}
	email.PK = email.Email
	email.Dummy = "dumb"
	email.CreatedAt = time.Now().UTC()

	// Insert Gym into dynamodb
	_, err := h.Insert(ctx, h.emailsTable, &email)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	json, err := json.Marshal(&email)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *EmailHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	result, err := h.GetByID(ctx, h.emailsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym not found")
	}

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}
	resp, err := h.Delete(ctx, h.emailsTable, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *EmailHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

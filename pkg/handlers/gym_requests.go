package handlers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	dynamodbsdk "github.com/Grapple-2024/backend/pkg/dynamodb"
	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

const (
	StatusPending  = "Pending"
	StatusAccepted = "Accepted"
	StatusDenied   = "Denied"
)

type GymRequestHandler struct {
	*dynamodbsdk.Client
	*AuthService
	sendgridClient *sendgrid.Client

	requestsTable string
}

type GymRequest struct {
	PK    string `json:"pk" dynamodbav:"pk"`
	GymID string `json:"gym_id" dynamodbav:"gym_id,omitempty"`

	RequestorID    string `json:"requestor_id" dynamodbav:"requestor_id"`
	RequestorEmail string `json:"requestor_email" dynamodbav:"requestor_email"`

	FirstName string `json:"first_name" dynamodbav:"first_name,omitempty"`
	LastName  string `json:"last_name" dynamodbav:"last_name,omitempty"`
	Email     string `json:"email" dynamodbav:"email,omitempty"`

	Status string `json:"status" dynamodbav:"status,omitempty"`

	Dummy     string    `json:"-" dynamodbav:"dummy"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
}

func NewGymRequestHandler(ctx context.Context, dynamoEndpoint, sendGridAPIKey string) (*GymRequestHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &GymRequestHandler{
		Client:         db,
		AuthService:    authSVC,
		sendgridClient: sendgrid.NewSendClient(sendGridAPIKey),
		requestsTable:  os.Getenv("GYM_REQUESTS_TABLE_NAME"),
	}, nil
}

func (h *GymRequestHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	requestor := req.QueryStringParameters["requestor"]
	gym := req.QueryStringParameters["gym"]
	status := req.QueryStringParameters["status"]
	ascending := parseBool(req.QueryStringParameters["ascending"], true)

	builder := expression.NewBuilder().
		WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))

	filter := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"gym_id": {
			Value:    gym,
			Operator: "Equal",
		},
		"requestor_id": {
			Value:    requestor,
			Operator: "Equal",
		},
		"status": {
			Value:    status,
			Operator: "Equal",
		},
	})
	if filter != nil {
		builder = builder.WithFilter(*filter)
	}

	expr, err := builder.Build()
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to build expression: %v", err))
	}

	// Send Query request to DynamoDB
	scanLimit := limit + 1000
	result, err := h.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &h.gymRequestsTable,
		Limit:                     &scanLimit,
		ExclusiveStartKey:         startKey,
		ScanIndexForward:          &ascending,
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		IndexName:                 aws.String("CreatedAtIndex"),
	})
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error querying table: %v", err))
	}

	var gymRequests []GymRequest
	resp, err := dynamodbsdk.MarshalResponse(
		aws.String("created_at"), limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &gymRequests,
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

func (h *GymRequestHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymRequests by ID request")

	result, err := h.GetByID(ctx, h.requestsTable, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	var requests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}
	if len(requests) == 0 {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("no request found with id %s", id))
	}

	json, err := json.Marshal(requests[0])
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil

}

func (h *GymRequestHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// unmarshal request body into GymRequest struct and validate input
	var gymRequest GymRequest
	if err := json.Unmarshal([]byte(req.Body), &gymRequest); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}
	err = validate.Struct(&gymRequest)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("request body failed validation: %v", err))
	}

	// Fetch the Gym that the Gym Request is referencing. Confirm that it exists. Return 400 bad request if it does not exist.
	if _, err := h.GetByID(ctx, h.gymsTable, gymRequest.GymID); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("error fetching gym by ID. make sure the gym_id you specified is a valid Gym ID: %v", err))
	}

	// create the request
	gymRequest.PK = base64.URLEncoding.EncodeToString([]byte(fmt.Sprintf("gymRequest#%s/%s", token.Sub, gymRequest.GymID)))
	gymRequest.RequestorID = token.Sub
	gymRequest.RequestorEmail = token.Email
	gymRequest.Status = StatusPending
	gymRequest.Dummy = "dumb"
	gymRequest.CreatedAt = time.Now().UTC()

	res, err := h.Insert(ctx, h.requestsTable, &gymRequest, "pk")
	if err != nil {
		return lambda.ServerError(err)
	}

	var returnRequest GymRequest
	err = attributevalue.UnmarshalMap(res.Attributes, &returnRequest)
	if err != nil {
		return lambda.ServerError(err)
	}

	if err := h.notifyCoach(ctx, &gymRequest); err != nil {
		log.Warn().Msgf("Failed to notify coach by email of new gym request for Gym ID %q: %v", gymRequest.GymID, err)
	}

	json, err := json.Marshal(&gymRequest)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymRequestHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	// Fetch the Gym Request
	result, err := h.GetByID(ctx, h.requestsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym request not found")
	}

	var requests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}
	requestor := requests[0].RequestorID
	if requestor != token.Sub {
		return lambda.ClientError(http.StatusForbidden, "permission denied: you must be the creator of the gym request to delete it")
	}

	log.Printf("Received DELETE request with id = %s", id)

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Delete(ctx, h.requestsTable, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}
func (h *GymRequestHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	// Fetch the Gym Request the user is trying to modify
	result, err := h.GetByID(ctx, h.requestsTable, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	} else if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found with id %v", id))
	}

	var requests []GymRequest
	err = attributevalue.UnmarshalListOfMaps(result.Items, &requests)
	if err != nil {
		return lambda.ServerError(err)
	}

	// check if the token has permission to modify the request (They must be the gym's coach)
	log.Info().Msgf("Checking if user request token is associated with the coach of gym %++v", requests[0])
	if err := h.IsCoach(ctx, req.Headers, requests[0].GymID); err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: must be a coach to modify a gym request: %v", err))
	}

	var payload GymRequest
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	if payload.Status != StatusAccepted && payload.Status != StatusPending && payload.Status != StatusDenied {
		return lambda.ClientError(http.StatusBadRequest, "status field must be one of [Accepted, Pending, Denied] %v")
	}

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	)

	if payload.Status != "" {
		builder.WithUpdate(
			expression.Set(
				expression.Name("status"),
				expression.Value(payload.Status),
			))
	}

	expr, err := builder.Build()
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "bad request payload")
	}

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Update(ctx, h.requestsTable, key, &expr, false)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var gymRequest GymRequest
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gymRequest); err != nil {
		return lambda.ServerError(err)
	}

	log.Info().Msgf("Gym request: %v", gymRequest)
	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// email template for gym announcement notifications
const requestNotificationEmail = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Grapple MMA: a student requests to join your gym</title>
    <style>
        body {
            font-family: Arial, sans-serif;
            background-color: #F24B4B;
            color: #000;
            padding: 20px;
            margin: 0;
        }
        .container {
            max-width: 600px;
            margin: 0 auto;
			background-color: #3E3E3E; /* Darker background color for contrast */
            padding: 20px;
            border-radius: 8px;
        }
        .header {
            text-align: center;
        }
        .content {
            text-align: center;
        }
        .content p {
            font-size: 18px;
            color: #F24B4B;
            margin: 10px 0;
        }
        .footer {
            margin-top: 20px;
            font-size: 14px;
            text-align: center;
            color: #888;
        }
        .unsubscribe {
            margin-top: 10px;
        }
        .unsubscribe a {
            color: #888;
            text-decoration: none;
        }
		.separator {
            border-top: 1px solid #888; /* Separator line color */
            margin: 20px 0; /* Adjust margin as needed */
        }
    </style>
</head>
<body>

    <div class="container">
        <div class="header">
            <img src="https://grapplemma.com/logo.png" alt="Grapple MMA Logo">
        </div>
		<hr class="separator">
        <div class="content">
            <p style="color: white"> A student has requested to join your gym, %s. The user's email address is: %s</p>
            <p style="font-style: italic;">Please <a href="https://grapplemma.com/coach/my-gym">login</a> to Grapple to approve or deny the student access to your gym.</p>
        </div>
		
        <div class="footer">
			<p>Grapple MMA</p>
            <p>2702 Cepa Uno, San Clemente, California 92673</p>
            <div class="unsubscribe">
                <p>To unsubscribe from future emails, <a href="#">click here</a>.</p>
            </div>
            <p>This email was sent in compliance with United States and California anti-spam laws.</p>
        </div>
    </div>

</body>
</html>`

// notifyCoach sends an email to the coach when a student requests to join their gym.
func (h *GymRequestHandler) notifyCoach(ctx context.Context, r *GymRequest) error {
	// Get the Gym associated with this announcement
	gymID, err := attributevalue.Marshal(r.GymID)
	if err != nil {
		return err
	}
	gymPK := map[string]types.AttributeValue{
		"pk": gymID,
	}
	o, err := h.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &h.gymsTable,
		Key:       gymPK,
	})
	if err != nil {
		return fmt.Errorf("failed to get item in table %s: %v", h.gymsTable, err)
	}

	var gym Gym
	if err = attributevalue.UnmarshalMap(o.Item, &gym); err != nil {
		return err
	}

	// Craft email content
	subject := fmt.Sprintf("Grapple MMA | %s: a student has requested to join your gym", gym.Name)
	from := mail.NewEmail("Grapple Notifications", "support@grapplemma.com")
	to := mail.NewEmail("Grapple Coach", gym.CoachEmail)
	htmlContent := fmt.Sprintf(requestNotificationEmail, gym.Name, r.RequestorEmail)
	message := mail.NewSingleEmail(from, subject, to, "", htmlContent)

	// Send email
	log.Info().Msgf("Notifying coach of new request: %+v", message)
	_, err = h.sendgridClient.Send(message)
	if err != nil {
		log.Warn().Msgf("Failed to send email notification to student: %v", err)
	}

	return nil
}

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

	"github.com/Grapple-2024/backend/pkg/cognito"
	dynamodbsdk "github.com/Grapple-2024/backend/pkg/dynamodb"

	"github.com/Grapple-2024/backend/pkg/lambda"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
)

type GymAnnouncementHandler struct {
	*AuthService
	*dynamodbsdk.Client
	CognitoClient *cognito.Client

	sendgridClient         *sendgrid.Client
	announcementsTableName string
	userProfilesTableName  string
}

type GymAnnouncement struct {
	PK string `json:"pk" dynamodbav:"pk"`

	GymID     string    `json:"gym_id" dynamodbav:"gym_id"`
	Title     string    `json:"title" dynamodbav:"title,omitempty"`
	Content   string    `json:"content" dynamodbav:"content,omitempty"`
	CreatedAt time.Time `json:"created_at" dynamodbav:"created_at"`
	UpdatedAt time.Time `json:"updated_at" dynamodbav:"updated_at"`
	Dummy     string    `json:"-" dynamodbav:"dummy,omitempty"`
}

func NewGymAnnouncementHandler(ctx context.Context, dynamoEndpoint, sendGridAPIKey, cognitoClientID, cognitoClientSecret string) (*GymAnnouncementHandler, error) {
	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	cc, err := cognito.NewClient(
		region,
		cognito.WithClientID(cognitoClientID),
		cognito.WithClientSecret(cognitoClientSecret),
	)
	if err != nil {
		return nil, err
	}

	return &GymAnnouncementHandler{
		Client:                 db,
		AuthService:            authSVC,
		CognitoClient:          cc,
		sendgridClient:         sendgrid.NewSendClient(sendGridAPIKey),
		announcementsTableName: os.Getenv("GYM_ANNOUNCEMENTS_TABLE_NAME"),
		userProfilesTableName:  os.Getenv("USER_PROFILES_TABLE_NAME"),
	}, nil
}

func (h *GymAnnouncementHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	gym := req.QueryStringParameters["gym"]
	if gym == "" {
		return lambda.ClientError(http.StatusBadRequest, "?gym query parameter is required")
	}
	ascending := parseBool(req.QueryStringParameters["ascending"], true)

	// check permissions
	isNotCoach := h.IsCoach(ctx, req.Headers, gym)
	isNotStudent := h.IsStudent(ctx, req.Headers, gym)
	if isNotCoach != nil && isNotStudent != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("permission denied: %v\n %v", isNotStudent, isNotCoach))
	}

	// Build the filter and key expressions
	builder := expression.NewBuilder().WithKeyCondition(expression.Key("dummy").Equal(expression.Value("dumb")))
	filterExpr := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"gym_id": {
			Operator: "Equal",
			Value:    gym,
		},
	})
	if filterExpr != nil {
		builder = builder.WithFilter(*filterExpr)
	}

	expr, err := builder.Build()
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to build expression: %v", err))
	}

	input := &dynamodb.QueryInput{
		TableName:                 &h.announcementsTableName,
		IndexName:                 aws.String("LastUpdatedIndex"),
		ScanIndexForward:          &ascending,
		KeyConditionExpression:    expr.KeyCondition(),
		FilterExpression:          expr.Filter(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),
		Limit:                     &limit,
	}
	if _, ok := startKey["pk"]; ok {
		input.ExclusiveStartKey = startKey
	}

	result, err := h.Query(ctx, input)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to query table: %v", err))
	}

	var gymAnnouncements []GymAnnouncement
	resp, err := dynamodbsdk.MarshalResponse(
		aws.String("updated_at"), limit, result.Count, result.ScannedCount, result.LastEvaluatedKey, result.Items, &gymAnnouncements,
	)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("err marshalling response: %v", err))
	}

	json, err := json.Marshal(resp)
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched GymAnnouncement item %s", json)

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	log.Print("Received GET gymAnnouncements by ID request")

	result, err := h.GetByID(ctx, h.announcementsTableName, id)
	if err != nil {
		return lambda.ServerError(err)
	}

	var announcements []GymAnnouncement
	err = attributevalue.UnmarshalListOfMaps(result.Items, &announcements)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(announcements[0])
	if err != nil {
		return lambda.ServerError(err)
	}
	log.Printf("Successfully fetched Gyms by ID: %s", string(json))

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil

}

func (h *GymAnnouncementHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var announcement GymAnnouncement
	if err := json.Unmarshal([]byte(req.Body), &announcement); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("request body invalid: %v", req.Body))
	}

	// if err := h.IsCoach(ctx, req.Headers, gymAnnouncement.GymID); err != nil {
	// 	// user is not a coach of this gym, deny the request to create an announcement
	// 	return lambda.ClientError(http.StatusForbidden, err.Error())
	// }

	if err := validate.Struct(&announcement); err != nil {
		return lambda.ClientError(http.StatusBadRequest, "request body failed validation")
	}

	// Create the Gym Announcement record
	announcement.CreatedAt = time.Now().UTC()
	announcement.UpdatedAt = announcement.CreatedAt
	announcement.Dummy = "dumb"
	announcement.PK = base64.URLEncoding.EncodeToString([]byte(
		fmt.Sprintf("gymAnnouncement#%s/%d", announcement.GymID, announcement.CreatedAt.Unix())),
	)

	r, err := h.Insert(ctx, h.announcementsTableName, &announcement, "pk")
	if err != nil {
		return lambda.ServerError(err)
	}

	// notify students of the new announcement via email
	if err := h.notifyStudents(ctx, &announcement); err != nil {
		log.Warn().Err(err).Msgf("Failed to send email notification for announcement!")
	}

	var returnGym GymAnnouncement
	err = attributevalue.UnmarshalMap(r.Attributes, &returnGym)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&announcement)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Get request ID path parameter
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "bad request: id parameter not found in path")
	}

	// Fetch the Gym Request
	result, err := h.GetByID(ctx, h.announcementsTableName, id)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("gym request not found: %v", err))
	}
	if result.Count == 0 {
		return lambda.ClientError(http.StatusNotFound, "gym request not found")
	}

	var announcements []GymAnnouncement
	if err := attributevalue.UnmarshalListOfMaps(result.Items, &announcements); err != nil {
		return lambda.ServerError(err)
	}

	log.Printf("Received DELETE request with id = %s", id)

	pk, err := attributevalue.Marshal(id)
	if err != nil {
		return lambda.ServerError(err)
	}
	key := map[string]types.AttributeValue{
		"pk": pk,
	}

	resp, err := h.Delete(ctx, h.announcementsTableName, key)
	if err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *GymAnnouncementHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id, ok := req.PathParameters["id"]
	if !ok {
		return lambda.ClientError(http.StatusBadRequest, "id parameter not found")
	}

	var payload GymAnnouncement
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to unmarshal request body: %v", err))
	}

	// Marshal to AV
	av, _ := attributevalue.MarshalMap(payload)
	update := expression.UpdateBuilder{}
	for k, v := range av {
		if k == "pk" || k == "gym_id" || k == "created_at" || k == "updated_at" || k == "title" || k == "dummy" {
			continue
		}
		update = update.Set(expression.Name(k), expression.Value(v))
	}
	log.Info().Msgf("Update query: %+v", update)

	builder := expression.NewBuilder().WithCondition(
		expression.Equal(
			expression.Name("pk"),
			expression.Value(id),
		),
	).WithUpdate(update)

	// Update the timestamp on the announcement
	payload.UpdatedAt = time.Now().UTC()

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

	resp, err := h.Update(ctx, h.announcementsTableName, key, &expr, false)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to update record: %v", err))
	}

	var gymAnnouncement GymAnnouncement
	if err := attributevalue.UnmarshalMap(resp.Attributes, &gymAnnouncement); err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(resp.Attributes)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// email template for gym announcement notifications
const announcementNotificationEmail = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Grapple MMA: %s posted a new announcement</title>
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
            <p style="color: white"> A new announcement was posted by %s:</p>
            <p style="font-style: italic;">%s</p>
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

// notifyStudents sends an email to each student of a gym. This function is called on successful announcement creation.
func (h *GymAnnouncementHandler) notifyStudents(ctx context.Context, a *GymAnnouncement) error {
	// Get the Gym associated with this announcement
	gymID, err := attributevalue.Marshal(a.GymID)
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
		return err
	}

	var gym Gym
	if err = attributevalue.UnmarshalMap(o.Item, &gym); err != nil {
		return err
	}

	// Get all GymRequest objects for this gym with status=Accepted
	builder := expression.NewBuilder().WithKeyCondition(
		expression.Key("gym_id").Equal(expression.Value(a.GymID)),
	)
	filterExpr := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"status": {
			Operator: "Equal",
			Value:    "Accepted",
		},
	})
	builder = builder.WithFilter(*filterExpr)

	e, err := builder.Build()
	if err != nil {
		return err
	}

	qo, err := h.Client.Query(ctx, &dynamodb.QueryInput{
		TableName:                 &h.gymRequestsTableName,
		IndexName:                 aws.String("GymIndex"),
		KeyConditionExpression:    e.KeyCondition(),
		FilterExpression:          e.Filter(),
		ExpressionAttributeNames:  e.Names(),
		ExpressionAttributeValues: e.Values(),
	})
	if err != nil {
		return fmt.Errorf("failed to query gym requests table %q: %v", h.gymRequestsTableName, err)
	}

	var requests []GymRequest
	if err = attributevalue.UnmarshalListOfMaps(qo.Items, &requests); err != nil {
		return err
	}

	// Send email via AWS SES
	subject := fmt.Sprintf("Grapple MMA | %s: %s", gym.Name, a.Title)
	for _, r := range requests {
		// get UserPreferences object for this user
		up, err := h.getUserProfile(ctx, r.RequestorID)
		if err != nil {
			log.Warn().Err(err).Msgf("Unable to find user preferences for user ID %q", r.RequestorID)
		}
		log.Info().Msgf("Got user profile: %+v for user id %s", up, r.RequestorID)

		// skip the student if they did not opt into emails
		if !up.NotifyOnAnnouncements {
			log.Info().Msgf("User %s has opted out of announcement notifications, no email will be sent!", r.RequestorEmail)
			continue
		}
		log.Info().Msgf("Notifying student email address %s of announcement", r.RequestorEmail)

		from := mail.NewEmail("Grapple Notifications", "support@grapplemma.com")
		to := mail.NewEmail("Grapple Student", r.RequestorEmail)
		htmlContent := fmt.Sprintf(announcementNotificationEmail, gym.Name, gym.Name, a.Content)
		message := mail.NewSingleEmail(from, subject, to, "", htmlContent)

		// Send email
		_, err = h.sendgridClient.Send(message)
		if err != nil {
			log.Warn().Msgf("Failed to send email notification to student: %v", err)
		}

	}

	return nil
}

func (h *GymAnnouncementHandler) getUserProfile(ctx context.Context, id string) (*UserProfile, error) {
	// construct key for the object to fetch
	userID, err := attributevalue.Marshal(id)
	if err != nil {
		return nil, err
	}
	key := map[string]types.AttributeValue{
		"user_id": userID,
	}

	qo, err := h.Client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: &h.userProfilesTableName,
		Key:       key,
	})
	if err != nil {
		return nil, err
	}

	var up UserProfile
	if err = attributevalue.UnmarshalMap(qo.Item, &up); err != nil {
		return nil, err
	}

	return &up, nil
}

package announcements

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	lambda "github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/mgo.v2/bson"
)

//go:embed templates/new_announcement.html
var newAnnouncementEmailTmpl string

// Service is the object that handles the business logic of all announcement related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD announcement objects.
type Service struct {
	*rbac.RBAC
	*mongoext.Client
	*mongo.Collection

	sendGridClient  *sendgrid.Client
	profilesService *profiles.Service
}

// NewService creates a new instance of a Announcement Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, sendGridClient *sendgrid.Client, profilesService *profiles.Service, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("announcements")

	// Create unique index for announcement names
	svc := &Service{
		RBAC:            rbac,
		Client:          mc,
		Collection:      c,
		sendGridClient:  sendGridClient,
		profilesService: profilesService,
	}
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /announcements/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// Parse filter query params
	showByWeek := req.QueryStringParameters["show_by_week"]
	gymID := req.QueryStringParameters["gym_id"]
	if gymID == "" {
		return lambda.ClientError(http.StatusBadRequest, "?gym_id=<gym_id query parameter is required")

	}

	// check permissions for read
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, gymID, rbac.ResourceAnnouncements)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionRead)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s': %v", rbac.ActionRead, resourceID, err),
		)
	}

	// parse pagination query params
	page := req.QueryStringParameters["page"]
	if page == "" {
		page = "1" // default to first page
	}
	pageSize := req.QueryStringParameters["page_size"]
	if pageSize == "" {
		pageSize = "10" // default to 10 records per page
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil && pageSize != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page_size query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// create the filter based on query parameters in the request
	filter := bson.M{}
	gymObjID, err := primitive.ObjectIDFromHex(gymID)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
	}
	filter["gym_id"] = gymObjID

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
		}
		year, week := time.ISOWeek()
		filter["created_at_year"] = year
		filter["created_at_week"] = week
	}

	// Fetch records with pagination
	var records []Announcement
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, true, &records); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []Announcement{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("announcements", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /announcements/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// Get the announcement by ID
	var announcement Announcement
	if err := mongoext.FindByID(ctx, s.Collection, id, &announcement); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find announcement by ID: %v", err))
	}

	// check permissions for read
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, announcement.GymID, rbac.ResourceAnnouncements)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionRead)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s': %v", rbac.ActionRead, resourceID, err),
		)
	}

	// Return record as JSON
	json, err := json.Marshal(announcement)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /announcements
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var announcement Announcement
	if err := json.Unmarshal([]byte(req.Body), &announcement); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(announcement); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	// check permissions
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, announcement.GymID.Hex(), rbac.ResourceAnnouncements)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionCreate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s': %v", rbac.ActionCreate, resourceID, err),
		)
	}

	announcement.CreatedAt = time.Now().Local().UTC()
	announcement.UpdatedAt = announcement.CreatedAt
	announcement.CreatedAtYear, announcement.CreatedAtWeek = announcement.CreatedAt.ISOWeek()

	// insert the announcement, store the resulting record in 'result' variable
	var result Announcement
	if err := mongoext.Insert(ctx, s.Collection, &announcement, &result); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert record: %v", err))
	}

	// notify all students in this gym that a new announcement was posted!
	if err := s.notifyStudentsOnAnnouncement(ctx, &result); err != nil {
		log.Warn().Msgf("failed to notify students of new announcements - FAILING SILENTLY: %v", err)
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /announcements/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]

	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var payload Announcement
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	var announcement Announcement
	if err := mongoext.FindByID(ctx, s.Collection, id, &announcement); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to find announcement with id %s: %v", id, err))
	}

	// check permissions
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, announcement.GymID.Hex(), rbac.ResourceAnnouncements)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionUpdate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s': %v", rbac.ActionUpdate, resourceID, err),
		)
	}

	// Update the announcement only if the user has permission to
	announcement.UpdatedAt = time.Now().Local().UTC()

	// update the record in mongo
	var result Announcement
	if err := mongoext.UpdateByID(ctx, s.Collection, id, announcement, &result, nil); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to update announcement: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /announcements/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]

	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	var announcement Announcement
	if err := mongoext.FindByID(ctx, s.Collection, id, &announcement); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to find announcement with id %s: %v", id, err))
	}

	// check permissions
	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, announcement.GymID.Hex(), rbac.ResourceAnnouncements)
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionDelete)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s': %v", rbac.ActionDelete, resourceID, err),
		)
	}

	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	// create filter and options
	filter := bson.M{"_id": objID}
	opts := options.Delete().SetHint(bson.M{"_id": 1}) // use _id index to find object

	result, err := s.Collection.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return lambda.ServerError(err)
	}

	if result.DeletedCount == 0 {
		return lambda.NewResponse(http.StatusNotFound, ``, nil), nil
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) notifyStudentsOnAnnouncement(ctx context.Context, a *Announcement) error {
	studentRole := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, a.GymID, rbac.Students)
	// get all students for this gym
	memberships, err := s.profilesService.GetGymAssociationsBy(ctx, a.GymID.Hex(), studentRole)
	if err != nil {
		return err
	}
	if len(memberships) == 0 {
		log.Warn().Msgf("No students found for gym %q, no notifications will be sent", a.GymID)
		return nil
	}

	// fetch the gym from mongo
	gymsColl := s.Database().Collection("gyms")
	var gym *gyms.Gym
	if err := mongoext.FindByID(ctx, gymsColl, a.GymID.Hex(), &gym); err != nil {
		return err
	}

	// render email template
	tmpl, err := template.New("").Parse(newAnnouncementEmailTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var tmplOut bytes.Buffer
	tmplData := struct {
		GymName             string
		AnnouncementTitle   string
		AnnouncementContent string
	}{
		GymName:             gym.Name,
		AnnouncementTitle:   a.Title,
		AnnouncementContent: a.Content,
	}
	if err := tmpl.Execute(&tmplOut, tmplData); err != nil {
		return fmt.Errorf("error executing template with data %v:\n %v", tmplData, err)
	}

	// craft email object to send to sendgrid API
	var tos []*mail.Email
	for _, m := range memberships {
		if m.Email == "" {
			log.Warn().Msgf("Student membership found but no email: %v", m)
			continue
		}
		tos = append(tos, mail.NewEmail(profiles.StudentRole, m.Email))
	}
	email := mail.NewPersonalization()
	email.AddTos(tos...)

	// Just for debugging, all mail will be BCC'd to Jordan
	email.AddBCCs([]*mail.Email{
		mail.NewEmail("Jordan", "jordan@dionysustechnologygroup.com"),
	}...)

	payload := mail.NewV3Mail().
		AddPersonalizations(email).
		AddContent(mail.NewContent("text/html", tmplOut.String())).
		SetFrom(mail.NewEmail("Grapple Notifications", "support@grapplemma.com"))
	payload.Subject = fmt.Sprintf("%s | %s: %s", gym.Name, a.Title, a.Content)

	resp, err := s.sendGridClient.Send(payload)
	if err != nil {
		log.Warn().Msgf("Failed to send email: %v", err)
		return fmt.Errorf("failed to send email to coach: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Warn().Msgf("Failed to send email: %+v", resp)
	}

	return nil
}

// ensureIndices ensures the proper indices are created for the 'announcements' collection.
func (s *Service) ensureIndices(ctx context.Context) error {
	// Gym name index
	_, err := s.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.M{
			"gym_id": 1,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

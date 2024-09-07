package gym_requests

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

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/gyms"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/lambda_v2"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
	"gopkg.in/mgo.v2/bson"
)

//go:embed templates/new_request.html
var newRequestEmailTmpl string

//go:embed templates/request_accepted.html
var requestAcceptedEmailTmpl string

// Service is the object that handles the business logic of all gymRequest related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gymRequest objects.
type Service struct {
	mongo.Session

	*mongoext.Client
	*mongo.Collection

	sendGridClient *sendgrid.Client
}

// NewService creates a new instance of a GymRequest Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, sendGridClient *sendgrid.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("gymRequests")

	// Create Mongo Session (needed for transactions)
	svc := &Service{
		Client:         mc,
		Collection:     c,
		sendGridClient: sendGridClient,
	}

	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session

	return svc, nil
}

// ProcessGetAll handles HTTP requests for GET /gym-requests/
// TODO: remove dynamodb map after switching off fully
func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, _ map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	// Parse filter query params
	showByWeek := req.QueryStringParameters["show_by_week"]
	gymID := req.QueryStringParameters["gym_id"]
	gymStatus := req.QueryStringParameters["status"]
	requestorID := req.QueryStringParameters["requestor"]

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
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &pageSize query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	// create the filter based on query parameters in the request
	filter := bson.M{}
	if gymID != "" {
		gymObjID, err := primitive.ObjectIDFromHex(gymID)
		if err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
		}
		filter["gym_id"] = gymObjID
	}

	if gymStatus != "" {
		filter["status"] = gymStatus
	}

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
		}
		year, week := time.ISOWeek()
		filter["created_at_year"] = year
		filter["created_at_week"] = week
	}

	if requestorID != "" {
		filter["requestor_id"] = requestorID
	}

	// Fetch records with pagination
	var records []GymRequest
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, true, &records); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if records == nil {
		records = []GymRequest{}
	}

	// Get the total count of documents
	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gymRequests", records, totalCount, len(records), pageInt, pageSizeInt)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gymRequests/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gymRequest by ID
	var gymRequest GymRequest
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymRequest); err != nil {
		return lambda_v2.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymRequest by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(gymRequest)
	if err != nil {
		return lambda_v2.ServerError(err)
	}
	return lambda_v2.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gymRequests
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var payload GymRequest
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}
	payload.Status = RequestPending

	// Validate request body for required fields
	validate := validator.New()
	if err := validate.Struct(payload); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}

	gymsColl := s.Database().Collection("gyms")
	var gym *gyms.Gym
	if err := mongoext.FindByID(ctx, gymsColl, payload.GymID.Hex(), &gym); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("could not find gym with id %q: %v", payload.GymID, err))
	}

	// insert the GymRequest, store the resulting record in 'result' variable
	payload.CreatedAt = time.Now().Local().UTC()
	payload.UpdatedAt = payload.CreatedAt

	var result GymRequest
	if err := mongoext.Insert(ctx, s.Collection, &payload, &result); err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	// notify the coach by email that a new request was submitted for their gym
	if err := s.notifyCoachesOnNewRequest(ctx, &result); err != nil {
		log.Warn().Msgf("Failed to notify coach of new gym request - FAILING SILENTLY! %v", err)
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /gymRequests/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var gymRequest GymRequest
	if err := json.Unmarshal([]byte(req.Body), &gymRequest); err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	if !isValidStatus(gymRequest.Status) {
		return lambda_v2.ClientError(http.StatusBadRequest, "invalid value for status field, must be one of [Pending, Accepted, Denied]")
	}

	// update the record in mongo
	id := req.PathParameters["id"]
	result, err := s.updateGymRequestTX(ctx, &gymRequest, id)
	if err != nil {
		return lambda_v2.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to finish updateGymRequest transaction: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda_v2.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda_v2.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gymRequests/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	objID, err := primitive.ObjectIDFromHex(id)
	if err != nil {
		return lambda_v2.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object id specified in url %q: %v", id, err))
	}

	// create filter and options
	filter := bson.M{"_id": objID}
	opts := options.Delete().SetHint(bson.M{"_id": 1}) // use _id index to find object

	result, err := s.Collection.DeleteOne(context.TODO(), filter, opts)
	if err != nil {
		return lambda_v2.ServerError(err)
	}

	if result.DeletedCount == 0 {
		return lambda_v2.NewResponse(http.StatusNotFound, ``, nil), nil
	}

	return lambda_v2.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) notifyStudent(ctx context.Context, request *GymRequest) error {

	// get the profile for this student
	profilesColl := s.Client.Database("grapple").Collection("profiles")
	filter := bson.M{
		"cognito_id": request.RequestorID,
	}
	var profile profiles.Profile
	if err := mongoext.Find(ctx, profilesColl, filter, &profile); err != nil {
		return fmt.Errorf("could not find any profile with cognito id %q: %v", request.RequestorID, err)
	}

	log.Info().Msgf("Got profile for student: %v", profile)
	if !profile.NotifyOnRequestAccepted {
		log.Warn().Msgf("Student has disabled notifications for new requests, nothing will be sent")
		return nil
	}

	// execute the template for the email
	tmpl, err := template.New("").Parse(requestAcceptedEmailTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	gymsColl := s.Database().Collection("gyms")
	var gym *gyms.Gym
	if err := mongoext.FindByID(ctx, gymsColl, request.GymID.Hex(), &gym); err != nil {
		return fmt.Errorf("could not find gym with ID %v: %v", request.GymID, err)
	}

	var tmplOut bytes.Buffer
	tmplData := struct {
		GymName string
	}{
		GymName: gym.Name,
	}
	if err := tmpl.Execute(&tmplOut, tmplData); err != nil {
		return fmt.Errorf("error executing template with data %v:\n %v", tmplData, err)
	}

	subject := fmt.Sprintf("Grapple MMA: your request to join %s was accepted!", gym.Name)
	from := mail.NewEmail("Grapple Notifications", "support@grapplemma.com")
	to := mail.NewEmail("Grapple Student", request.RequestorEmail)
	message := mail.NewSingleEmail(from, subject, to, "", tmplOut.String())

	_, err = s.sendGridClient.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email to student: %v", err)
	}

	return nil
}

func (s *Service) notifyCoachesOnNewRequest(ctx context.Context, request *GymRequest) error {
	tmpl, err := template.New("").Parse(newRequestEmailTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	gymsColl := s.Database().Collection("gyms")
	var gym *gyms.Gym
	if err := mongoext.FindByID(ctx, gymsColl, request.GymID.Hex(), &gym); err != nil {
		return fmt.Errorf("could not find gym with ID %v: %v", request.GymID, err)
	}

	var tmplOut bytes.Buffer
	tmplData := struct {
		GymName      string
		StudentEmail string
	}{
		GymName:      gym.Name,
		StudentEmail: request.RequestorEmail,
	}
	if err := tmpl.Execute(&tmplOut, tmplData); err != nil {
		return fmt.Errorf("error executing template with data %v:\n %v", tmplData, err)
	}

	// get all coaches for this gym
	profilesColl := s.Client.Database("grapple").Collection("profiles")
	filter := bson.M{
		"gyms": bson.M{
			"$elemMatch": bson.M{
				"gym_id": request.GymID,
				"role":   "Coach",
			},
		},
	}
	var profiles []profiles.Profile
	if err := mongoext.Paginate(ctx, profilesColl, filter, 1, 1000, true, &profiles); err != nil {
		return fmt.Errorf("could not find any profiles that have a coach association to gym id %q %v", request.GymID, err)
	}

	log.Info().Msgf("Found profiles with coach association to gym: %v", profiles)

	var tos []*mail.Email
	for _, p := range profiles {
		for _, g := range p.Gyms {
			if g.GymID != request.GymID || !g.EmailPreferences.NotifyOnRequests {
				continue
			}
			tos = append(tos, mail.NewEmail("Grapple Coach", p.Email))
		}
	}
	if len(tos) == 0 {
		return nil
	}
	log.Info().Msgf("Notifying coaches of new gym request: %v", tos)

	email := mail.NewPersonalization()
	email.AddTos(tos...)

	// Just for debugging, all mail will be BCC'd to Jordan
	email.AddBCCs([]*mail.Email{
		mail.NewEmail("Jordan", "jordan@dionysustechnologygroup.com"),
	}...)

	payload := mail.NewV3Mail().
		AddPersonalizations(email).
		AddContent(mail.NewContent("text/html", tmplOut.String()))

	log.Info().Msgf("Email tos: %v %v", tos[0].Address, tos[1].Address)
	payload.SetFrom(mail.NewEmail("Grapple Notifications", "support@grapplemma.com"))
	payload.Subject = fmt.Sprintf("Grapple MMA: a student has requested to join %s", gym.Name)

	resp, err := s.sendGridClient.Send(payload)
	if err != nil {
		log.Warn().Msgf("Failed to send email: %v", err)
		return fmt.Errorf("failed to send email to coach: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Warn().Msgf("Mail failed to send: %+v", resp)
	}

	return nil
}

func (s *Service) updateGymRequestTX(ctx context.Context, payload *GymRequest, id string) (*GymRequest, error) {
	transactionOptions := options.Transaction().SetReadConcern(readconcern.Local()).SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (any, error) {
		var request GymRequest
		request.UpdatedAt = time.Now().Local().UTC()

		if err := mongoext.UpdateByID(ctx, s.Collection, id, payload, &request, nil); err != nil {
			return lambda_v2.ServerError(fmt.Errorf("failed to update gym record: %v", err))
		}

		// return early if the request was not approved
		if payload.Status != RequestAccepted {
			return request, nil
		}

		// send notification to the student that their request was approved
		// TODO: implement this
		if err := s.notifyStudent(ctx, &request); err != nil {
			return nil, err
		}

		// The request was approved by the coach: update the student profile's gym_associations field.
		log.Debug().Msgf("a gym request was approved by coach for student %q (%s)", request.RequestorEmail, request.RequestorID)

		// create new profile
		gymAssociation := profiles.GymAssociation{
			CoachName: "TODO",
			Email:     request.RequestorEmail,
			GymID:     request.GymID,
			Role:      profiles.StudentRole,
			EmailPreferences: &profiles.EmailPreferences{
				NotifyOnAnnouncements: true,
				NotifyOnRequests:      true, // only used if this is a coach profile
			},
		}

		// create filter & update statements, send to mongodb to update the student's profile.
		filter := bson.M{
			"cognito_id": request.RequestorID,
		}
		update := bson.M{
			"$push": bson.M{
				"gyms": gymAssociation,
			},
		}

		// Update student profile with the new gym association
		var upsertResult profiles.Profile
		coll := s.Client.Database("grapple").Collection("profiles")
		if err := mongoext.Update(ctx, coll, update, filter, &upsertResult, nil); err != nil {
			return nil, fmt.Errorf("failed to upsert student's profile with filter %v after creating a gym request: %v", filter, err)
		}

		log.Info().Msgf("Successfully added gym association to user profile: %s", request.RequestorID)
		return request, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("createProfile transaction completed successfully!")
	}

	if request, ok := result.(GymRequest); ok {
		return &request, nil
	}

	return nil, err
}

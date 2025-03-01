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

	"github.com/Grapple-2024/backend/internal/dao"
	"github.com/Grapple-2024/backend/internal/rbac"
	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/internal/service/profiles"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readconcern"
	"go.mongodb.org/mongo-driver/v2/mongo/writeconcern"
)

//go:embed templates/new_request.html
var newRequestEmailTmpl string

//go:embed templates/request_accepted.html
var requestAcceptedEmailTmpl string

// Service is the object that handles the business logic of all gymRequest related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD gymRequest objects.
type Service struct {
	*rbac.RBAC
	*mongo.Session

	*mongoext.Client
	*mongo.Collection

	sendGridClient *sendgrid.Client
}

// NewService creates a new instance of a dao.GymRequest Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, sendGridClient *sendgrid.Client, rbac *rbac.RBAC) (*Service, error) {
	c := mc.Database("grapple").Collection("gymRequests")

	// Create Mongo Session (needed for transactions)
	svc := &Service{
		RBAC:           rbac,
		Client:         mc,
		Collection:     c,
		sendGridClient: sendGridClient,
	}

	session, err := svc.StartSession()
	if err != nil {
		return nil, err
	}
	svc.Session = session
	if err := svc.ensureIndices(ctx); err != nil {
		return nil, err
	}

	return svc, nil
}

func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	showByWeek := req.QueryStringParameters["show_by_week"]
	gymID := req.QueryStringParameters["gym_id"]
	status := req.QueryStringParameters["status"]
	requestorID := req.QueryStringParameters["requestor_id"]
	membershipType := req.QueryStringParameters["membership_type"]
	search := req.QueryStringParameters["search"]
	role := req.QueryStringParameters["role"]
	sortColumn := req.QueryStringParameters["sort_column"]
	if sortColumn == "" {
		sortColumn = "first_name"
	}
	sortDirection := req.QueryStringParameters["sort_direction"]
	if sortDirection == "" {
		sortDirection = "-1"
	}
	sortDirectionInt, err := strconv.Atoi(sortDirection)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, "invalid &sort_direction query parameter: must be one of [1 or -1]")
	}

	page := req.QueryStringParameters["page"]
	if page == "" {
		page = "1"
	}
	pageSize := req.QueryStringParameters["page_size"]
	if pageSize == "" {
		pageSize = "10"
	}
	pageSizeInt, err := strconv.Atoi(pageSize)
	if err != nil && pageSize != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page_size query parameter: "+pageSize)
	}
	pageInt, err := strconv.Atoi(page)
	if err != nil && page != "" {
		return lambda.ClientError(http.StatusBadRequest, "invalid &page query parameter: "+page)
	}

	filter := bson.M{}
	if gymID != "" {
		gymObjID, err := bson.ObjectIDFromHex(gymID)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid object ID specified for gym_id query param: %s", gymID))
		}
		filter["gym_id"] = gymObjID
	}
	if membershipType != "" {
		filter["membership_type"] = membershipType
	}
	if role != "" {
		filter["role"] = role
	}

	if status != "" {
		filter["status"] = status
	}

	if showByWeek != "" {
		time, err := time.Parse(time.RFC3339, showByWeek)
		if err != nil {
			return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("invalid value for &show_by_week query param %q: must conform to RFC3339 standards: %v", showByWeek, err))
		}
		year, week := time.ISOWeek()
		filter["created_at_year"] = year
		filter["created_at_week"] = week
	}

	if requestorID != "" {
		filter["requestor_id"] = requestorID
	}

	searchFilter := []bson.M{}
	if search != "" {
		searchFilter = append(searchFilter,
			bson.M{
				"first_name": bson.M{
					"$regex":   search,
					"$options": "i",
				},
			},
			bson.M{
				"last_name": bson.M{
					"$regex":   search,
					"$options": "i",
				},
			},
			bson.M{
				"status": bson.M{
					"$regex":   search,
					"$options": "i",
				},
			},
		)
	}
	if len(searchFilter) > 0 {
		filter["$or"] = searchFilter
	}

	log.Info().Msgf("Requests Filter: %+v", filter)
	var requests []dao.GymRequest
	opts := options.Find().SetSort(bson.M{sortColumn: sortDirectionInt}) // -1 = DESCENDING (newest at the top), 1 = ASCENDING (oldest at the top)
	if err := mongoext.Paginate(ctx, s.Collection, filter, pageInt, pageSizeInt, false, opts, &requests); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find objects: %v", err))
	}
	// if no records are found, initialize empty slice so we can return [] instead of nil in JSON :)
	if requests == nil {
		requests = []dao.GymRequest{}
	}

	// join the profiles collection on each gym request
	requests = s.joinProfileOnRequests(ctx, requests)

	totalCount, err := s.Collection.CountDocuments(ctx, filter, nil)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("error counting documents: %v", err))
	}

	resp, err := service.NewGetAllResponse("gymRequests", requests, totalCount, len(requests), pageInt, pageSizeInt)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessGet handles HTTP requests for GET /gymRequests/{id}
func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	// Get the gymRequest by ID
	var gymRequest dao.GymRequest
	if err := mongoext.FindByID(ctx, s.Collection, id, &gymRequest); err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to find gymRequest by ID: %v", err))
	}

	// Return record as JSON
	json, err := json.Marshal(gymRequest)
	if err != nil {
		return lambda.ServerError(err)
	}
	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

// ProcessPost handles HTTP requests for POST /gym-requests
func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// get token and set Cognito ID and Email on the Request payload to the values tied to the token
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("authentication failure: %v", err))
	}

	var payload dao.GymRequest
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}
	payload.Status = dao.RequestPending
	payload.RequestorID = token.Sub
	// payload.RequestorEmail = token.Email
	// payload.FirstName = token.GivenName
	// payload.LastName = token.FamilyName

	if !rbac.ValidateRole(payload.Role) {
		return lambda.ClientError(http.StatusBadRequest, "invalid role name, valid values: [coach, owner, student]")
	}

	// Validate the payload struct
	validate, err := service.NewValidator()
	if err != nil {
		return lambda.ServerError(err)
	}
	if err := validate.Struct(payload); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusBadRequest, errMsgs...)
	}
	if err := validateMembershipType(payload.MembershipType); err != nil {
		return lambda.ClientError(http.StatusBadRequest, err.Error())
	}

	gymsColl := s.Database().Collection("gyms")
	var gym *dao.Gym
	if err := mongoext.FindByID(ctx, gymsColl, payload.GymID.Hex(), &gym); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("could not find gym with id %q: %v", payload.GymID, err))
	}

	// insert the dao.GymRequest, store the resulting record in 'result' variable
	payload.CreatedAt = time.Now().Local().UTC()
	payload.UpdatedAt = payload.CreatedAt

	var result dao.GymRequest
	if err := mongoext.Insert(ctx, s.Collection, &payload, &result); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert gym request ooc: %v", err))
	}

	requests := s.joinProfileOnRequests(ctx, []dao.GymRequest{result})
	if len(requests) == 0 {
		return lambda.ServerError(fmt.Errorf("data inconsistency on joining profile with request: %v", requests))
	}
	request := requests[0]

	// notify the coach by email that a new request was submitted for their gym
	if err := s.notifyCoachesOnNewRequest(ctx, &request); err != nil {
		log.Warn().Msgf("Failed to notify coach of new gym request - FAILING SILENTLY: %v", err)
	}

	resp, err := json.Marshal(request)
	if err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusCreated, string(resp), nil), nil
}

// ProcessPut handles HTTP requests for PUT /gym-requests/{id}
func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	token, err := service.GetToken(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusUnauthorized, fmt.Sprintf("permission denied: %v", err))
	}

	var payload dao.GymRequest
	if err := json.Unmarshal([]byte(req.Body), &payload); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}
	if !dao.IsValidStatus(payload.Status) {
		return lambda.ClientError(http.StatusBadRequest, "invalid value for status field, must be one of [Pending, Accepted, Denied]")
	}

	id := req.PathParameters["id"]
	var request dao.GymRequest
	if err := mongoext.FindByID(ctx, s.Collection, id, &request); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to find gym request with ID %s: %v", id, err))
	}

	resourceID := fmt.Sprintf("%s:%s:%s", rbac.ResourceGym, request.GymID.Hex(), rbac.ResourceGymRequests) // gym:<gym_id>:requests
	isAuthorized, err := s.IsAuthorized(ctx, token.Username, resourceID, rbac.ActionUpdate)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	} else if !isAuthorized {
		return lambda.ClientError(http.StatusForbidden,
			fmt.Sprintf("permission denied: user is not authorized for action '%s' on '%s'", rbac.ActionUpdate, resourceID),
		)
	}

	// update the record in mongo
	result, err := s.updateGymRequestTX(ctx, &payload, id)
	if err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("failed to finish updateGymRequest transaction: %v", err))
	}

	// Marshal result to JSON and return it in the response
	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

// ProcessDelete handles HTTP requests for DELETE /gymRequests/{id}
func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	id := req.PathParameters["id"]
	if err := mongoext.DeleteOne(ctx, s.Collection, id); err != nil {
		return lambda.NewResponse(http.StatusBadRequest, fmt.Sprintf("failed to delete record: %v", err), nil), nil
	}

	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) getProfileByCognitoID(ctx context.Context, cognitoSubID string) (*dao.Profile, error) {
	c := s.Client.Database("grapple").Collection("profiles")
	filter := bson.M{"cognito_id": cognitoSubID}
	var profile dao.Profile
	if err := mongoext.FindOne(ctx, c, filter, &profile); err != nil {
		return nil, fmt.Errorf("could not find any profile with cognito id %q: %v", cognitoSubID, err)
	}

	return &profile, nil
}

func (s *Service) notifyUserOnRequestAccepted(ctx context.Context, request *dao.GymRequest) error {
	profile, err := s.getProfileByCognitoID(ctx, request.RequestorID)
	if err != nil {
		return err
	}

	if !profile.NotifyOnRequestAccepted {
		log.Warn().Msgf("User has disabled notifications for new requests, nothing will be sent")
		return nil
	}

	tmpl, err := template.New("").Parse(requestAcceptedEmailTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	gymsColl := s.Database().Collection("gyms")
	var gym *dao.Gym
	if err := mongoext.FindByID(ctx, gymsColl, request.GymID.Hex(), &gym); err != nil {
		return fmt.Errorf("could not find gym with ID %v: %v", request.GymID, err)
	}

	var tmplOut bytes.Buffer
	tmplData := struct {
		GymName string
		Role    string
	}{
		GymName: gym.Name,
		Role:    request.Role,
	}
	if err := tmpl.Execute(&tmplOut, tmplData); err != nil {
		return fmt.Errorf("error executing template with data %v:\n %v", tmplData, err)
	}

	subject := fmt.Sprintf("Grapple MMA: your request to join %s was accepted!", gym.Name)
	from := mail.NewEmail("Grapple Notifications", "support@grapplemma.com")
	to := mail.NewEmail("Grapple User", request.RequestorEmail)
	message := mail.NewSingleEmail(from, subject, to, "", tmplOut.String())

	_, err = s.sendGridClient.Send(message)
	if err != nil {
		return fmt.Errorf("failed to send email to user: %v", err)
	}

	return nil
}

func (s *Service) notifyCoachesOnNewRequest(ctx context.Context, request *dao.GymRequest) error {
	tmpl, err := template.New("").Parse(newRequestEmailTmpl)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	gymID := request.GymID.Hex()
	gymsColl := s.Database().Collection("gyms")
	var gym *dao.Gym
	if err := mongoext.FindByID(ctx, gymsColl, gymID, &gym); err != nil {
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

	// get all coaches and owners for this gym
	coachesGroup := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, gymID, rbac.Coaches)
	coaches, err := s.RBAC.ListUsersInGroup(ctx, coachesGroup)
	if err != nil {
		return err
	}

	ownersGroup := fmt.Sprintf("%s::%s::%s", rbac.ResourceGym, gymID, rbac.Owners)
	owners, err := s.RBAC.ListUsersInGroup(ctx, ownersGroup)
	if err != nil {
		return err
	}

	allCoaches := append(owners, coaches...)

	var tos []*mail.Email
	for _, u := range allCoaches {
		tos = append(tos, mail.NewEmail("Grapple Coach", *u.Username))
	}
	if len(tos) == 0 {
		log.Info().Msgf("No coaches or owners found for gym %v", gymID)
		return nil
	}
	log.Info().Msgf("Notifying coaches of new gym request: %v", tos)

	email := mail.NewPersonalization()
	email.AddTos(tos...)

	// Just for debugging, all mail will be BCC'd to Jordan
	email.AddBCCs([]*mail.Email{
		mail.NewEmail("Jordan", "jordan@dionysustechnologygroup.com"),
		mail.NewEmail("Stephen", "stephen@dionysustechnologygroup.com"),
	}...)

	payload := mail.NewV3Mail().
		AddPersonalizations(email).
		AddContent(mail.NewContent("text/html", tmplOut.String()))

	payload.SetFrom(mail.NewEmail("Grapple Notifications", "support@grapplemma.com"))
	payload.Subject = fmt.Sprintf("Grapple MMA: a student has requested to join %s", gym.Name)

	resp, err := s.sendGridClient.Send(payload)
	if err != nil {
		log.Warn().Msgf("failed to send email: %v", err)
		return fmt.Errorf("failed to send email to coach: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Warn().Msgf("mail failed to send: %+v", resp.StatusCode)
	}

	return nil
}

func (s *Service) updateGymRequestTX(ctx context.Context, payload *dao.GymRequest, requestID string) (*dao.GymRequest, error) {
	transactionOptions := options.Transaction().
		SetReadConcern(&readconcern.ReadConcern{Level: "local"}).
		SetWriteConcern(&writeconcern.WriteConcern{W: 1})

	result, err := s.WithTransaction(ctx, func(sessCtx context.Context) (any, error) {
		var request dao.GymRequest
		request.UpdatedAt = time.Now().Local().UTC()
		if err := mongoext.UpdateByID(ctx, s.Collection, requestID, payload, &request, nil); err != nil {
			return nil, fmt.Errorf("failed to update gym record: %v", err)
		}

		gymID := request.GymID.Hex()

		// Handle request denied
		if payload.Status == dao.RequestDenied {
			if err := profiles.DeleteGymAssociation(ctx, s.Client, request.GymID, request.RequestorID); err != nil {
				return nil, err
			}
		}

		if payload.Status != dao.RequestAccepted {
			return request, nil
		}

		// Handle request approved
		log.Info().Msgf("a gym request was approved by coach for student %q (%s)", request.RequestorEmail, request.RequestorID)

		// fetch the gym associated with this gym request, make sure it exists.
		var gym dao.Gym
		gymsColl := s.Database().Collection("gyms")
		if err := mongoext.FindByID(ctx, gymsColl, gymID, &gym); err != nil {
			return nil, fmt.Errorf("failed to find gym with id %v: %v", payload.GymID.Hex(), err)
		}

		// Assign user to the proper cognito group and update/insert their gym association as a student.
		if err := s.RBAC.AssignUserToGymRole(ctx, gymID, request.RequestorEmail, request.Role); err != nil {
			return nil, fmt.Errorf("could not assign user to %s group of gym %s: %v", request.Role, gymID, err)
		}
		if err := profiles.UpsertGymAssociation(ctx, s.Client, &gym, request.Role, &request); err != nil {
			return nil, fmt.Errorf("could not upsert gym association: %v", err)
		}

		// Notify user that their request was accepted
		if err := s.notifyUserOnRequestAccepted(ctx, &request); err != nil {
			return nil, err
		}

		return request, nil
	}, transactionOptions)

	if err != nil {
		log.Warn().Err(err).Msgf("failed to run mongo transaction for profile creation")
		return nil, err
	} else {
		log.Info().Msgf("updateGymRequest transaction completed successfully!")
	}

	if request, ok := result.(dao.GymRequest); ok {
		return &request, nil
	}

	return nil, err
}

func Find(ctx context.Context, collection *mongo.Collection, filter bson.M, result *[]dao.GymRequest) error {
	cursor, err := collection.Find(ctx, filter, nil)
	if err != nil {
		return err
	}

	for cursor.Next(ctx) {
		var request dao.GymRequest
		if err := cursor.Decode(&request); err != nil {
			return err
		}
		log.Info().Msgf("Cursor decoded %v", request)

		*result = append(*result, request)
	}

	if err := cursor.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Service) ensureIndices(ctx context.Context) error {
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{"gym_id", 1}, {"requestor_id", -1}},
		Options: options.Index().SetUnique(true),
	}

	_, err := s.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return err
	}

	return nil
}

func validateMembershipType(membershipType string) error {
	switch membershipType {
	case dao.VirtualMembership:
		return nil
	case dao.InPersonMembership:
		return nil

	default:
		return fmt.Errorf("invalid membership type: %v, must be one of [%s, %s]",
			membershipType,
			dao.VirtualMembership,
			dao.InPersonMembership,
		)
	}
}

// joinProfileOnRequests joins the profile record onto each gym request in the requests slice.
func (s *Service) joinProfileOnRequests(ctx context.Context, requests []dao.GymRequest) []dao.GymRequest {
	result := make([]dao.GymRequest, len(requests))

	for i, req := range requests {
		profile, err := s.getProfileByCognitoID(ctx, req.RequestorID)
		if err != nil {
			log.Warn().Msgf("Could not find profile by gym request! Highly likely this is a data consistency issue: %v", err)
			continue
		}

		joined := req
		joined.Profile = profile
		result[i] = joined
	}

	return result
}

// UpsertGymRequest upserts (inserts or updates) a gym request.
func UpsertGymRequest(ctx context.Context, mc *mongoext.Client, payload *dao.GymRequest) (*dao.GymRequest, error) {
	collection := mc.Database("grapple").Collection("gymRequests")
	filter := bson.M{
		"requestor_email": payload.RequestorEmail,
		"gym_id":          payload.GymID,
	}

	updateQuery := bson.M{
		"$set": payload,
	}
	upsert := true // Example option: enable upsert
	opts := options.UpdateOne().SetUpsert(upsert)

	var result dao.GymRequest
	if err := mongoext.UpdateOne(ctx, collection, updateQuery, filter, &result, opts); err != nil {
		return nil, fmt.Errorf("failed to upsert profile with filter %v: %v", filter, err)
	}

	return &result, nil
}

func DeleteGymRequestsByGymID(ctx context.Context, requestsCollection *mongo.Collection, gymId string) error {
	filter := bson.M{"gym_id": gymId}
	_, err := requestsCollection.DeleteMany(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to delete gym requests for gym %v: %v", gymId, err)
	}

	return nil
}

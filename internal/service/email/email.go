package email

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"
	"time"

	"github.com/Grapple-2024/backend/internal/service"
	"github.com/Grapple-2024/backend/pkg/lambda"
	mongoext "github.com/Grapple-2024/backend/pkg/mongo"
	"github.com/aws/aws-lambda-go/events"
	"github.com/go-playground/validator/v10"
	"github.com/rs/zerolog/log"
	"github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

//go:embed templates/demo_request.html
var newDemoEmailTemplate string

// Service is the object that handles the business logic of all announcement related operations.
// Service talks to the underlying Mongo Client (Data access layer) to CRUD announcement objects.
type Service struct {
	*mongoext.Client
	*mongo.Collection

	sendGridClient *sendgrid.Client
}

// NewService creates a new instance of a Announcement Service given a mongo client
func NewService(ctx context.Context, mc *mongoext.Client, sendGridClient *sendgrid.Client) (*Service, error) {
	c := mc.Database("grapple").Collection("emailList")

	// Create unique index for announcement names
	svc := &Service{
		Client:         mc,
		Collection:     c,
		sendGridClient: sendGridClient,
	}

	return svc, nil
}

func (s *Service) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if req.Path == "/emails/demo" {
		return s.ProcessDemoEmail(ctx, req)
	}

	var email Email
	if err := json.Unmarshal([]byte(req.Body), &email); err != nil {
		return lambda.ClientError(http.StatusUnprocessableEntity, fmt.Sprintf("invalid request body: %v", err))
	}

	// Validate email format
	if !isValidEmail(email.Email) {
		return lambda.ServerError(fmt.Errorf("invalid email format"))
	}

	validate, err := service.NewValidator()

	if err != nil {
		return lambda.ServerError(err)
	}

	if err := validate.Struct(email); err != nil {
		var errMsgs []string
		for _, err := range err.(validator.ValidationErrors) {
			errMsgs = append(errMsgs, fmt.Sprintf("Field '%s' failed validation with tag '%s'", err.Field(), err.Tag()))
		}
		return lambda.ClientError(http.StatusUnprocessableEntity, errMsgs...)
	}
	email.CreatedAt = time.Now().Local().UTC()
	email.UpdatedAt = email.CreatedAt

	// insert the announcement, store the resulting record in 'result' variable
	var result Email
	if err := mongoext.Insert(ctx, s.Collection, &email, &result); err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to insert record: %v", err))
	}

	resp, err := json.Marshal(result)
	if err != nil {
		return lambda.ServerError(fmt.Errorf("failed to marshal response: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(resp), nil), nil
}

func (s *Service) ProcessDemoEmail(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Send a demo email
	if err := s.sendDemoEmail(ctx, req); err != nil {
		return lambda.ServerError(fmt.Errorf("failed to send demo email: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, "Email Sent", nil), nil
}

func (s *Service) sendDemoEmail(ctx context.Context, req events.APIGatewayProxyRequest) error {
	// Parse the request body
	var requestData struct {
		GymName     string `json:"gym_name"`
		PhoneNumber string `json:"phone_number"`
		Email       string `json:"email"`
		Name        string `json:"name"`
		Message     string `json:"message"`
	}

	if err := json.Unmarshal([]byte(req.Body), &requestData); err != nil {
		return fmt.Errorf("failed to parse request body: %v", err)
	}

	tmpl, err := template.New("demo_email").Parse(newDemoEmailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse template: %v", err)
	}

	var tmplOut bytes.Buffer
	tmplData := struct {
		GymName     string
		PhoneNumber string
		Email       string
		Name        string
		Message     string
	}{
		GymName:     requestData.GymName,
		PhoneNumber: requestData.PhoneNumber,
		Email:       requestData.Email,
		Name:        requestData.Name,
		Message:     requestData.Message,
	}

	// Validate email format
	if !isValidEmail(requestData.Email) {
		return fmt.Errorf("invalid email format: %s", requestData.Email)
	}

	// Validate phone number format
	if !isValidPhoneNumber(requestData.PhoneNumber) {
		return fmt.Errorf("invalid phone number format: %s", requestData.PhoneNumber)
	}

	if err := tmpl.Execute(&tmplOut, tmplData); err != nil {
		return fmt.Errorf("error executing template with data %v:\n %v", tmplData, err)
	}

	// Create email to Alec
	email := mail.NewPersonalization()
	email.AddTos(mail.NewEmail("Alec", "Alec@grapplemma.com"))

	// BCC Jordan for debugging
	email.AddBCCs([]*mail.Email{
		mail.NewEmail("Jordan", "jordan@dionysustechnologygroup.com"),
	}...)

	payload := mail.NewV3Mail().
		AddPersonalizations(email).
		AddContent(mail.NewContent("text/html", tmplOut.String())).
		SetFrom(mail.NewEmail("Grapple Notifications", "support@grapplemma.com"))

	payload.Subject = fmt.Sprintf("Demo Request from %s", requestData.GymName)

	resp, err := s.sendGridClient.Send(payload)
	if err != nil {
		log.Warn().Msgf("failed to send email: %v", err)
		return fmt.Errorf("failed to send demo request email: %v", err)
	}
	if resp.StatusCode != http.StatusAccepted {
		log.Warn().Msgf("failed to send email: %+v", resp)
		return fmt.Errorf("unexpected status code from email service: %d", resp.StatusCode)
	}

	return nil
}

func (s *Service) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, ``, nil), nil
}

func (s *Service) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, id string) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

func (s *Service) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(``), nil), nil
}

// isValidEmail validates email format
func isValidEmail(email string) bool {
	// Basic email validation
	if email == "" {
		return false
	}

	// Check for @ symbol
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	// Check local part and domain
	localPart, domain := parts[0], parts[1]
	if localPart == "" || domain == "" {
		return false
	}

	// Check domain has at least one dot
	domainParts := strings.Split(domain, ".")
	if len(domainParts) < 2 {
		return false
	}

	// Check TLD is not empty
	if domainParts[len(domainParts)-1] == "" {
		return false
	}

	return true
}

// isValidPhoneNumber validates phone number format
func isValidPhoneNumber(phone string) bool {
	// Remove all non-digit characters
	digitsOnly := strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, phone)

	// Check if we have a reasonable number of digits (10-15 for international numbers)
	if len(digitsOnly) < 10 || len(digitsOnly) > 15 {
		return false
	}

	// For US numbers, usually 10 digits or 11 digits with country code "1"
	if len(digitsOnly) == 10 || (len(digitsOnly) == 11 && digitsOnly[0] == '1') {
		return true
	}

	// For international numbers, let's just make sure we have enough digits
	return len(digitsOnly) >= 10
}

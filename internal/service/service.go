package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/MicahParks/keyfunc/v3"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/go-http-utils/headers"
	"github.com/go-playground/validator/v10"
	"github.com/golang-jwt/jwt/v5"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
)

// API_URL stores the base api url for generating next and previous page URLs.
var API_URL string

// Contains helpers for all the services
type GetAllResponse struct {

	// The main data of the response
	Data any `json:"data"`

	// Number of records in the response
	Count int `json:"count"`

	// Total number of records in this collection
	TotalCount int64 `json:"total_count"`

	// URL to the next page
	NextPage *string `json:"next_page"`

	// URL to the previous page
	PreviousPage *string `json:"previous_page"`
}

func init() {
	API_URL = os.Getenv("API_URL")
}

// NewGetAllResponse creates a new "GET All" response and returns it as JSON bytes.
func NewGetAllResponse(subpath string, data any, totalCount int64, count, currPage, pageSize int) ([]byte, error) {
	n := float64(totalCount) / float64(pageSize)
	totalPages := int64(math.Ceil(n))
	resp := &GetAllResponse{
		Data:       data,
		Count:      count,
		TotalCount: totalCount,
	}

	// if we're not on the last page, add the next page's URL to the response.
	if totalPages > int64(currPage) {
		nextPageURL := fmt.Sprintf("%s/%s/?pageSize=%d&page=%d", API_URL, subpath, pageSize, currPage+1)
		resp.NextPage = &nextPageURL
	}

	// if we're not on the first page, add the previous page's URL to the response.
	if currPage > 1 && totalPages >= int64(currPage) {
		prevPageURL := fmt.Sprintf("%s/%s/?pageSize=%d&page=%d", API_URL, subpath, pageSize, currPage-1)
		resp.PreviousPage = &prevPageURL
	}

	return json.Marshal(resp)
}

func IsAlphaNumericAndSpaces(fl validator.FieldLevel) bool {
	pattern := "^[A-Za-z0-9 ]*$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to compile validation regexp: %v", pattern)
		return false
	}

	// Check if the string matches the pattern
	return re.MatchString(fl.Field().String())
}

func IsState(fl validator.FieldLevel) bool {
	pattern := "^(Alabama|Alaska|Arizona|Arkansas|California|Colorado|Connecticut|Delaware|Florida|Georgia|Hawaii|Idaho|Illinois|Indiana|Iowa|Kansas|Kentucky|Louisiana|Maine|Maryland|Massachusetts|Michigan|Minnesota|Mississippi|Missouri|Montana|Nebraska|Nevada|New Hampshire|New Jersey|New Mexico|New York|North Carolina|North Dakota|Ohio|Oklahoma|Oregon|Pennsylvania|Rhode Island|South Carolina|South Dakota|Tennessee|Texas|Utah|Vermont|Virginia|Washington|West Virginia|Wisconsin|Wyoming)$"
	re, err := regexp.Compile(pattern)
	if err != nil {
		log.Warn().Err(err).Msgf("failed to compile validation regexp %v", pattern)
		return false
	}

	// Check if the string matches the pattern
	return re.MatchString(fl.Field().String())
}

// Token represents the AWS Cognito user token
type Token struct {
	Username   string   `mapstructure:"cognito:username"`
	Email      string   `mapstructure:"email"`
	Roles      []string `mapstructure:"cognito:roles"`
	Groups     []string `mapstructure:"cognito:groups"`
	GivenName  string   `mapstructure:"given_name"`
	FamilyName string   `mapstructure:"family_name"`

	Sub string `mapstructure:"sub"`
}

func GetToken(hdrs map[string]string) (*Token, error) {
	authHeader := hdrs[headers.Authorization]
	if len(authHeader) <= 1 {
		return nil, fmt.Errorf("auth header not valid: %v", authHeader)
	}
	bearer := strings.Split(authHeader, "Bearer")
	var err error
	if len(bearer) <= 1 {
		return nil, fmt.Errorf("auth header not valid: %v", authHeader)
	}

	tokenString := strings.TrimSpace(bearer[1])

	regionID := "us-west-1"
	userPoolID := "us-west-1_HT5oR6AwO"
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}

	// Parse the JWT.
	token, err := jwt.Parse(tokenString, jwks.Keyfunc)
	if err != nil {
		return nil, err
	}

	// Check if the token is valid.
	if !token.Valid {
		return nil, err
	}

	var t *Token
	if err := mapstructure.Decode(token.Claims.(jwt.MapClaims), &t); err != nil {
		return nil, err
	}
	// log.Info().Msgf("Token: %+v", t)

	return t, nil
}

// generatePresignedURL generates a presigned URL given a urlType of either "upload" or "download".
// if urlType is 'upload': the presigned URL will be for an upload PUT request to the bucket.
// if urlType is 'download': the presigned URL will be for a download (GET) request to the bucket.
func GeneratePresignedURL(ctx context.Context, psc *s3.PresignClient, bucketName, operation, key string) (*v4.PresignedHTTPRequest, error) {
	opts := func(opts *s3.PresignOptions) {
		opts.Expires = time.Minute * 30
	}

	switch operation {
	case "upload":
		params := &s3.PutObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		}
		r, err := psc.PresignPutObject(ctx, params, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get presigned upload url for object %q in bucket %q: %v", key, bucketName, err)
		}

		return r, nil
	case "download":
		params := &s3.GetObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(key),
		}

		r, err := psc.PresignGetObject(ctx, params, opts)
		if err != nil {
			return nil, fmt.Errorf("failed to get presigned upload url for object %q in bucket %q: %v", key, bucketName, err)
		}

		return r, nil
	default:
		return nil, fmt.Errorf("urlType must be either 'upload' or 'download!'")
	}
}

func NewValidator() (*validator.Validate, error) {
	validator := validator.New()
	validator.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return validator, nil
}

package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/rs/zerolog/log"

	dynamodbsdk "github.com/Grapple-2024/backend/dynamodb"
	"github.com/Grapple-2024/backend/lambda"
	"github.com/aws/aws-lambda-go/events"
)

type UserAsset struct {
	UserID    string `json:"user_id" dynamodbav:"user_id"`
	URL       string `json:"url" dynamodbav:"url"`
	AssetName string `json:"asset_name" dynamodbav:"asset_name"`
}

type UserAssetHandler struct {
	*AuthService
	*s3.PresignClient
	DynamoClient         *dynamodbsdk.Client
	S3Client             *s3.Client
	userAssetsTableName  string
	userAssetsBucketName string
}

func NewUserAssetHandler(ctx context.Context, dynamoEndpoint, region string) (*UserAssetHandler, error) {
	// Using the SDK's default configuration, loading additional config
	// and credentials values from the environment variables, shared
	// credentials, and shared configuration files
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	c := s3.NewFromConfig(cfg)
	psc := s3.NewPresignClient(c)

	authSVC, err := NewAuthService(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	db, err := dynamodbsdk.NewClient(dynamoEndpoint)
	if err != nil {
		return nil, err
	}

	return &UserAssetHandler{
		DynamoClient:         db,
		S3Client:             c,
		PresignClient:        psc,
		AuthService:          authSVC,
		userAssetsTableName:  os.Getenv("PUBLIC_USER_ASSETS_TABLE_NAME"),
		userAssetsBucketName: os.Getenv("PUBLIC_USER_ASSETS_BUCKET_NAME"),
	}, nil
}

// Needed to satisfy interface, but not implemented
func (h *UserAssetHandler) ProcessGetByID(ctx context.Context, req events.APIGatewayProxyRequest, userID string) (events.APIGatewayProxyResponse, error) {
	assetName := req.QueryStringParameters["asset_name"]
	if assetName == "" {
		return lambda.ClientError(http.StatusBadRequest, "must specify asset name to fetch with ?asset_name=<avatar/other>")
	}

	log.Info().Msgf("Fetching user assets for user ID %q", userID)

	// build filter and key expressions
	builder := expression.NewBuilder().WithKeyCondition(
		expression.Key("user_id").Equal(expression.Value(userID)),
	)
	filterExpr := dynamodbsdk.BuildExpression(map[string]dynamodbsdk.Condition{
		"asset_name": {
			Operator: "Equal",
			Value:    assetName,
		},
	})
	builder = builder.WithFilter(*filterExpr)

	e, err := builder.Build()
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to build expresision: %v", err))
	}

	params := &dynamodb.QueryInput{
		TableName:                 &h.userAssetsTableName,
		IndexName:                 aws.String("UserIndex"),
		KeyConditionExpression:    e.KeyCondition(),
		FilterExpression:          e.Filter(),
		ExpressionAttributeNames:  e.Names(),
		ExpressionAttributeValues: e.Values(),
	}

	o, err := h.Query(ctx, params)
	if err != nil {
		return lambda.ClientError(http.StatusBadRequest, fmt.Sprintf("failed to query dynamodb: %v", err))
	}

	uas := []UserAsset{}
	if err := attributevalue.UnmarshalListOfMaps(o.Items, &uas); err != nil {
		return lambda.ServerError(err)
	}

	json, err := json.Marshal(&uas)
	if err := attributevalue.UnmarshalListOfMaps(o.Items, &uas); err != nil {
		return lambda.ServerError(err)
	}

	return lambda.NewResponse(http.StatusOK, string(json), nil), nil
}

func (h *UserAssetHandler) ProcessPost(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	key := req.QueryStringParameters["key"]
	if key == "" {
		return lambda.ClientError(http.StatusForbidden, "must specify file name with ?key=<file-name>")
	}
	assetName := req.QueryStringParameters["asset_name"]
	if assetName == "" {
		return lambda.ClientError(http.StatusForbidden, "must specify asset name with ?asset_name=<avatar/other>")
	}

	token, err := token(req.Headers)
	if err != nil {
		return lambda.ClientError(http.StatusForbidden, fmt.Sprintf("permission denied: %v", err))
	}

	// create the presigned upload URL
	objectKey := fmt.Sprintf("%s/%s", token.Email, key)
	_, err = h.uploadObject(ctx, h.userAssetsBucketName, objectKey, strings.NewReader(req.Body))
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error uploading object to s3: %v", err))
	}

	url := fmt.Sprintf("https://%s.s3.us-west-1.amazonaws.com/%s", h.userAssetsBucketName, objectKey)
	ua := &UserAsset{
		UserID:    token.Sub,
		URL:       url,
		AssetName: assetName,
	}

	_, err = h.DynamoClient.Insert(ctx, h.userAssetsTableName, ua, "user_id")
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("failed to insert user asset into dynamo: %v", err))
	}

	bytes, err := json.Marshal(ua)
	if err != nil {
		return lambda.ClientError(http.StatusNotFound, fmt.Sprintf("error marshaling presigned url response to json: %v", err))
	}

	return lambda.NewResponse(http.StatusOK, string(bytes), nil), nil
}

func (h *UserAssetHandler) uploadObject(ctx context.Context, bucketName string, objectKey string, body io.Reader) (*s3.PutObjectOutput, error) {
	params := &s3.PutObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(objectKey),
		Body:   body,
	}

	request, err := h.S3Client.PutObject(ctx, params, func(opts *s3.Options) {
		opts.Region = "us-west-1"
	})
	if err != nil {
		log.Info().Err(err).Msgf("Couldn't upload object to s3 in bucket %q, key: %q", bucketName, objectKey)
		return nil, err
	}

	return request, nil
}

func (h *UserAssetHandler) ProcessPut(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

func (h *UserAssetHandler) ProcessDelete(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

func (h *UserAssetHandler) ProcessGetAll(ctx context.Context, req events.APIGatewayProxyRequest, limit int32, startKey map[string]types.AttributeValue) (events.APIGatewayProxyResponse, error) {
	return lambda.NewResponse(http.StatusOK, string(""), nil), nil
}

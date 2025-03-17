package s3

import (
	"context"
	"strconv"

	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog/log"
)

const maxPartSize int64 = 50 * 1024 * 1024 // 50MB
type PresignedRequest struct {
	BucketName string `json:"bucket_name"`
	UploadPath string `json:"upload_path"`
	PartNumber string `json:"part_number"`
	UploadID   string `json:"upload_id"`
}

type CompleteUploadRequest struct {
	BucketName string `json:"bucket_name"`
	UploadID   string `json:"upload_id"`
	UploadPath string `json:"upload_path"`
}

type PresignedResponse struct {
	URL string `json:"url"`
}

type Client struct {
	*s3.PresignClient
	*s3.Client
}

func New(region string) (*Client, error) {
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}
	s3Client := s3.NewFromConfig(cfg)
	presignClient := s3.NewPresignClient(s3Client)

	return &Client{PresignClient: presignClient, Client: s3Client}, nil
}

// Part 1: Frontend requests multipart upload and receives the upload ID.
func (c *Client) StartMultipartUpload(ctx context.Context, bucketName, uploadPath, contentType string) (*s3.CreateMultipartUploadOutput, error) {

	input := &s3.CreateMultipartUploadInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(uploadPath),
		ContentType: aws.String(contentType),
	}

	log.Info().Msgf("starting multipart upload with key %s and content type %s in bucket %s", uploadPath, contentType, bucketName)

	return c.CreateMultipartUpload(context.TODO(), input)
}

// Frontend splits input file into chunks and requests a presigned upload url for each chunk.
// Frontend must keep track of the upload ID created in part 1.
func (c *Client) GeneratePresignedPartURL(ctx context.Context, req *PresignedRequest) (*v4.PresignedHTTPRequest, error) {
	partNumber, err := strconv.Atoi(req.PartNumber)
	if err != nil {
		return nil, err
	}
	input := &s3.UploadPartInput{
		Bucket:     aws.String(req.BucketName),
		Key:        aws.String(req.UploadPath),
		UploadId:   aws.String(req.UploadID),
		PartNumber: aws.Int32(int32(partNumber)),
	}
	log.Info().
		Str("bucketName", req.BucketName).
		Str("uploadPath", req.UploadPath).
		Str("Upload ID", req.UploadID).
		Str("part num", req.PartNumber).
		Msgf("Generating presigned url")

	return c.PresignUploadPart(ctx, input)
}

// Part 3: after all chunks are uploaded by frontend, frontend must call completeUpload, passing the upload ID.
func (c *Client) CompleteUpload(ctx context.Context, req *CompleteUploadRequest) (*s3.CompleteMultipartUploadOutput, error) {

	log.Info().Msgf("Trying to list s3 upload parts for bucket %s, key %s, and upload id %s", req.BucketName, req.UploadPath, req.UploadID)
	listPartsOutput, err := c.ListParts(ctx, &s3.ListPartsInput{
		Bucket:   aws.String(req.BucketName),
		Key:      aws.String(req.UploadPath),
		UploadId: aws.String(req.UploadID),
	})
	if err != nil {
		return nil, err
	}

	// Prepare the completed parts list
	var completedParts []types.CompletedPart
	for _, part := range listPartsOutput.Parts {
		completedParts = append(completedParts, types.CompletedPart{
			ETag:       part.ETag,
			PartNumber: part.PartNumber,
		})
	}

	// Complete the multipart upload
	return c.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   aws.String(req.BucketName),
		Key:      aws.String(req.UploadPath),
		UploadId: aws.String(req.UploadID),
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completedParts,
		},
	})
}

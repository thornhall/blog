package backup

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type SpaceClient struct {
	Client *s3.Client
	Bucket string
}

func NewSpaceClient() (*SpaceClient, error) {
	if os.Getenv("ENV") != "prod" {
		return nil, fmt.Errorf("environment is not prod - not configuring backups")
	}
	key := os.Getenv("SPACES_KEY")
	secret := os.Getenv("SPACES_SECRET")
	endpoint := os.Getenv("SPACES_ENDPOINT")
	region := os.Getenv("SPACES_REGION")
	bucket := os.Getenv("SPACES_BUCKET")

	creds := credentials.NewStaticCredentialsProvider(key, secret, "")

	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(creds),
	)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = aws.String(endpoint)
	})

	return &SpaceClient{
		Client: client,
		Bucket: bucket,
	}, nil
}

func (s *SpaceClient) UploadFile(ctx context.Context, objectKey string, fileReader io.Reader) error {
	_, err := s.Client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.Bucket),
		Key:    aws.String(objectKey),
		Body:   fileReader,
		ACL:    "private",
	})
	if err != nil {
		return fmt.Errorf("failed to upload file: %w", err)
	}
	return nil
}

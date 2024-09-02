package storage

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"enssat.tv/autovodsaver/constants"
	"enssat.tv/autovodsaver/twitch"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/rs/zerolog"
)

type S3Storage struct {
	Context context.Context
	Client  *s3.Client
	Bucket  string
}

type Credentials struct {
	AccessKey string
	SecretKey string
	Session   string
}

func NewS3Storage(endpoint string, region string, bucket string, creds Credentials) (*S3Storage, error) {
	return NewS3StorageWithContext(context.Background(), endpoint, region, bucket, creds)
}

func NewS3StorageWithContext(ctx context.Context, endpoint string, region string, bucket string, creds Credentials) (*S3Storage, error) {
	logger := ctx.Value(constants.LoggerKey).(*zerolog.Logger)

	client := s3.New(s3.Options{
		BaseEndpoint: aws.String(endpoint),
		Region:       region,
		Credentials:  credentials.NewStaticCredentialsProvider(creds.AccessKey, creds.SecretKey, creds.Session),
	})

	// Check if bucket exists
	buckets, listBucketError := client.ListBuckets(ctx, nil)
	if listBucketError != nil {
		return nil, listBucketError
	}

	found := false
	for _, v := range buckets.Buckets {
		if *v.Name == bucket {
			found = true
			logger.Debug().Msgf("bucket %s found", bucket)
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("bucket %s has not been found", bucket)
	}

	logger.Debug().Msgf("New S3 storage instance with bucket %s (endpoint: %s)", bucket, endpoint)
	return &S3Storage{
		Context: ctx,
		Client:  client,
		Bucket:  bucket,
	}, nil
}

func (s *S3Storage) GetVideos() ([]twitch.Video, error) {
	logger := s.Context.Value(constants.LoggerKey).(*zerolog.Logger)

	if s.Client == nil {
		return []twitch.Video{}, fmt.Errorf("s3 client is nil, is the client initialize correctly ?")
	}

	objects, listObjectsError := s.Client.ListObjectsV2(s.Context, &s3.ListObjectsV2Input{
		Bucket: &s.Bucket,
	})
	if listObjectsError != nil {
		return []twitch.Video{}, listObjectsError
	}
	logger.Debug().Msgf("number of document found in bucket %s: %d", s.Bucket, *objects.KeyCount)

	for _, object := range objects.Contents {
		obj, getObjectError := s.Client.GetObjectAttributes(s.Context, &s3.GetObjectAttributesInput{
			Bucket: &s.Bucket,
			Key:    object.Key,
			ObjectAttributes: []types.ObjectAttributes{
				types.ObjectAttributesObjectParts,
			},
		})
		if getObjectError != nil {
			logger.Error().Msg(getObjectError.Error())
			continue
		}
		logger.Debug().Msgf("%s", obj.ResultMetadata)
	}

	return make([]twitch.Video, 0), nil
}

func (s *S3Storage) Save(video *twitch.Video, filePath string) error {
	logger := s.Context.Value(constants.LoggerKey).(*zerolog.Logger)

	if s.Client == nil {
		return fmt.Errorf("s3 client is nil, is the client initialize correctly ?")
	}

	file, fileOpenError := os.OpenFile(filePath, os.O_RDONLY, 0660)
	if fileOpenError != nil {
		return fileOpenError
	}

	logger.Info().Msgf("video %s being stored in s3 bucket %s", video.Title, s.Bucket)

	_, putObjectError := s.Client.PutObject(s.Context, &s3.PutObjectInput{
		Bucket: &s.Bucket,
		Key:    aws.String(fmt.Sprintf("%s_%s.mp4", video.Title, video.Id)),
		Body:   file,
		Metadata: map[string]string{
			"id":           video.Id,
			"title":        video.Title,
			"description":  video.Description,
			"duration":     strconv.Itoa(int(video.LengthSeconds)),
			"publish_date": video.PublishedAt.String(),
		},
	})
	if putObjectError != nil {
		return putObjectError
	}

	logger.Info().Msgf("video %s stored in s3 bucket %s", video.Title, s.Bucket)

	return nil
}

package source

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

var (
	ErrInvalidS3Path = errors.New("invalid S3 path")
	ErrS3BucketEmpty = errors.New("S3 bucket name is empty")
)

type S3Source struct {
	client    *s3.Client
	bucket    string
	prefix    string
	region    string
	endpoint  string
	workers   int
	accessKey string
	secretKey string
}

type S3Option func(*S3Source)

func WithS3Region(region string) S3Option {
	return func(s *S3Source) {
		s.region = region
	}
}

func WithS3Endpoint(endpoint string) S3Option {
	return func(s *S3Source) {
		s.endpoint = endpoint
	}
}

func WithS3Workers(n int) S3Option {
	return func(s *S3Source) {
		if n > 0 {
			s.workers = n
		}
	}
}

func WithS3Credentials(accessKey, secretKey string) S3Option {
	return func(s *S3Source) {
		s.accessKey = accessKey
		s.secretKey = secretKey
	}
}

func NewS3Source(bucket, prefix string, opts ...S3Option) *S3Source {
	s := &S3Source{
		bucket:  bucket,
		prefix:  prefix,
		workers: 8,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

func (s *S3Source) initClient(ctx context.Context) error {
	opts := []func(*config.LoadOptions) error{}

	if s.region != "" {
		opts = append(opts, config.WithRegion(s.region))
	}

	if s.accessKey != "" && s.secretKey != "" {
		opts = append(opts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(s.accessKey, s.secretKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return err
	}

	clientOpts := []func(*s3.Options){}
	if s.endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(s.endpoint)
			o.UsePathStyle = true
		})
	}

	s.client = s3.NewFromConfig(cfg, clientOpts...)
	return nil
}

func (s *S3Source) List(ctx context.Context) (<-chan Result, error) {
	if s.bucket == "" {
		return nil, ErrS3BucketEmpty
	}

	if s.client == nil {
		if err := s.initClient(ctx); err != nil {
			return nil, err
		}
	}

	results := make(chan Result, s.workers*10)
	go s.listObjects(ctx, results)

	return results, nil
}

func (s *S3Source) listObjects(ctx context.Context, results chan<- Result) {
	defer close(results)

	paginator := s3.NewListObjectsV2Paginator(s.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(s.bucket),
		Prefix: aws.String(s.prefix),
	})

	var wg sync.WaitGroup
	sem := make(chan struct{}, s.workers)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			results <- Result{Err: err}
			return
		}

		for _, obj := range page.Contents {
			if obj.Key == nil || strings.HasSuffix(*obj.Key, "/") {
				continue
			}

			wg.Add(1)
			go func(obj types.Object) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()

				f := File{
					Path:   "s3://" + s.bucket + "/" + *obj.Key,
					Size:   *obj.Size,
					Source: "s3",
				}

				if obj.LastModified != nil {
					f.ModTime = obj.LastModified.Unix()
				}

				results <- Result{File: f}
			}(obj)
		}
	}

	wg.Wait()
}

func (s *S3Source) Read(ctx context.Context, path string) ([]byte, error) {
	if s.client == nil {
		if err := s.initClient(ctx); err != nil {
			return nil, err
		}
	}

	key := strings.TrimPrefix(path, "s3://"+s.bucket+"/")
	if key == path {
		return nil, ErrInvalidS3Path
	}

	resp, err := s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}

func (s *S3Source) Close() error {
	return nil
}

func parseS3Path(path string) (bucket, prefix string, err error) {
	path = strings.TrimPrefix(path, "s3://")
	if path == "" {
		return "", "", ErrInvalidS3Path
	}

	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", "", ErrInvalidS3Path
	}

	bucket = parts[0]
	if len(parts) > 1 {
		prefix = parts[1]
	}

	return bucket, prefix, nil
}

func openS3(path string) (Source, error) {
	bucket, prefix, err := parseS3Path(path)
	if err != nil {
		return nil, err
	}
	return NewS3Source(bucket, prefix), nil
}

func init() {
	Register("s3", openS3)
}

func ParseS3Path(path string) (bucket, prefix string, err error) {
	return parseS3Path(path)
}

func FormatS3Path(bucket, key string) string {
	return "s3://" + bucket + "/" + key
}

func GetS3PresignedURL(ctx context.Context, bucket, key string, expiry time.Duration, opts ...S3Option) (string, error) {
	src := NewS3Source(bucket, key, opts...)
	if err := src.initClient(ctx); err != nil {
		return "", err
	}

	presignClient := s3.NewPresignClient(src.client)
	req, err := presignClient.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return "", err
	}

	return req.URL, nil
}

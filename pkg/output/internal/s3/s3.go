package s3

import (
	"bytes"
	"context"
	"fmt"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
)

// Client wraps an S3-compatible object store client.
type Client struct {
	mc     *minio.Client
	bucket string
	log    logrus.FieldLogger
}

// PutOptions describes per-object metadata applied at upload time.
// Body bytes are uploaded verbatim — callers that want compression or
// encoding must do it before calling PutObject and set the matching
// header fields here.
type PutOptions struct {
	// ContentType is the MIME type (e.g. "text/plain").
	ContentType string

	// ContentEncoding is the Content-Encoding header (e.g. "gzip" when
	// Body is already gzip-compressed).
	ContentEncoding string
}

// New builds a Client. The connection is established lazily on first use;
// callers should issue a HEAD against the bucket via HeadBucket if they
// want startup-time validation.
func New(log logrus.FieldLogger, cfg *Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	mc, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKeyID, cfg.SecretAccessKey, ""),
		Secure: !cfg.Insecure,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("constructing s3 client: %w", err)
	}

	return &Client{
		mc:     mc,
		bucket: cfg.Bucket,
		log:    log,
	}, nil
}

// Bucket returns the bucket name this client targets.
func (c *Client) Bucket() string {
	return c.bucket
}

// HeadBucket checks the configured bucket is reachable and the
// credentials work. Suitable to call from a sink's Start.
func (c *Client) HeadBucket(ctx context.Context) error {
	ok, err := c.mc.BucketExists(ctx, c.bucket)
	if err != nil {
		return fmt.Errorf("checking bucket %q: %w", c.bucket, err)
	}

	if !ok {
		return fmt.Errorf("bucket %q does not exist or is not accessible", c.bucket)
	}

	return nil
}

// PutObject uploads body as a single object at the given key. The body
// is uploaded verbatim — no compression, no encoding transforms.
func (c *Client) PutObject(ctx context.Context, key string, body []byte, opts PutOptions) error {
	_, err := c.mc.PutObject(
		ctx,
		c.bucket,
		key,
		bytes.NewReader(body),
		int64(len(body)),
		minio.PutObjectOptions{
			ContentType:     opts.ContentType,
			ContentEncoding: opts.ContentEncoding,
		},
	)
	if err != nil {
		return fmt.Errorf("putting object %q to bucket %q: %w", key, c.bucket, err)
	}

	return nil
}

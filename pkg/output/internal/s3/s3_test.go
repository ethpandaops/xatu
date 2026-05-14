package s3_test

import (
	"context"
	"io"
	"net/url"
	"testing"
	"time"

	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/minio"

	xs3 "github.com/ethpandaops/xatu/pkg/output/internal/s3"
)

func TestConfigValidate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		cfg  *xs3.Config
		err  string
	}{
		{"nil", nil, "config is required"},
		{"missing endpoint", &xs3.Config{Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"}, "endpoint is required"},
		{"missing bucket", &xs3.Config{Endpoint: "e", AccessKeyID: "k", SecretAccessKey: "s"}, "bucket is required"},
		{"missing access key", &xs3.Config{Endpoint: "e", Bucket: "b", SecretAccessKey: "s"}, "accessKeyId is required"},
		{"missing secret", &xs3.Config{Endpoint: "e", Bucket: "b", AccessKeyID: "k"}, "secretAccessKey is required"},
		{"ok", &xs3.Config{Endpoint: "e", Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s"}, ""},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := tc.cfg.Validate()
			if tc.err == "" {
				assert.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.err)
		})
	}
}

func TestClient_PutObject_HeadBucket(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	endpoint, accessKey, secretKey := startMinio(t, ctx)

	bucket := "xatu-test"
	createBucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	client, err := xs3.New(logrus.New(), &xs3.Config{
		Endpoint:        endpoint,
		Bucket:          bucket,
		Region:          "us-east-1",
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Insecure:        true,
	})
	require.NoError(t, err)

	require.NoError(t, client.HeadBucket(ctx), "HeadBucket should succeed against a real bucket")

	body := []byte("hello world")
	require.NoError(t, client.PutObject(ctx, "greetings/hello.txt", body, xs3.PutOptions{
		ContentType: "text/plain",
	}))

	got := getObject(t, ctx, endpoint, accessKey, secretKey, bucket, "greetings/hello.txt")
	assert.Equal(t, body, got, "uploaded body should be byte-identical to what was put")
}

func TestClient_HeadBucket_NotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	endpoint, accessKey, secretKey := startMinio(t, ctx)

	client, err := xs3.New(logrus.New(), &xs3.Config{
		Endpoint:        endpoint,
		Bucket:          "does-not-exist",
		Region:          "us-east-1",
		AccessKeyID:     accessKey,
		SecretAccessKey: secretKey,
		Insecure:        true,
	})
	require.NoError(t, err)

	err = client.HeadBucket(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist or is not accessible")
}

// startMinio brings up a one-shot MinIO container, returning the host:port
// endpoint and root credentials. The container is torn down at test end.
func startMinio(t *testing.T, ctx context.Context) (endpoint, accessKey, secretKey string) {
	t.Helper()

	c, err := minio.Run(ctx, "minio/minio:RELEASE.2024-09-13T20-26-02Z")
	require.NoError(t, err)

	t.Cleanup(func() {
		if termErr := c.Terminate(context.Background()); termErr != nil {
			t.Logf("terminating minio container: %v", termErr)
		}
	})

	connStr, err := c.ConnectionString(ctx)
	require.NoError(t, err)

	u, err := url.Parse("http://" + connStr)
	require.NoError(t, err)

	return u.Host, c.Username, c.Password
}

func createBucket(t *testing.T, ctx context.Context, endpoint, ak, sk, bucket string) {
	t.Helper()

	mc, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(ak, sk, ""),
		Secure: false,
	})
	require.NoError(t, err)

	require.NoError(t, mc.MakeBucket(ctx, bucket, miniogo.MakeBucketOptions{}))
}

func getObject(t *testing.T, ctx context.Context, endpoint, ak, sk, bucket, key string) []byte {
	t.Helper()

	mc, err := miniogo.New(endpoint, &miniogo.Options{
		Creds:  credentials.NewStaticV4(ak, sk, ""),
		Secure: false,
	})
	require.NoError(t, err)

	obj, err := mc.GetObject(ctx, bucket, key, miniogo.GetObjectOptions{})
	require.NoError(t, err)

	defer obj.Close()

	body, err := io.ReadAll(obj)
	require.NoError(t, err)

	return body
}

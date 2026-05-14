package s3blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	xs3 "github.com/ethpandaops/xatu/pkg/output/internal/s3"
	"github.com/ethpandaops/xatu/pkg/processor"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	miniogo "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/minio"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const gzSuffix = ".gz"

// fakeUploader records every PutObject call for in-memory assertion.
type fakeUploader struct {
	mu          sync.Mutex
	puts        map[string]putRecord
	headErr     error
	putErr      error
	keyedErrs   map[string]error
	beforePut   func(ctx context.Context, key string) error
	inflight    atomic.Int64
	maxInflight atomic.Int64
}

type putRecord struct {
	body []byte
	opts xs3.PutOptions
}

func newFakeUploader() *fakeUploader {
	return &fakeUploader{puts: make(map[string]putRecord, 4)}
}

func (f *fakeUploader) HeadBucket(_ context.Context) error {
	return f.headErr
}

func (f *fakeUploader) PutObject(ctx context.Context, key string, body []byte, opts xs3.PutOptions) error {
	cur := f.inflight.Add(1)
	defer f.inflight.Add(-1)

	for {
		peak := f.maxInflight.Load()
		if cur <= peak || f.maxInflight.CompareAndSwap(peak, cur) {
			break
		}
	}

	if f.beforePut != nil {
		if err := f.beforePut(ctx, key); err != nil {
			return err
		}
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	if f.putErr != nil {
		return f.putErr
	}

	if err, ok := f.keyedErrs[key]; ok {
		return err
	}

	f.puts[key] = putRecord{body: append([]byte(nil), body...), opts: opts}

	return nil
}

func (f *fakeUploader) get(key string) (putRecord, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()

	r, ok := f.puts[key]

	return r, ok
}

func (f *fakeUploader) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.puts)
}

func newTestSink(t *testing.T, up *fakeUploader, cfg *Config) *Sink {
	t.Helper()

	if cfg == nil {
		cfg = &Config{
			Config: xs3.Config{
				Endpoint:        "test.local",
				Bucket:          "blobs",
				AccessKeyID:     "x",
				SecretAccessKey: "y",
			},
			KeySuffix:   gzSuffix,
			Concurrency: 4,
		}
	}

	filter, err := xatu.NewEventFilter(&xatu.EventFilterConfig{})
	require.NoError(t, err)

	return &Sink{
		name:        "test",
		log:         logrus.New(),
		client:      up,
		filter:      filter,
		keyPrefix:   cfg.KeyPrefix,
		keySuffix:   cfg.KeySuffix,
		concurrency: cfg.Concurrency,
	}
}

func newBlobEvent(network, versionedHash, blobHex string) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR,
			Id:       "evt-1",
			DateTime: timestamppb.New(time.Now()),
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{Name: network},
				},
				AdditionalData: &xatu.ClientMeta_EthV1BeaconBlobSidecar{
					EthV1BeaconBlobSidecar: &xatu.ClientMeta_AdditionalEthV1BeaconBlobSidecarData{
						VersionedHash: versionedHash,
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_EthV1BeaconBlockBlobSidecar{
			EthV1BeaconBlockBlobSidecar: &xatuethv1.BlobSidecar{
				Blob: blobHex,
			},
		},
	}
}

func newNonBlobEvent() *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			Id:   "evt-block",
		},
	}
}

func TestHandleNewDecoratedEvents_UploadsBlobAtExpectedKey(t *testing.T) {
	const (
		network   = "mainnet"
		versioned = "0x01abcdef"
		blobHex   = "0x1234"
	)

	up := newFakeUploader()
	s := newTestSink(t, up, nil)

	err := s.HandleNewDecoratedEvents(context.Background(),
		[]*xatu.DecoratedEvent{newBlobEvent(network, versioned, blobHex)})
	require.NoError(t, err)

	rec, ok := up.get("mainnet/0x01abcdef.gz")
	require.True(t, ok, "expected object at network/<versioned_hash>.gz")
	assert.Equal(t, "text/plain", rec.opts.ContentType)
	assert.Equal(t, "gzip", rec.opts.ContentEncoding)

	gz, err := gzip.NewReader(bytes.NewReader(rec.body))
	require.NoError(t, err)

	plain, err := io.ReadAll(gz)
	require.NoError(t, err)

	assert.Equal(t, blobHex, string(plain), "decompressed body should equal the original 0x-hex blob payload")
}

func TestHandleNewDecoratedEvents_KeyPrefixIsHonoured(t *testing.T) {
	up := newFakeUploader()
	cfg := &Config{
		Config: xs3.Config{
			Endpoint: "e", Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s",
		},
		KeyPrefix:   "archive/",
		KeySuffix:   gzSuffix,
		Concurrency: 1,
	}

	s := newTestSink(t, up, cfg)

	err := s.HandleNewDecoratedEvents(context.Background(),
		[]*xatu.DecoratedEvent{newBlobEvent("holesky", "0x01ff", "0xab")})
	require.NoError(t, err)

	_, ok := up.get("archive/holesky/0x01ff.gz")
	assert.True(t, ok, "key prefix should be prepended verbatim")
}

func TestHandleNewDecoratedEvents_IgnoresNonBlobEvents(t *testing.T) {
	up := newFakeUploader()
	s := newTestSink(t, up, nil)

	err := s.HandleNewDecoratedEvents(context.Background(), []*xatu.DecoratedEvent{
		newNonBlobEvent(),
		newNonBlobEvent(),
	})
	require.NoError(t, err)

	assert.Equal(t, 0, up.count(), "non-blob events should not produce any uploads")
}

func TestHandleNewDecoratedEvents_MalformedBlobFailsClosed(t *testing.T) {
	cases := []struct {
		name  string
		event *xatu.DecoratedEvent
	}{
		{"missing network", newBlobEvent("", "0x01ff", "0xab")},
		{"missing versioned hash", newBlobEvent("mainnet", "", "0xab")},
		{"missing blob payload", newBlobEvent("mainnet", "0x01ff", "")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			up := newFakeUploader()
			s := newTestSink(t, up, nil)

			err := s.HandleNewDecoratedEvents(context.Background(),
				[]*xatu.DecoratedEvent{tc.event})
			require.Error(t, err,
				"malformed blob events must fail closed so cannon retries; "+
					"silent skip would advance the checkpoint past data the archive never saw")
			assert.Contains(t, err.Error(), "missing required fields")
			assert.Equal(t, 0, up.count(),
				"no upload should be attempted when any required field is empty")
		})
	}
}

func TestHandleNewDecoratedEvents_BatchUploadsAll(t *testing.T) {
	up := newFakeUploader()
	s := newTestSink(t, up, nil)

	batch := []*xatu.DecoratedEvent{
		newBlobEvent("mainnet", "0x01aa", "0x01"),
		newBlobEvent("mainnet", "0x01bb", "0x02"),
		newBlobEvent("mainnet", "0x01cc", "0x03"),
		newNonBlobEvent(),
	}

	err := s.HandleNewDecoratedEvents(context.Background(), batch)
	require.NoError(t, err)

	assert.Equal(t, 3, up.count(), "only blob events should produce uploads")

	for _, h := range []string{"0x01aa", "0x01bb", "0x01cc"} {
		_, ok := up.get("mainnet/" + h + gzSuffix)
		assert.True(t, ok, "missing object for %s", h)
	}
}

func TestHandleNewDecoratedEvents_PutErrorPropagates(t *testing.T) {
	putErr := errors.New("upload boom")
	up := &fakeUploader{puts: map[string]putRecord{}, putErr: putErr}
	s := newTestSink(t, up, nil)

	err := s.HandleNewDecoratedEvents(context.Background(),
		[]*xatu.DecoratedEvent{newBlobEvent("mainnet", "0x01aa", "0x01")})
	require.Error(t, err)
	assert.ErrorIs(t, err, putErr, "underlying error should be wrapped, not swallowed")
}

func TestHandleNewDecoratedEvents_EmptyBatchIsNoop(t *testing.T) {
	up := newFakeUploader()
	s := newTestSink(t, up, nil)

	require.NoError(t, s.HandleNewDecoratedEvents(context.Background(), nil))
	require.NoError(t, s.HandleNewDecoratedEvents(context.Background(), []*xatu.DecoratedEvent{}))
	assert.Equal(t, 0, up.count())
}

func TestStart_ChecksBucketExistence(t *testing.T) {
	bucketErr := errors.New("no such bucket")
	up := &fakeUploader{puts: map[string]putRecord{}, headErr: bucketErr}
	s := newTestSink(t, up, nil)

	err := s.Start(context.Background())
	assert.ErrorIs(t, err, bucketErr)
}

// TestHandleNewDecoratedEvents_ConcurrentPutOneFails simulates 16 blobs
// landing through a Concurrency=4 sink with one keyed put error. Two
// things matter: the error is returned (so cannon does not advance the
// checkpoint), and the other 15 puts still happen (proving the failed
// blob did not poison the rest of the batch).
func TestHandleNewDecoratedEvents_ConcurrentPutOneFails(t *testing.T) {
	const (
		total     = 16
		poisonIdx = 7
	)

	putErr := errors.New("upload boom")

	events := make([]*xatu.DecoratedEvent, total)
	for i := 0; i < total; i++ {
		events[i] = newBlobEvent("mainnet", fmt.Sprintf("0x01%04d", i), fmt.Sprintf("0x%02x", i))
	}

	poisonKey := "mainnet/" + fmt.Sprintf("0x01%04d", poisonIdx) + gzSuffix

	up := &fakeUploader{
		puts:      map[string]putRecord{},
		keyedErrs: map[string]error{poisonKey: putErr},
		// Hold each put for long enough that errgroup's bounded
		// concurrency actually produces observable overlap. Without
		// this the map insert is too fast to race.
		beforePut: func(_ context.Context, _ string) error {
			time.Sleep(5 * time.Millisecond)

			return nil
		},
	}
	s := newTestSink(t, up, &Config{
		Config: xs3.Config{
			Endpoint: "e", Bucket: "b", AccessKeyID: "k", SecretAccessKey: "s",
		},
		KeySuffix:   gzSuffix,
		Concurrency: 4,
	})

	err := s.HandleNewDecoratedEvents(context.Background(), events)
	require.Error(t, err, "any put failure must propagate so cannon retries")
	assert.ErrorIs(t, err, putErr)
	assert.Equal(t, total-1, up.count(),
		"every non-poisoned blob should have been uploaded despite one sibling failing")
	assert.GreaterOrEqual(t, up.maxInflight.Load(), int64(2),
		"Concurrency=4 with 16 blobs should observe at least 2 in-flight puts at peak")
}

// TestHandleNewDecoratedEvents_UserFilterOverridesHardcodedBlobFilter
// covers the interaction between the sink's hard-coded blob_sidecar
// filter and a user-supplied EventFilterConfig. A user filter that
// excludes blob_sidecar must result in zero uploads; a filter that
// excludes some other event must NOT block blob uploads.
func TestHandleNewDecoratedEvents_UserFilterOverridesHardcodedBlobFilter(t *testing.T) {
	t.Run("user excludes blob_sidecar -> no uploads", func(t *testing.T) {
		up := newFakeUploader()
		filter, err := xatu.NewEventFilter(&xatu.EventFilterConfig{
			ExcludeEventNames: []string{"BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR"},
		})
		require.NoError(t, err)

		s := newTestSink(t, up, nil)
		s.filter = filter

		err = s.HandleNewDecoratedEvents(context.Background(),
			[]*xatu.DecoratedEvent{newBlobEvent("mainnet", "0x01aa", "0x01")})
		require.NoError(t, err)
		assert.Equal(t, 0, up.count(),
			"explicit user exclusion must override the sink's blob-only inclusion")
	})

	t.Run("user excludes other event -> blobs still upload", func(t *testing.T) {
		up := newFakeUploader()
		filter, err := xatu.NewEventFilter(&xatu.EventFilterConfig{
			ExcludeEventNames: []string{"BEACON_API_ETH_V2_BEACON_BLOCK"},
		})
		require.NoError(t, err)

		s := newTestSink(t, up, nil)
		s.filter = filter

		err = s.HandleNewDecoratedEvents(context.Background(),
			[]*xatu.DecoratedEvent{newBlobEvent("mainnet", "0x01aa", "0x01")})
		require.NoError(t, err)
		assert.Equal(t, 1, up.count(),
			"a user filter for an unrelated event must not block blob uploads")
	})
}

// TestHandleNewDecoratedEvents_ContextCancellationPropagates cancels
// the caller's context mid-batch. The sink should observe cancellation
// at the upload boundary and surface it.
func TestHandleNewDecoratedEvents_ContextCancellationPropagates(t *testing.T) {
	up := &fakeUploader{
		puts: map[string]putRecord{},
		// Block PutObject until ctx is cancelled — proves the sink
		// surfaces ctx.Err rather than swallowing the cancellation.
		beforePut: func(ctx context.Context, _ string) error {
			<-ctx.Done()

			return ctx.Err()
		},
	}

	s := newTestSink(t, up, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel shortly after the call starts so the uploads in-flight
	// see the cancellation.
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := s.HandleNewDecoratedEvents(ctx, []*xatu.DecoratedEvent{
		newBlobEvent("mainnet", "0x01aa", "0x01"),
		newBlobEvent("mainnet", "0x01bb", "0x02"),
	})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.Canceled,
		"canceled context must propagate to the caller through errgroup")
}

func TestSinkInterfaceCompliance(t *testing.T) {
	up := newFakeUploader()
	s := newTestSink(t, up, nil)

	assert.Equal(t, "test", s.Name())
	assert.Equal(t, "s3blobstore", s.Type())
	require.NoError(t, s.Stop(context.Background()))
}

// TestEndToEnd_AgainstMinio exercises the full Sink against a real MinIO
// container: New → Start (HeadBucket) → HandleNewDecoratedEvents → object
// readable from the store and matches the expected blob bytes after gunzip.
func TestEndToEnd_AgainstMinio(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	endpoint, accessKey, secretKey := startMinio(t, ctx)

	const bucket = "xatu-blobs"
	createBucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	cfg := &Config{
		Config: xs3.Config{
			Endpoint:        endpoint,
			Bucket:          bucket,
			Region:          "us-east-1",
			AccessKeyID:     accessKey,
			SecretAccessKey: secretKey,
			Insecure:        true,
		},
		KeySuffix:   gzSuffix,
		Concurrency: 4,
	}

	sink, err := New("e2e", cfg, logrus.New(), &xatu.EventFilterConfig{}, processor.ShippingMethodSync)
	require.NoError(t, err)

	require.NoError(t, sink.Start(ctx))
	t.Cleanup(func() { _ = sink.Stop(ctx) })

	const (
		network   = "mainnet"
		versioned = "0x01deadbeef"
		blobHex   = "0xc0ffee"
	)

	require.NoError(t, sink.HandleNewDecoratedEvents(ctx,
		[]*xatu.DecoratedEvent{newBlobEvent(network, versioned, blobHex)}))

	body := getObject(t, ctx, endpoint, accessKey, secretKey, bucket, "mainnet/0x01deadbeef.gz")

	gz, err := gzip.NewReader(bytes.NewReader(body))
	require.NoError(t, err)

	plain, err := io.ReadAll(gz)
	require.NoError(t, err)

	assert.Equal(t, blobHex, string(plain))
}

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

// Package s3blobstore implements an output sink that archives beacon
// blob sidecars to an S3-compatible object store (Cloudflare R2,
// AWS S3, MinIO).
//
// It is purpose-built for the blob-archive use case: only
// BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR events are processed; all other
// event types are skipped silently. Each blob produces a single object
// keyed by `<keyPrefix><network>/<versioned_hash><keySuffix>`, with the
// blob payload (hex-encoded, "0x..." form) uploaded gzip-compressed.
// This matches the existing Vector-based blob-archive output 1:1.
//
// Unlike batch sinks (xatu/kafka/http/stdout), this sink does NOT use
// processor.BatchItemProcessor. HandleNewDecoratedEvents flushes inline,
// so cannon's deriver only advances its checkpoint once the uploads
// have been accepted by the object store.
package s3blobstore

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/output/internal/s3"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"golang.org/x/sync/errgroup"
)

// SinkType identifies this sink in output.Config.
const SinkType = "s3blobstore"

const (
	contentType     = "text/plain"
	contentEncoding = "gzip"
)

// s3Uploader is the subset of s3.Client the sink uses. Declared as an
// interface so tests can substitute a stub without spinning up a real
// object store.
type s3Uploader interface {
	HeadBucket(ctx context.Context) error
	PutObject(ctx context.Context, key string, body []byte, opts s3.PutOptions) error
}

// Sink archives beacon blob sidecars to an S3-compatible store.
type Sink struct {
	name        string
	log         logrus.FieldLogger
	client      s3Uploader
	filter      xatu.EventFilter
	keyPrefix   string
	keySuffix   string
	concurrency int
}

// New constructs an s3blobstore sink. shippingMethod is accepted for
// interface uniformity but ignored — see the package-level godoc.
func New(
	name string,
	cfg *Config,
	log logrus.FieldLogger,
	filterConfig *xatu.EventFilterConfig,
	shippingMethod processor.ShippingMethod,
) (*Sink, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	sLog := log.
		WithField("output_name", name).
		WithField("output_type", SinkType)

	if shippingMethod != "" && shippingMethod != processor.ShippingMethodSync {
		sLog.WithField("shipping_method", shippingMethod).
			Warn("s3blobstore sink ignores shippingMethod — uploads are inline per call so the deriver checkpoint advances only after the object store has acknowledged each blob")
	}

	client, err := s3.New(sLog, &cfg.Config)
	if err != nil {
		return nil, fmt.Errorf("creating s3 client: %w", err)
	}

	filter, err := xatu.NewEventFilter(filterConfig)
	if err != nil {
		return nil, err
	}

	return &Sink{
		name:        name,
		log:         sLog,
		client:      client,
		filter:      filter,
		keyPrefix:   cfg.KeyPrefix,
		keySuffix:   cfg.KeySuffix,
		concurrency: cfg.Concurrency,
	}, nil
}

// Name returns the sink instance's user-supplied name.
func (s *Sink) Name() string {
	return s.name
}

// Type returns the sink type identifier.
func (s *Sink) Type() string {
	return SinkType
}

// Start validates the configured bucket is reachable.
func (s *Sink) Start(ctx context.Context) error {
	return s.client.HeadBucket(ctx)
}

// Stop is a no-op — the underlying client has no resources to release.
func (s *Sink) Stop(_ context.Context) error {
	return nil
}

// HandleNewDecoratedEvent flushes a single event.
func (s *Sink) HandleNewDecoratedEvent(ctx context.Context, event *xatu.DecoratedEvent) error {
	return s.HandleNewDecoratedEvents(ctx, []*xatu.DecoratedEvent{event})
}

// HandleNewDecoratedEvents uploads every blob_sidecar event in the
// batch, returning a joined error if any upload fails so the caller's
// checkpoint does not advance on partial failure. Non-blob events are
// dropped silently — the broader cannon fan-out delivers every deriver
// event to every sink, so this sink must filter to its own concern.
func (s *Sink) HandleNewDecoratedEvents(ctx context.Context, events []*xatu.DecoratedEvent) error {
	if len(events) == 0 {
		return nil
	}

	blobs := make([]*xatu.DecoratedEvent, 0, len(events))

	for _, event := range events {
		if event == nil {
			continue
		}

		if event.GetEvent().GetName() != xatu.Event_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR {
			continue
		}

		shouldDrop, err := s.filter.ShouldBeDropped(event)
		if err != nil {
			return fmt.Errorf("filtering event: %w", err)
		}

		if shouldDrop {
			continue
		}

		blobs = append(blobs, event)
	}

	if len(blobs) == 0 {
		return nil
	}

	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(s.concurrency)

	for _, blob := range blobs {
		g.Go(func() error {
			return s.uploadBlob(gctx, blob)
		})
	}

	return g.Wait()
}

func (s *Sink) uploadBlob(ctx context.Context, event *xatu.DecoratedEvent) error {
	network := event.GetMeta().GetClient().GetEthereum().GetNetwork().GetName()
	versionedHash := event.GetMeta().GetClient().GetEthV1BeaconBlobSidecar().GetVersionedHash()
	blob := event.GetEthV1BeaconBlockBlobSidecar().GetBlob()

	if network == "" || versionedHash == "" || blob == "" {
		// Fail closed. Cannon always populates these for blob_sidecar
		// events; if any is empty, the cluster shape has changed and
		// swallowing the event would silently advance the checkpoint
		// past data that never reached the archive. Returning an error
		// pins cannon on this batch until the upstream is fixed.
		return fmt.Errorf(
			"blob_sidecar event %s missing required fields (network=%t versioned_hash=%t blob=%t) — refusing to archive",
			event.GetEvent().GetId(),
			network != "",
			versionedHash != "",
			blob != "",
		)
	}

	body, err := gzipBytes([]byte(blob))
	if err != nil {
		return fmt.Errorf("gzipping blob %s: %w", versionedHash, err)
	}

	key := s.keyPrefix + network + "/" + versionedHash + s.keySuffix

	if err := s.client.PutObject(ctx, key, body, s3.PutOptions{
		ContentType:     contentType,
		ContentEncoding: contentEncoding,
	}); err != nil {
		return fmt.Errorf("uploading blob %s: %w", key, err)
	}

	return nil
}

func gzipBytes(in []byte) ([]byte, error) {
	var buf bytes.Buffer

	w := gzip.NewWriter(&buf)

	if _, err := w.Write(in); err != nil {
		_ = w.Close()

		return nil, err
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

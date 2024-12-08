package xatu

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	pb "github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/retry"
	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	grpc_codes "google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type ItemExporter struct {
	config  *Config
	log     logrus.FieldLogger
	headers map[string]string

	client pb.EventIngesterClient
	conn   *grpc.ClientConn

	// As the xatu exporter supports streaming, these attributes specific
	// to streaming.
	stream     pb.EventIngester_CreateEventsStreamClient
	streamLock sync.Mutex
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (*ItemExporter, error) {
	log = log.WithFields(logrus.Fields{
		"output_name": name,
		"output_type": SinkType,
	})

	config.Name = name

	if config.Streaming.Enabled {
		log.Info("Streaming enabled for item exporter")
	} else {
		log.Info("Unary (non-streaming) enabled for item exporter")
	}

	opts := []grpc.DialOption{
		grpc.WithChainUnaryInterceptor(grpc_prometheus.UnaryClientInterceptor, retry.UnaryClientInterceptor()),
		grpc.WithChainStreamInterceptor(grpc_prometheus.StreamClientInterceptor, retry.StreamClientInterceptor()),
	}

	if config.KeepAlive.Enabled != nil && *config.KeepAlive.Enabled {
		log.
			WithField("keepalive_time", config.KeepAlive.Time).
			WithField("keepalive_timeout", config.KeepAlive.Timeout).
			Info("Enabling keepalive")

		opts = append(opts, grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                config.KeepAlive.Time,
			Timeout:             config.KeepAlive.Timeout,
			PermitWithoutStream: true,
		}))
	} else {
		log.Info("Disabling keepalive")
	}

	if config.TLS {
		host, _, err := net.SplitHostPort(config.Address)
		if err != nil {
			return nil, fmt.Errorf("fail to get host from address: %v", err)
		}

		opts = append(opts, grpc.WithTransportCredentials(credentials.NewClientTLSFromCert(nil, host)))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(config.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("fail to dial: %v", err)
	}

	return &ItemExporter{
		config:  config,
		headers: config.Headers,
		log:     log,
		conn:    conn,
		client:  pb.NewEventIngesterClient(conn),
	}, nil
}

// ExportItems exports a batch of items via unary RPC.
func (e *ItemExporter) ExportItems(ctx context.Context, items []*pb.DecoratedEvent) error {
	e.log.WithFields(logrus.Fields{"events": len(items), "address": e.config.Address}).Info("Using unary mode")

	_, span := observability.Tracer().Start(ctx, "XatuItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to xatu sink")

	if err := e.sendViaUnary(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

// ExportItemsViaStream exports a batch of items via streaming RPC. This method is responsible for
// ensuring that the stream is initialised before sending any items. If the stream is not active,
// it will attempt to initialise it. If sending fails, it will mark the stream as inactive, allowing
// the caller to handle retries or fallback of their choice.
func (e *ItemExporter) ExportItemsViaStream(ctx context.Context, items []*pb.DecoratedEvent) error {
	log := e.log.WithFields(logrus.Fields{"items_count": len(items), "address": e.config.Address})

	// Ensure the stream is initialized before sending items.
	if err := e.initStream(ctx); err != nil {
		return err
	}

	log.Debug("Sending items via stream")

	if err := e.stream.Send(&pb.CreateEventsRequest{
		Events: items,
	}); err != nil {
		log.WithError(err).Error("Failed to send via stream")

		return fmt.Errorf("failed to send via stream: %w", err)
	}

	log.Debug("Successfully sent items via stream")

	return nil
}

// IsStreaming returns true if streaming is enabled (helper).
func (e *ItemExporter) IsStreaming() bool {
	return e.config.Streaming.Enabled
}

// InitStream initializes a new gRPC stream connection for exporting items.
// It is expected to be called before any items are sent via the stream.
// The method is thread-safe and ensures that only one goroutine can
// initialize the stream at a time.
func (e *ItemExporter) InitStream(ctx context.Context) error {
	return e.initStream(ctx)
}

// CloseStream closes the stream connection.
func (e *ItemExporter) CloseStream(ctx context.Context) error {
	if e.stream != nil {
		return e.stream.CloseSend()
	}

	return nil
}

// Shutdown gracefully shuts down the xatu exporter. It ensures that all resources are cleaned up,
// including closing the gRPC stream and the underlying connection. The method follows these steps:
// 1. Lock the stream to prevent concurrent access during shutdown.
// 2. Attempt to close the stream if its active.
// 3. Create a context with a timeout for shutting down the gRPC connection.
// 4. Attempt to close the gRPC connection and log the result.
// 5. Return any errors encountered during the shutdown process.
func (e *ItemExporter) Shutdown(ctx context.Context) error {
	e.log.Info("Beginning shutdown sequence")

	e.streamLock.Lock()
	defer e.streamLock.Unlock()

	// Close the stream if its active.
	if e.stream != nil {
		e.log.Debug("Closing stream during shutdown")

		if err := e.stream.CloseSend(); err != nil {
			e.log.WithError(err).Debug("Error closing stream during shutdown")
		}

		// Wait briefly for any pending responses to ensure all data is sent.
		time.Sleep(100 * time.Millisecond)

		// Set the stream to nil.
		e.stream = nil
	}

	// Create a context with a timeout for shutting down the gRPC connection.
	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	e.log.Debug("Closing gRPC connection")

	errCh := make(chan error, 1)
	go func() {
		errCh <- e.conn.Close()
	}()

	// Wait for the connection to close or for the context to timeout.
	select {
	case err := <-errCh:
		if err == nil {
			e.log.Info("Successfully shut down connection")

			return nil
		}

		e.log.WithError(err).Error("Error during connection shutdown")

		return err
	case <-shutdownCtx.Done():
		return fmt.Errorf("shutdown timed out after 5s: %w", ctx.Err())
	}
}

func (e *ItemExporter) sendViaUnary(ctx context.Context, items []*pb.DecoratedEvent) error {
	req := &pb.CreateEventsRequest{
		Events: items,
	}

	logCtx := e.log.WithField("num_events", len(items))

	md := metadata.New(e.config.Headers)
	ctx = metadata.NewOutgoingContext(ctx, md)

	for key, value := range e.headers {
		md.Set(key, value)
	}

	var rsp *pb.CreateEventsResponse

	var err error

	opts := []grpc.CallOption{
		grpc.UseCompressor(gzip.Name),
	}

	startTime := time.Now()

	if e.config.Retry.Enabled {
		opts = append(opts,
			retry.WithOnRetryCallback(func(ctx context.Context, attempt uint, err error) {
				duration := time.Since(startTime)

				logCtx.
					WithField("attempt", attempt).
					WithError(err).
					WithField("duration", duration).
					Warn("Failed to export events. Retrying...")

				// Reset the startTime to the current time
				startTime = time.Now()
			}),
			retry.WithMax(uint(e.config.Retry.MaxAttempts)),
			retry.WithBackoff(retry.BackoffExponential(e.config.Retry.Scalar)),
			retry.WithCodes(
				grpc_codes.Unavailable,
				grpc_codes.Internal,
				grpc_codes.ResourceExhausted,
				grpc_codes.Unknown,
				grpc_codes.Canceled,
			),
		)
	}

	rsp, err = e.client.CreateEvents(ctx, req, opts...)
	if err != nil {
		return err
	}

	logCtx.WithFields(logrus.Fields{
		"events_ingested": rsp.GetEventsIngested().GetValue(),
		"events_sent":     len(items),
	}).Debug("Received response from Xatu sink")

	return nil
}

// handleStreamResponses handles incoming responses from the stream.
// It ensures that the stream response errors mark the stream as inactive to
// allow the caller to retry.
func (e *ItemExporter) handleStreamResponses() {
	defer func() {
		e.streamLock.Lock()
		wasActive := e.stream != nil
		e.stream = nil
		e.streamLock.Unlock()

		if wasActive {
			e.log.Info("Stream connection closed")
		}
	}()

	for {
		resp, err := e.stream.Recv()
		if err != nil {
			// Lock the stream to safely mark it as inactive.
			e.streamLock.Lock()
			e.stream = nil
			e.streamLock.Unlock()

			if err == io.EOF {
				e.log.Debug("Stream closed by server (EOF)")

				return
			}

			switch status.Code(err) {
			case grpc_codes.Canceled:
				e.log.Debug("Stream context canceled (expected during shutdown)")
			default:
				e.log.WithFields(logrus.Fields{
					"error":      err,
					"error_type": fmt.Sprintf("%T", err),
					"grpc_code":  status.Code(err),
				}).Error("Unexpected error receiving stream response")
			}

			return
		}

		e.log.WithFields(logrus.Fields{
			"events_ingested": resp.GetEventsIngested().GetValue(),
			"address":         e.config.Address,
		}).Debug("Received streaming response")
	}
}

// initStream checks if the stream is nil and initializes it if necessary.
// This function is responsible for creating a new gRPC stream connection and
// setting it in the exporter. It returns an error if the stream cannot be created.
func (e *ItemExporter) initStream(ctx context.Context) error {
	e.streamLock.Lock()
	defer e.streamLock.Unlock()

	// If the stream is already initialised, do nothing.
	if e.stream != nil {
		return nil
	}

	e.log.WithField("address", e.config.Address).Info("Initializing new gRPC stream connection")

	// Create a new context (with attached metadata) for the stream with
	// cancellation capabilities. This context will be used for the stream's
	// lifecycle and allows for cancellation if needed.
	streamCtx, cancel := context.WithCancel(context.Background())
	md := metadata.New(e.config.Headers)
	streamCtx = metadata.NewOutgoingContext(streamCtx, md)

	// Attempt to create a new stream using the gRPC client.
	stream, err := e.client.CreateEventsStream(streamCtx)
	if err != nil {
		cancel()

		return fmt.Errorf("failed to create stream: %w", err)
	}

	// Store the newly created stream in the exporter for future use.
	e.stream = stream
	e.log.Info("Successfully established gRPC stream connection")

	// Start a goroutine to handle incoming responses from the stream.
	go func() {
		e.handleStreamResponses()
		cancel() // Ensure the context is cancelled when the handler exits.
	}()

	return nil
}

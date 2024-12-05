package eventingester

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"sync/atomic"
	"unsafe"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	ocodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType = "event-ingester"
)

// shutdownState houses fields related to graceful shutdown of the ingester.
// This type encapsulates all shutdown-related state to keep the main Ingester type clean
// and make shutdown coordination logic more maintainable.
type shutdownState struct {
	// draining indicates the server is in full shutdown mode and should reject all new requests.
	// This is the final stage of shutdown after any existing streams have been closed and in-flight
	// events have been processed.
	draining atomic.Bool

	// unaryFallback signals that streams should begin transitioning to unary mode.
	// This is the first stage of shutdown where we want to gracefully close streams
	// but still allow events to be processed via unary endpoints.
	unaryFallback atomic.Bool

	// inFlightEvents tracks events currently being processed. This ensures we don't lose
	// events during shutdown by waiting for all processing to complete before full shutdown.
	// The wg is incremented before processing starts and decremented after completion.
	inFlightEvents sync.WaitGroup

	// shutdownOnce ensures that the shutdown sequence is executed only once.
	shutdownOnce sync.Once
}

// Ingester implements the EventIngester service, providing both unary and streaming endpoints
// for event ingestion.
type Ingester struct {
	xatu.UnimplementedEventIngesterServer

	log     logrus.FieldLogger
	config  *Config
	handler *Handler
	auth    *auth.Authorization
	sinks   []output.Sink

	// shutdown coordinates the graceful shutdown sequence across
	// all components of the ingester.
	shutdown shutdownState
}

// NewIngester creates and initializes a new event ingester service.
// It sets up all required components including auth, event handling, output sinks.
// The ingester won't begin processing events until Start() is called.
func NewIngester(ctx context.Context, log logrus.FieldLogger, conf *Config, clockDrift *time.Duration, geoipProvider geoip.Provider, cache store.Cache) (*Ingester, error) {
	// Decorate logger with service identifier
	log = log.WithField("server/module", ServiceType)

	// Initialize authorization system first as it's required for all operations
	a, err := auth.NewAuthorization(log, conf.Authorization)
	if err != nil {
		return nil, err
	}

	e := &Ingester{
		log:     log,
		config:  conf,
		auth:    a,
		handler: NewHandler(log, clockDrift, geoipProvider, cache, conf.ClientNameSalt),
	}

	sinks, err := e.CreateSinks()
	if err != nil {
		return e, err
	}

	e.sinks = sinks

	return e, nil
}

func (e *Ingester) Start(ctx context.Context, grpcServer *grpc.Server) error {
	e.log.Info("Starting module")

	if err := e.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

	xatu.RegisterEventIngesterServer(grpcServer, e)

	for _, sink := range e.sinks {
		if err := sink.Start(ctx); err != nil {
			return err
		}
	}

	return nil
}

// Stop gracefully shuts down the Ingester and all its sinks. A context timeout is
// introduced to prevent indefinite hanging during shutdown.
func (e *Ingester) Stop(ctx context.Context) error {
	e.log.Info("Beginning ingester shutdown")

	// Here we concurrently shutdown the sinks using a context timeout to prevent
	// any resources hanging, if they take longer than 5 seconds we'll kill them.
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	g := new(errgroup.Group)

	for _, sink := range e.sinks {
		s := sink

		g.Go(func() error {
			if err := s.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop sink %s: %w", s.Name(), err)
			}

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		e.log.Warn("Error during sink shutdown")

		return fmt.Errorf("shutdown error: %w", err)
	}

	e.log.Info("Ingester shutdown completed")

	return nil
}

// CreateEvents handles unary (non-streaming) event ingestion requests.
// Each request contains a batch of events to be processed.
// See: https://grpc.io/docs/what-is-grpc/core-concepts/#unary-rpc
func (e *Ingester) CreateEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	// Use common processing path for both streaming and unary requests.
	return e.processEvents(ctx, req)
}

// CreateEventsStream handles bi-directional streaming event ingestion.
// Clients can send multiple batches of events and receive responses for each batch.
// See: https://grpc.io/docs/what-is-grpc/core-concepts/#bidirectional-streaming-rpc
func (e *Ingester) CreateEventsStream(stream xatu.EventIngester_CreateEventsStreamServer) error {
	// Reject new streams if server is draining.
	if e.shutdown.draining.Load() {
		e.log.Debug("Rejecting stream - server is draining")

		return status.Error(codes.Unavailable, "server is draining")
	}

	e.log.Info("New streaming client connected")

	// Create a cancellable context for this stream. This context will be used to cancel the stream
	// if the server is draining. We cancel upon this method returning, or via monitorStreamShutdown.
	ctx := stream.Context()
	streamCtx, cancel := context.WithCancel(ctx)

	defer cancel()

	// We need to coordinate stream shutdown with in-flight event processing to prevent event loss.
	// This is handled through two monitoring goroutines:
	// 1. monitorStreamShutdown: Watches for context cancellation and ensures in-flight processing
	//    completes before signaling stream closure via the 'done' channel.
	// 2. monitorFallbackMode: Watches for unaryFallback mode (triggered by StartDrain) and closes
	//    the stream once all in-flight events are processed.
	//
	// During shutdown:
	// - StartDrain is called externally, enabling unaryFallback mode.
	// - Existing streams detect this and gracefully close.
	// - New events are processed via unary endpoints.
	// - We track all events with inFlightEvents to ensure completion.

	// 'done' to signal the stream should stop, and 'processing' to track in-flight processing.
	var (
		done       = make(chan struct{})
		processing = make(chan struct{}, 1)
	)

	go e.monitorStreamShutdown(streamCtx, cancel, done, processing)
	go e.monitorFallbackMode(streamCtx, done)

	return e.handleStreamEvents(streamCtx, stream, done, processing)
}

func (e *Ingester) CreateSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(e.config.Outputs))

	for i, out := range e.config.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodSync

			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			e.log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, err
		}

		sinks[i] = sink
	}

	return sinks, nil
}

// StartDrain initiates graceful shutdown of the ingester.
func (e *Ingester) StartDrain() {
	e.shutdown.shutdownOnce.Do(func() {
		// Start by enabling unary fallback mode
		e.shutdown.unaryFallback.Store(true)
		e.log.Info("Event ingester entering unary fallback mode for shutdown")

		// Create a new WaitGroup specifically for shutdown
		var shutdownWg sync.WaitGroup

		shutdownWg.Add(1)

		// Wait for in-flight events in a separate goroutine
		go func() {
			defer shutdownWg.Done()
			e.shutdown.inFlightEvents.Wait()
		}()

		// Wait with timeout for in-flight events to complete
		done := make(chan struct{})
		go func() {
			shutdownWg.Wait()
			close(done)
		}()

		select {
		case <-done:
			e.log.Info("All in-flight events processed")
		case <-time.After(10 * time.Second):
			e.log.Warn("Timeout waiting for in-flight events")
		}

		// Enable full drain mode
		e.shutdown.draining.Store(true)
		e.log.Info("Event ingester now draining")
	})
}

// processEvents is the common path for handling events from both streaming and unary requests.
// It tracks in-flight events and handles graceful shutdown.
func (e *Ingester) processEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	// Track this batch of events as in-flight
	e.shutdown.inFlightEvents.Add(1)
	defer e.shutdown.inFlightEvents.Done()

	// If we're draining, reject the request.
	if e.shutdown.draining.Load() {
		return nil, status.Error(codes.Unavailable, "server is draining")
	}

	return e.handleEvents(ctx, req)
}

// handleEvents processes a batch of events through authorization, filtering, and sinks.
// This is the core event processing logic used by both streaming and unary paths.
func (e *Ingester) handleEvents(ctx context.Context, req *xatu.CreateEventsRequest) (*xatu.CreateEventsResponse, error) {
	ctx, span := observability.Tracer().Start(
		ctx,
		"EventIngester.HandleEvents",
		trace.WithAttributes(attribute.Int64("events", int64(len(req.GetEvents())))),
	)
	defer span.End()

	e.log.WithField("events", len(req.Events)).Debug("Processing events")

	user, group, err := e.handleAuthorization(ctx, span)
	if err != nil {
		return nil, err
	}

	filteredEvents, err := e.handler.Events(ctx, req.GetEvents(), user, group)
	if err != nil {
		errMsg := fmt.Errorf("failed to filter events: %w", err)
		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, status.Error(codes.Internal, errMsg.Error())
	}

	if err := e.sendToSinks(ctx, span, filteredEvents); err != nil {
		return nil, err
	}

	return &xatu.CreateEventsResponse{
		EventsIngested: &wrapperspb.UInt64Value{
			Value: uint64(len(filteredEvents)),
		},
	}, nil
}

// monitorStreamShutdown watches for context cancellation and manages graceful stream shutdown.
// It ensures in-flight processing completes or times out before signaling stream closure.
func (e *Ingester) monitorStreamShutdown(streamCtx context.Context, cancel context.CancelFunc, done, processing chan struct{}) {
	select {
	case <-streamCtx.Done():
		e.log.Info("Connected stream context canceled, initiating stream shutdown")
		cancel()

		// Wait for any in-progress batch to complete.
		<-processing

		e.log.Debug("Connected stream shutdown, in-progress batch processed")

		close(done)
	case <-done:
		e.log.Debug("Connected stream shutdown complete")
	}
}

// monitorFallbackMode watches for the unaryFallback flag and closes streams when fallback
// mode is enabled and no events are in-flight.
func (e *Ingester) monitorFallbackMode(streamCtx context.Context, done chan struct{}) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-streamCtx.Done():
			return
		case <-ticker.C:
			// Check if we should switch to unary mode and no events are being processed.
			if e.shutdown.unaryFallback.Load() && e.getInFlightCount() == 0 {
				e.log.Info("Stream detected unary fallback mode and no in-flight events, closing stream")

				close(done)

				return
			}
		}
	}
}

// handleStreamEvents is the main event processing loop for streaming connections.
// It will continuously receive events and process them until shutdown is signaled.
func (e *Ingester) handleStreamEvents(ctx context.Context, stream xatu.EventIngester_CreateEventsStreamServer, done, processing chan struct{}) error {
	for {
		select {
		case <-done:
			e.log.Info("Stream shutdown complete")

			return nil
		case <-ctx.Done():
			e.log.Info("Stream context canceled")

			return nil
		default:
			if err := e.processNextStreamEvent(ctx, stream, processing); err != nil {
				return err
			}
		}
	}
}

// processNextStreamEvent handles a single batch of events from the stream.
// It coordinates with the processing semaphore to track in-flight events.
func (e *Ingester) processNextStreamEvent(ctx context.Context, stream xatu.EventIngester_CreateEventsStreamServer, processing chan struct{}) error {
	// Receive the next event batch with timeout handling.
	req, err := e.receiveStreamEventWithTimeout(ctx, stream)
	if err != nil {
		return err
	}

	// Track in-flight events via 'processing'.
	select {
	case processing <- struct{}{}:
		// Release 'processing' when done.
		defer func() { <-processing }()

		// Process the received events, sending them to the configured sinks.
		resp, err := e.processEvents(ctx, req)
		if err != nil {
			return err
		}

		// Now fire off our processed response back to the stream.
		return stream.Send(resp)
	case <-ctx.Done():
		e.log.Info("Shutdown requested during batch processing")

		return nil
	}
}

// receiveStreamEventWithTimeout handles receiving events from the stream with context cancellation.
// It uses channels to handle timeouts and ctx cancellation.
func (e *Ingester) receiveStreamEventWithTimeout(ctx context.Context, stream xatu.EventIngester_CreateEventsStreamServer) (*xatu.CreateEventsRequest, error) {
	// Create a context with a timeout.
	// @TODO(matty): Confirm with Sammo/Steve what this timeout should be. Need to think through it.
	timeoutCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	recvChan := make(chan *xatu.CreateEventsRequest, 1)
	errChan := make(chan error, 1)

	// stream.Recv() is blocking, fire off in a goroutine.
	go func() {
		req, err := stream.Recv()
		if err != nil {
			errChan <- err

			return
		}

		recvChan <- req
	}()

	// Wait for receive completion or cancellation.
	select {
	case <-timeoutCtx.Done():
		return nil, status.Error(codes.DeadlineExceeded, "timeout while receiving")
	case err := <-errChan:
		if err == io.EOF {
			e.log.Info("Streaming client disconnected (EOF)")

			return nil, err
		}

		if status.Code(err) == codes.Canceled {
			e.log.Info("Stream canceled by client")

			return nil, err
		}

		e.log.WithError(err).Error("Streaming client error")

		return nil, err
	case req := <-recvChan:
		return req, nil
	}
}

// handleAuthorization handles authorization for incoming requests.
func (e *Ingester) handleAuthorization(ctx context.Context, span trace.Span) (*auth.User, *auth.Group, error) {
	if !e.config.Authorization.Enabled {
		return nil, nil, nil
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, nil, errors.New("failed to get metadata from context")
	}

	authorization := md.Get("authorization")
	if len(authorization) == 0 {
		errMsg := errors.New("no authorization header provided")
		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, nil, status.Error(codes.Unauthenticated, errMsg.Error())
	}

	username, err := e.auth.IsAuthorized(authorization[0])
	if err != nil {
		errMsg := fmt.Errorf("failed to authorize user: %w", err)
		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, nil, status.Error(codes.Unauthenticated, errMsg.Error())
	}

	if username == "" {
		errMsg := errors.New("unauthorized")
		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, nil, status.Error(codes.Unauthenticated, errMsg.Error())
	}

	user, group, err := e.auth.GetUserAndGroup(username)
	if err != nil {
		errMsg := fmt.Errorf("failed to get user and group: %w", err)
		span.SetStatus(ocodes.Error, errMsg.Error())

		return nil, nil, status.Error(codes.Unauthenticated, errMsg.Error())
	}

	span.SetAttributes(attribute.String("user", username))
	span.SetAttributes(attribute.String("group", group.Name()))

	return user, group, nil
}

// sendToSinks sends events to the configured sinks.
func (e *Ingester) sendToSinks(ctx context.Context, parentSpan trace.Span, events []*xatu.DecoratedEvent) error {
	for _, sink := range e.sinks {
		spanCtx, span := observability.Tracer().Start(ctx,
			"EventIngester.SendEventsToSink",
			trace.WithAttributes(
				attribute.String("sink", sink.Name()),
				attribute.Int64("events", int64(len(events))),
			),
		)

		if err := sink.HandleNewDecoratedEvents(spanCtx, events); err != nil {
			errMsg := fmt.Errorf("failed to handle new decorated events: %w", err)
			span.SetStatus(ocodes.Error, errMsg.Error())
			span.End()

			return status.Error(codes.Internal, errMsg.Error())
		}

		span.End()
	}

	return nil
}

// getInFlightCount provides a safe way to access the WaitGroup counter.
// This is used during shutdown to ensure all events are processed.
func (e *Ingester) getInFlightCount() int64 {
	return atomic.LoadInt64((*int64)(unsafe.Pointer(&e.shutdown.inFlightEvents)))
}

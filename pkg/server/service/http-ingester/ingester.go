package httpingester

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/output"
	"github.com/ethpandaops/xatu/pkg/processor"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	ocodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

const (
	ServiceType = "http-ingester"
)

// Ingester handles HTTP event ingestion.
type Ingester struct {
	log           logrus.FieldLogger
	config        *Config
	auth          *auth.Authorization
	handler       *eventingester.Handler
	sinks         []output.Sink
	geoipProvider geoip.Provider
	cache         store.Cache
	clockDrift    *time.Duration
	server        *http.Server
}

// NewIngester creates a new HTTP event ingester.
func NewIngester(
	ctx context.Context,
	log logrus.FieldLogger,
	conf *Config,
	clockDrift *time.Duration,
	geoipProvider geoip.Provider,
	cache store.Cache,
) (*Ingester, error) {
	log = log.WithField("server/module", ServiceType)

	a, err := auth.NewAuthorization(log, conf.EventIngester.Authorization)
	if err != nil {
		return nil, fmt.Errorf("failed to create authorization: %w", err)
	}

	i := &Ingester{
		log:           log,
		config:        conf,
		auth:          a,
		geoipProvider: geoipProvider,
		cache:         cache,
		clockDrift:    clockDrift,
		handler:       eventingester.NewHandler(log, clockDrift, geoipProvider, cache, conf.EventIngester.ClientNameSalt),
	}

	sinks, err := i.createSinks()
	if err != nil {
		return nil, fmt.Errorf("failed to create sinks: %w", err)
	}

	i.sinks = sinks

	return i, nil
}

// Name returns the service name.
func (i *Ingester) Name() string {
	return ServiceType
}

// Start starts the HTTP ingester server.
func (i *Ingester) Start(ctx context.Context) error {
	i.log.WithField("addr", i.config.Addr).Info("Starting HTTP ingester")

	if err := i.auth.Start(ctx); err != nil {
		return fmt.Errorf("failed to start authorization: %w", err)
	}

	for _, sink := range i.sinks {
		if err := sink.Start(ctx); err != nil {
			return fmt.Errorf("failed to start sink %s: %w", sink.Name(), err)
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/events", i.handleEvents)

	i.server = &http.Server{
		Addr:              i.config.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		ReadTimeout:       60 * time.Second,
		WriteTimeout:      60 * time.Second,
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}

	return i.server.ListenAndServe()
}

// Stop stops the HTTP ingester server.
func (i *Ingester) Stop(ctx context.Context) error {
	i.log.Info("Stopping HTTP ingester")

	for _, sink := range i.sinks {
		if err := sink.Stop(ctx); err != nil {
			return fmt.Errorf("failed to stop sink %s: %w", sink.Name(), err)
		}
	}

	if i.server != nil {
		return i.server.Shutdown(ctx)
	}

	return nil
}

// handleEvents handles POST /v1/events requests.
func (i *Ingester) handleEvents(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	ctx, span := observability.Tracer().Start(ctx, "HTTPIngester.handleEvents")
	defer span.End()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)

		return
	}

	// Validate Content-Type (accept both protobuf and JSON)
	contentType := r.Header.Get("Content-Type")
	if contentType != "application/x-protobuf" && contentType != "application/json" {
		http.Error(w, "unsupported content type, expected application/x-protobuf or application/json", http.StatusUnsupportedMediaType)

		return
	}

	// Handle gzip-encoded request body
	var bodyReader io.Reader = r.Body

	if r.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, gzipErr := gzip.NewReader(r.Body)
		if gzipErr != nil {
			span.SetStatus(ocodes.Error, gzipErr.Error())
			http.Error(w, "failed to decompress gzip body", http.StatusBadRequest)

			return
		}

		defer gzipReader.Close()

		bodyReader = gzipReader
	}

	defer r.Body.Close()

	// Read request body
	body, readErr := io.ReadAll(bodyReader)
	if readErr != nil {
		span.SetStatus(ocodes.Error, readErr.Error())
		http.Error(w, "failed to read request body", http.StatusBadRequest)

		return
	}

	// Unmarshal request based on content type
	req := &xatu.CreateEventsRequest{}

	var unmarshalErr error

	if contentType == "application/json" {
		// Vector batches JSON events as an array: [{"events":[...]}, {"events":[...]}, ...]
		// Check if the body is a JSON array and handle accordingly
		trimmedBody := bytes.TrimSpace(body)
		if len(trimmedBody) > 0 && trimmedBody[0] == '[' {
			// Parse as array of CreateEventsRequest and merge all events
			var requests []json.RawMessage
			if jsonErr := json.Unmarshal(trimmedBody, &requests); jsonErr != nil {
				unmarshalErr = fmt.Errorf("failed to parse JSON array: %w", jsonErr)
			} else {
				// Parse each request and collect all events
				allEvents := make([]*xatu.DecoratedEvent, 0, len(requests))

				for idx, rawReq := range requests {
					singleReq := &xatu.CreateEventsRequest{}
					if protoErr := protojson.Unmarshal(rawReq, singleReq); protoErr != nil {
						unmarshalErr = fmt.Errorf("failed to parse request at index %d: %w", idx, protoErr)

						break
					}

					allEvents = append(allEvents, singleReq.GetEvents()...)
				}

				if unmarshalErr == nil {
					req.Events = allEvents
				}
			}
		} else {
			unmarshalErr = protojson.Unmarshal(body, req)
		}
	} else {
		unmarshalErr = proto.Unmarshal(body, req)
	}

	if unmarshalErr != nil {
		i.log.WithError(unmarshalErr).WithField("body_length", len(body)).WithField("content_type", contentType).Error("Failed to unmarshal request")
		span.SetStatus(ocodes.Error, unmarshalErr.Error())
		http.Error(w, fmt.Sprintf("failed to parse request: %v", unmarshalErr), http.StatusBadRequest)

		return
	}

	span.SetAttributes(attribute.Int64("events", int64(len(req.GetEvents()))))

	// Log received events for debugging
	if len(req.GetEvents()) > 0 {
		eventTypes := make([]string, 0, len(req.GetEvents()))
		for _, ev := range req.GetEvents() {
			if ev != nil && ev.Event != nil {
				eventTypes = append(eventTypes, ev.Event.Name.String())
			}
		}

		i.log.WithField("events_count", len(req.GetEvents())).WithField("event_types", eventTypes).Debug("Received events via HTTP")
	}

	// Authenticate
	var user *auth.User

	var group *auth.Group

	if i.config.EventIngester.Authorization.Enabled {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			span.SetStatus(ocodes.Error, "no authorization header")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		username, authErr := i.auth.IsAuthorized(authHeader)
		if authErr != nil {
			span.SetStatus(ocodes.Error, authErr.Error())
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		var userErr error

		user, group, userErr = i.auth.GetUserAndGroup(username)
		if userErr != nil {
			span.SetStatus(ocodes.Error, userErr.Error())
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		span.SetAttributes(attribute.String("user", username))
		span.SetAttributes(attribute.String("group", group.Name()))
	}

	// Extract client IP
	clientIP := i.extractClientIP(r)

	// Create a context with gRPC-style metadata so the handler can extract it
	md := metadata.Pairs(
		"x-forwarded-for", clientIP,
		"x-real-ip", clientIP,
	)
	ctx = metadata.NewIncomingContext(ctx, md)

	// Add a peer to the context (required by the handler for IP extraction fallback)
	// Parse the client IP as a TCP address for the peer
	peerAddr := &net.TCPAddr{IP: net.ParseIP(clientIP)}
	ctx = peer.NewContext(ctx, &peer.Peer{Addr: peerAddr})

	// Process events through the handler
	filteredEvents, handlerErr := i.handler.Events(ctx, req.GetEvents(), user, group)
	if handlerErr != nil {
		i.log.WithError(handlerErr).WithField("events_count", len(req.GetEvents())).Error("Failed to process events")
		span.SetStatus(ocodes.Error, handlerErr.Error())
		http.Error(w, fmt.Sprintf("failed to process events: %v", handlerErr), http.StatusInternalServerError)

		return
	}

	i.log.WithField("filtered_events_count", len(filteredEvents)).WithField("input_events_count", len(req.GetEvents())).Debug("Events processed by handler")

	// Send to sinks
	for _, sink := range i.sinks {
		_, sinkSpan := observability.Tracer().Start(ctx,
			"HTTPIngester.handleEvents.SendEventsToSink",
			trace.WithAttributes(
				attribute.String("sink", sink.Name()),
				attribute.Int64("events", int64(len(filteredEvents))),
			),
		)

		if sinkErr := sink.HandleNewDecoratedEvents(ctx, filteredEvents); sinkErr != nil {
			i.log.WithError(sinkErr).WithField("sink", sink.Name()).WithField("events_count", len(filteredEvents)).Error("Failed to send events to sink")
			sinkSpan.SetStatus(ocodes.Error, sinkErr.Error())
			sinkSpan.End()
			http.Error(w, fmt.Sprintf("failed to send events to sink: %v", sinkErr), http.StatusInternalServerError)

			return
		}

		sinkSpan.End()
	}

	// Build response
	response := &xatu.CreateEventsResponse{
		EventsIngested: &wrapperspb.UInt64Value{
			Value: uint64(len(filteredEvents)),
		},
	}

	respBytes, marshalErr := proto.Marshal(response)
	if marshalErr != nil {
		span.SetStatus(ocodes.Error, marshalErr.Error())
		http.Error(w, "failed to marshal response", http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/x-protobuf")
	w.WriteHeader(http.StatusAccepted)

	if _, err := w.Write(respBytes); err != nil {
		i.log.WithError(err).Error("Failed to write response")
	}
}

// extractClientIP extracts the client IP from HTTP headers or remote address.
// Priority: X-Real-IP > X-Forwarded-For > RemoteAddr
func (i *Ingester) extractClientIP(r *http.Request) string {
	// Check X-Real-IP header first
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Check X-Forwarded-For header
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, take the first one
		ips := strings.Split(forwardedFor, ",")

		return strings.TrimSpace(ips[0])
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}

	return host
}

// createSinks creates the output sinks from configuration.
func (i *Ingester) createSinks() ([]output.Sink, error) {
	sinks := make([]output.Sink, len(i.config.EventIngester.Outputs))

	for idx, out := range i.config.EventIngester.Outputs {
		if out.ShippingMethod == nil {
			shippingMethod := processor.ShippingMethodSync
			out.ShippingMethod = &shippingMethod
		}

		sink, err := output.NewSink(out.Name,
			out.SinkType,
			out.Config,
			i.log,
			out.FilterConfig,
			*out.ShippingMethod,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create sink %s: %w", out.Name, err)
		}

		sinks[idx] = sink
	}

	return sinks, nil
}

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
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	eventingester "github.com/ethpandaops/xatu/pkg/server/service/event-ingester"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/auth"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	ocodes "go.opentelemetry.io/otel/codes"
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
	log      logrus.FieldLogger
	config   *Config
	pipeline *eventingester.Pipeline
	server   *http.Server
}

// NewIngester creates a new HTTP event ingester.
// The eventIngesterConf is the shared event ingester configuration from services.eventIngester.
func NewIngester(
	ctx context.Context,
	log logrus.FieldLogger,
	conf *Config,
	eventIngesterConf *eventingester.Config,
	clockDrift *time.Duration,
	geoipProvider geoip.Provider,
	cache store.Cache,
) (*Ingester, error) {
	log = log.WithField("server/module", ServiceType)

	pipeline, err := eventingester.NewPipeline(ctx, log, eventIngesterConf, clockDrift, geoipProvider, cache)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline: %w", err)
	}

	i := &Ingester{
		log:      log,
		config:   conf,
		pipeline: pipeline,
	}

	return i, nil
}

// Name returns the service name.
func (i *Ingester) Name() string {
	return ServiceType
}

// Start starts the HTTP ingester server.
func (i *Ingester) Start(ctx context.Context) error {
	i.log.WithField("addr", i.config.Addr).Info("Starting HTTP ingester")

	if err := i.pipeline.Start(ctx); err != nil {
		return fmt.Errorf("failed to start pipeline: %w", err)
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

	if err := i.pipeline.Stop(ctx); err != nil {
		return fmt.Errorf("failed to stop pipeline: %w", err)
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

	// Authenticate BEFORE reading body to prevent DDoS via large payload parsing
	var (
		user  *auth.User
		group *auth.Group
	)

	if i.pipeline.AuthEnabled() {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			span.SetStatus(ocodes.Error, "no authorization header")
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		username, authErr := i.pipeline.Auth().IsAuthorized(authHeader)
		if authErr != nil {
			span.SetStatus(ocodes.Error, authErr.Error())
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		var userErr error

		user, group, userErr = i.pipeline.Auth().GetUserAndGroup(username)
		if userErr != nil {
			span.SetStatus(ocodes.Error, userErr.Error())
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		span.SetAttributes(attribute.String("user", username))
		span.SetAttributes(attribute.String("group", group.Name()))
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
		unmarshalOptions := protojson.UnmarshalOptions{DiscardUnknown: true}

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
					if protoErr := unmarshalOptions.Unmarshal(rawReq, singleReq); protoErr != nil {
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
			unmarshalErr = unmarshalOptions.Unmarshal(body, req)
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

	// Process events through the pipeline
	filteredCount, processErr := i.pipeline.ProcessAndSend(ctx, req.GetEvents(), user, group, "HTTPIngester.handleEvents")
	if processErr != nil {
		i.log.WithError(processErr).WithField("events_count", len(req.GetEvents())).Error("Failed to process events")
		span.SetStatus(ocodes.Error, processErr.Error())
		http.Error(w, fmt.Sprintf("failed to process events: %v", processErr), http.StatusInternalServerError)

		return
	}

	i.log.WithField("filtered_events_count", filteredCount).WithField("input_events_count", len(req.GetEvents())).Debug("Events processed by handler")

	// Build response
	response := &xatu.CreateEventsResponse{
		EventsIngested: &wrapperspb.UInt64Value{
			Value: filteredCount,
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

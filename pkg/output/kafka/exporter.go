package kafka

import (
	"context"
	"errors"
	"strings"

	"github.com/IBM/sarama"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// ItemExporter sends batches of events to Kafka topics. It supports both
// a static Topic and a dynamic TopicPattern that resolves per event.
type ItemExporter struct {
	config *Config
	log    observability.ContextualLogger
	client sarama.SyncProducer
}

// NewItemExporter creates a new ItemExporter using the provided SyncProducer.
// The caller is responsible for creating the producer (see NewSyncProducer).
func NewItemExporter(
	name string,
	config *Config,
	log observability.ContextualLogger, producer sarama.SyncProducer,
) ItemExporter {
	return ItemExporter{
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),
		client: producer,
	}
}

// ExportItems sends a batch of decorated events to Kafka.
func (e ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	propagationCtx := ctx

	ctx, span := observability.Tracer().Start(ctx,
		"KafkaItemExporter.ExportItems",
		trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))),
	)
	defer span.End()

	e.log.WithField("events", len(items)).WithContext(ctx).Debug("Sending batch of events to Kafka sink")

	if err := e.sendUpstream(ctx, propagationCtx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).WithContext(ctx).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

// Shutdown closes the underlying Kafka producer to drain inflight messages.
func (e ItemExporter) Shutdown(_ context.Context) error {
	return e.client.Close()
}

func (e *ItemExporter) sendUpstream(ctx, propagationCtx context.Context, items []*xatu.DecoratedEvent) error {
	msgs := make([]*sarama.ProducerMessage, 0, len(items))

	propagator := otel.GetTextMapPropagator()

	for _, p := range items {
		r, err := e.config.MarshalEvent(p)
		if err != nil {
			return err
		}

		topic := e.topicForEvent(p)
		routingKey := sarama.StringEncoder(p.Event.Id)
		eventPayload := sarama.StringEncoder(r)

		m := &sarama.ProducerMessage{
			Topic: topic,
			Key:   routingKey,
			Value: eventPayload,
		}

		propagator.Inject(propagationCtx, newSaramaHeaderCarrier(&m.Headers))

		msgByteSize := m.ByteSize(2)
		if msgByteSize > e.config.MaxMessageBytes {
			e.log.
				WithField("event_id", routingKey).
				WithField("msg_size", msgByteSize).
				WithField("max_message_bytes", e.config.MaxMessageBytes).WithContext(ctx).
				Warn("Message exceeds max_message_bytes, dropping")

			continue
		}

		msgs = append(msgs, m)
	}

	if len(msgs) == 0 {
		return nil
	}

	errorCount := 0

	err := e.client.SendMessages(msgs)
	if err != nil {
		var errs sarama.ProducerErrors
		if errors.As(err, &errs) {
			errorCount = len(errs)

			for _, producerError := range errs {
				e.log.
					WithError(producerError.Err).
					WithField("events", errorCount).WithContext(ctx).
					Error("Failed to send events to Kafka")

				return producerError
			}
		}

		return err
	}

	e.log.WithField("count", len(msgs)-errorCount).WithContext(ctx).Debug("Items written to Kafka")

	return nil
}

// topicForEvent returns the Kafka topic for a given event, dispatching
// between a static Topic and a dynamic TopicPattern.
func (e *ItemExporter) topicForEvent(event *xatu.DecoratedEvent) string {
	if e.config.TopicPattern != "" {
		return resolveTopic(e.config.TopicPattern, event)
	}

	return e.config.Topic
}

// resolveTopic replaces template variables in the topic pattern with values
// derived from the event name. Supported variables:
//   - ${EVENT_NAME}  — raw SCREAMING_SNAKE_CASE name (e.g. BEACON_API_ETH_V1_EVENTS_BLOCK)
//   - ${event-name}  — kebab-case name (e.g. beacon-api-eth-v1-events-block)
func resolveTopic(pattern string, event *xatu.DecoratedEvent) string {
	rawName := event.GetEvent().GetName().String()
	kebabName := strings.ToLower(strings.ReplaceAll(rawName, "_", "-"))

	topic := strings.ReplaceAll(pattern, "${EVENT_NAME}", rawName)
	topic = strings.ReplaceAll(topic, "${event-name}", kebabName)

	return topic
}

package kafka

import (
	"context"
	"errors"
	"strings"

	"github.com/IBM/sarama"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// ItemExporter sends batches of events to Kafka topics. It supports both
// a static Topic and a dynamic TopicPattern that resolves per event.
type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger
	client sarama.SyncProducer
}

// NewItemExporter creates a new ItemExporter using the provided SyncProducer.
// The caller is responsible for creating the producer (see NewSyncProducer).
func NewItemExporter(
	name string,
	config *Config,
	log logrus.FieldLogger,
	producer sarama.SyncProducer,
) ItemExporter {
	return ItemExporter{
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),
		client: producer,
	}
}

// ExportItems sends a batch of decorated events to Kafka.
func (e ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx,
		"KafkaItemExporter.ExportItems",
		trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))),
	)
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to Kafka sink")

	if err := e.sendUpstream(items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
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

func (e *ItemExporter) sendUpstream(items []*xatu.DecoratedEvent) error {
	msgs := make([]*sarama.ProducerMessage, 0, len(items))

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

		msgByteSize := m.ByteSize(2)
		if msgByteSize > e.config.FlushBytes {
			e.log.
				WithField("event_id", routingKey).
				WithField("msg_size", msgByteSize).
				Debug("Message too large, consider increasing `max_message_bytes`")

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
					WithField("events", errorCount).
					Error("Failed to send events to Kafka")

				return producerError
			}
		}

		return err
	}

	e.log.WithField("count", len(msgs)-errorCount).Debug("Items written to Kafka")

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

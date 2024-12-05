package kafka

import (
	"context"
	"errors"

	"github.com/IBM/sarama"
	"github.com/ethpandaops/xatu/pkg/observability"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/protobuf/encoding/protojson"
)

type ItemExporter struct {
	config *Config
	log    logrus.FieldLogger
	client sarama.SyncProducer
}

func NewItemExporter(name string, config *Config, log logrus.FieldLogger) (*ItemExporter, error) {
	producer, err := NewSyncProducer(config)

	if err != nil {
		log.
			WithError(err).
			WithField("output_name", name).
			WithField("output_type", SinkType).
			Error("Error while creating the Kafka Client")

		return nil, err
	}

	return &ItemExporter{
		config: config,
		log:    log.WithField("output_name", name).WithField("output_type", SinkType),
		client: producer,
	}, nil
}
func (e *ItemExporter) ExportItems(ctx context.Context, items []*xatu.DecoratedEvent) error {
	_, span := observability.Tracer().Start(ctx, "KafkaItemExporter.ExportItems", trace.WithAttributes(attribute.Int64("num_events", int64(len(items)))))
	defer span.End()

	e.log.WithField("events", len(items)).Debug("Sending batch of events to Kafka sink")

	if err := e.sendUpstream(ctx, items); err != nil {
		e.log.
			WithError(err).
			WithField("num_events", len(items)).
			Error("Failed to send events upstream")

		span.SetStatus(codes.Error, err.Error())

		return err
	}

	return nil
}

func (e *ItemExporter) Shutdown(ctx context.Context) error {
	return nil
}

func (e *ItemExporter) sendUpstream(ctx context.Context, items []*xatu.DecoratedEvent) error {
	msgs := make([]*sarama.ProducerMessage, 0, len(items))
	msgByteSize := 0

	for _, p := range items {
		r, err := protojson.Marshal(p)
		if err != nil {
			return err
		}

		routingKey, eventPayload := sarama.StringEncoder(p.Event.Id), sarama.StringEncoder(r)
		m := &sarama.ProducerMessage{
			Topic: e.config.Topic,
			Key:   routingKey,
			Value: eventPayload,
		}

		msgByteSize = m.ByteSize(2)
		if msgByteSize > e.config.FlushBytes {
			e.log.WithField("event_id", routingKey).WithField("msg_size", msgByteSize).Debug("Message too large, consider increasing `max_message_bytes`")

			continue
		}

		msgs = append(msgs, m)
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

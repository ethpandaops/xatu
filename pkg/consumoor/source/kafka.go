package source

import (
	"fmt"
	"strconv"

	"github.com/redpanda-data/benthos/v4/public/service"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const unknownKafkaTopic = "unknown"

type kafkaMessageMetadata struct {
	Topic     string
	Partition int32
	Offset    int64
}

func kafkaMetadata(msg *service.Message) kafkaMessageMetadata {
	meta := kafkaMessageMetadata{
		Topic:     unknownKafkaTopic,
		Partition: -1,
		Offset:    -1,
	}

	if msg == nil {
		return meta
	}

	meta.Topic = kafkaTopicMetadata(msg)

	if partition, ok := kafkaInt32Metadata(msg, "kafka_partition"); ok {
		meta.Partition = partition
	}

	if offset, ok := kafkaInt64Metadata(msg, "kafka_offset"); ok {
		meta.Offset = offset
	}

	return meta
}

func kafkaInt64Metadata(msg *service.Message, key string) (int64, bool) {
	if msg == nil {
		return 0, false
	}

	raw, ok := msg.MetaGet(key)
	if !ok || raw == "" {
		return 0, false
	}

	out, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0, false
	}

	return out, true
}

func kafkaInt32Metadata(msg *service.Message, key string) (int32, bool) {
	if msg == nil {
		return 0, false
	}

	raw, ok := msg.MetaGet(key)
	if !ok || raw == "" {
		return 0, false
	}

	out, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		return 0, false
	}

	return int32(out), true
}

func kafkaTopicMetadata(msg *service.Message) string {
	if msg == nil {
		return unknownKafkaTopic
	}

	topic, ok := msg.MetaGet("kafka_topic")
	if !ok || topic == "" {
		return unknownKafkaTopic
	}

	return topic
}

func decodeDecoratedEvent(encoding string, data []byte) (*xatu.DecoratedEvent, error) {
	event := &xatu.DecoratedEvent{}

	switch encoding {
	case "protobuf":
		if err := proto.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("protobuf unmarshal: %w", err)
		}
	default:
		opts := protojson.UnmarshalOptions{DiscardUnknown: true}
		if err := opts.Unmarshal(data, event); err != nil {
			return nil, fmt.Errorf("json unmarshal: %w", err)
		}
	}

	return event, nil
}

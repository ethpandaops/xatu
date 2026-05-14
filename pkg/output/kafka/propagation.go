package kafka

import (
	"github.com/IBM/sarama"
)

// saramaHeaderCarrier adapts sarama.RecordHeaders to the OTel
// propagation.TextMapCarrier interface so producers can inject the
// active trace context into Kafka message headers.
type saramaHeaderCarrier struct {
	headers *[]sarama.RecordHeader
}

func newSaramaHeaderCarrier(headers *[]sarama.RecordHeader) saramaHeaderCarrier {
	return saramaHeaderCarrier{headers: headers}
}

func (c saramaHeaderCarrier) Get(key string) string {
	for _, h := range *c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}

	return ""
}

func (c saramaHeaderCarrier) Set(key, value string) {
	for i, h := range *c.headers {
		if string(h.Key) == key {
			(*c.headers)[i].Value = []byte(value)

			return
		}
	}

	*c.headers = append(*c.headers, sarama.RecordHeader{
		Key:   []byte(key),
		Value: []byte(value),
	})
}

func (c saramaHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(*c.headers))
	for _, h := range *c.headers {
		keys = append(keys, string(h.Key))
	}

	return keys
}

// consumerHeaderCarrier adapts sarama.ConsumerMessage headers to the
// OTel propagation.TextMapCarrier interface for trace context extraction.
type consumerHeaderCarrier struct {
	headers []*sarama.RecordHeader
}

func newConsumerHeaderCarrier(headers []*sarama.RecordHeader) consumerHeaderCarrier {
	return consumerHeaderCarrier{headers: headers}
}

func (c consumerHeaderCarrier) Get(key string) string {
	for _, h := range c.headers {
		if string(h.Key) == key {
			return string(h.Value)
		}
	}

	return ""
}

func (c consumerHeaderCarrier) Set(key, value string) {
	// Extraction-only carrier; Set is unused but required by the interface.
	_ = key
	_ = value
}

func (c consumerHeaderCarrier) Keys() []string {
	keys := make([]string, 0, len(c.headers))
	for _, h := range c.headers {
		keys = append(keys, string(h.Key))
	}

	return keys
}

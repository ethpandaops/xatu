package source

import (
	"github.com/redpanda-data/benthos/v4/public/service"
)

// benthosHeaderCarrier adapts a Benthos service.Message to the OTel
// propagation.TextMapCarrier interface for trace context extraction
// from Kafka headers (which Benthos exposes as message metadata).
type benthosHeaderCarrier struct {
	msg *service.Message
}

func newBenthosHeaderCarrier(msg *service.Message) benthosHeaderCarrier {
	return benthosHeaderCarrier{msg: msg}
}

func (c benthosHeaderCarrier) Get(key string) string {
	if c.msg == nil {
		return ""
	}

	v, ok := c.msg.MetaGet(key)
	if !ok {
		return ""
	}

	return v
}

func (c benthosHeaderCarrier) Set(key, value string) {
	if c.msg == nil {
		return
	}

	c.msg.MetaSet(key, value)
}

func (c benthosHeaderCarrier) Keys() []string {
	if c.msg == nil {
		return nil
	}

	keys := make([]string, 0, 8)

	_ = c.msg.MetaWalk(func(k, _ string) error {
		keys = append(keys, k)

		return nil
	})

	return keys
}

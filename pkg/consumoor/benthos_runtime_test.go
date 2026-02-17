package consumoor

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/redpanda-data/benthos/v4/public/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

func TestBenthosConfigYAML(t *testing.T) {
	cfg := &Config{
		LoggingLevel: "debug",
		Kafka: KafkaConfig{
			Brokers:                []string{"kafka-1:9092", "kafka-2:9092"},
			Topics:                 []string{"^general-.+"},
			ConsumerGroup:          "xatu-consumoor",
			Encoding:               "protobuf",
			FetchMinBytes:          64,
			FetchWaitMaxMs:         250,
			MaxPartitionFetchBytes: 1048576,
			SessionTimeoutMs:       30000,
			HeartbeatIntervalMs:    3000,
			OffsetDefault:          "newest",
			CommitInterval:         7 * time.Second,
		},
	}

	raw, err := benthosConfigYAML(cfg)
	require.NoError(t, err)

	var parsed map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &parsed))

	input, ok := parsed["input"].(map[string]any)
	require.True(t, ok)

	kafka, ok := input["kafka_franz"].(map[string]any)
	require.True(t, ok)

	assert.Equal(t, "xatu-consumoor", kafka["consumer_group"])
	assert.Equal(t, "latest", kafka["start_offset"])
	assert.Equal(t, "7s", kafka["commit_period"])

	output, ok := parsed["output"].(map[string]any)
	require.True(t, ok)

	_, hasOutput := output[benthosOutputType]
	assert.True(t, hasOutput)
}

func TestBenthosSASLObjectUsesPasswordFile(t *testing.T) {
	dir := t.TempDir()
	secretPath := filepath.Join(dir, "secret.txt")
	require.NoError(t, os.WriteFile(secretPath, []byte("token-from-file\n"), 0o600))

	obj, err := benthosSASLObject(&SASLConfig{
		Mechanism:    "OAUTHBEARER",
		User:         "ignored",
		PasswordFile: secretPath,
	})
	require.NoError(t, err)

	assert.Equal(t, "OAUTHBEARER", obj["mechanism"])
	assert.Equal(t, "token-from-file", obj["token"])
	_, hasPassword := obj["password"]
	assert.False(t, hasPassword)
}

func TestKafkaTopicMetadataFallback(t *testing.T) {
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(nil))

	msg := service.NewMessage(nil)
	assert.Equal(t, unknownKafkaTopic, kafkaTopicMetadata(msg))

	msg.MetaSet("kafka_topic", "xatu-mainnet")
	assert.Equal(t, "xatu-mainnet", kafkaTopicMetadata(msg))
}

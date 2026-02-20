package kafka

import (
	"testing"

	"github.com/IBM/sarama"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func TestProducerConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  ProducerConfig
		wantErr string
	}{
		{
			name:    "missing brokers",
			config:  ProducerConfig{},
			wantErr: "brokers is required",
		},
		{
			name: "valid minimal config",
			config: ProducerConfig{
				Brokers: "localhost:9092",
			},
		},
		{
			name: "valid multi-broker config",
			config: ProducerConfig{
				Brokers: "broker1:9092,broker2:9092",
			},
		},
		{
			name: "TLS and SASL both set",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				TLSClientConfig: &TLSClientConfig{
					CertificatePath: "/cert",
					KeyPath:         "/key",
				},
				SASLConfig: &SASLConfig{
					User:     "user",
					Password: "pass",
				},
			},
			wantErr: "only one of 'tlsClientConfig' and 'sasl' can be specified",
		},
		{
			name: "TLS cert without key",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				TLSClientConfig: &TLSClientConfig{
					CertificatePath: "/cert",
				},
			},
			wantErr: "client key is required",
		},
		{
			name: "SASL missing user",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				SASLConfig: &SASLConfig{
					Password: "pass",
				},
			},
			wantErr: "'user' is required",
		},
		{
			name: "SASL password and passwordFile both set",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				SASLConfig: &SASLConfig{
					User:         "user",
					Password:     "pass",
					PasswordFile: "/file",
				},
			},
			wantErr: "either 'password' or 'passwordFile' can be specified",
		},
		{
			name: "SASL neither password nor passwordFile",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				SASLConfig: &SASLConfig{
					User: "user",
				},
			},
			wantErr: "'password' or 'passwordFile' is required",
		},
		{
			name: "valid SASL with password",
			config: ProducerConfig{
				Brokers: "localhost:9092",
				SASLConfig: &SASLConfig{
					User:     "user",
					Password: "pass",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestMarshalEvent(t *testing.T) {
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name: xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
			Id:   "test-id",
		},
	}

	tests := []struct {
		name     string
		encoding EncodingType
		validate func(t *testing.T, data []byte)
	}{
		{
			name:     "json encoding",
			encoding: EncodingTypeJSON,
			validate: func(t *testing.T, data []byte) {
				t.Helper()
				// Should be valid JSON - verify by round-tripping.
				var roundTrip xatu.DecoratedEvent

				err := protojson.Unmarshal(data, &roundTrip)
				require.NoError(t, err)
				assert.Equal(t,
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
					roundTrip.GetEvent().GetName(),
				)
			},
		},
		{
			name:     "protobuf encoding",
			encoding: EncodingTypeProtobuf,
			validate: func(t *testing.T, data []byte) {
				t.Helper()
				// Should be valid protobuf wire format.
				var roundTrip xatu.DecoratedEvent

				err := proto.Unmarshal(data, &roundTrip)
				require.NoError(t, err)
				assert.Equal(t,
					xatu.Event_BEACON_API_ETH_V1_EVENTS_BLOCK,
					roundTrip.GetEvent().GetName(),
				)
			},
		},
		{
			name:     "default encoding is json",
			encoding: "",
			validate: func(t *testing.T, data []byte) {
				t.Helper()

				var roundTrip xatu.DecoratedEvent

				err := protojson.Unmarshal(data, &roundTrip)
				require.NoError(t, err)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &ProducerConfig{Encoding: tt.encoding}

			data, err := cfg.MarshalEvent(event)
			require.NoError(t, err)
			assert.NotEmpty(t, data)

			tt.validate(t, data)
		})
	}
}

func TestInitSaramaConfig(t *testing.T) {
	t.Run("compression strategies", func(t *testing.T) {
		tests := []struct {
			strategy CompressionStrategy
			want     sarama.CompressionCodec
		}{
			{CompressionStrategyNone, sarama.CompressionNone},
			{CompressionStrategyGZIP, sarama.CompressionGZIP},
			{CompressionStrategySnappy, sarama.CompressionSnappy},
			{CompressionStrategyLZ4, sarama.CompressionLZ4},
			{CompressionStrategyZSTD, sarama.CompressionZSTD},
			{"unknown", sarama.CompressionNone},
		}

		for _, tt := range tests {
			t.Run(string(tt.strategy), func(t *testing.T) {
				cfg := &ProducerConfig{
					Brokers:     "localhost:9092",
					Compression: tt.strategy,
				}

				sc, err := InitSaramaConfig(cfg)
				require.NoError(t, err)
				assert.Equal(t, tt.want, sc.Producer.Compression)
			})
		}
	})

	t.Run("required acks", func(t *testing.T) {
		tests := []struct {
			acks RequiredAcks
			want sarama.RequiredAcks
		}{
			{RequiredAcksNone, sarama.NoResponse},
			{RequiredAcksAll, sarama.WaitForAll},
			{RequiredAcksLeader, sarama.WaitForLocal},
			{"unknown", sarama.WaitForLocal},
		}

		for _, tt := range tests {
			t.Run(string(tt.acks), func(t *testing.T) {
				cfg := &ProducerConfig{
					Brokers:      "localhost:9092",
					RequiredAcks: tt.acks,
				}

				sc, err := InitSaramaConfig(cfg)
				require.NoError(t, err)
				assert.Equal(t, tt.want, sc.Producer.RequiredAcks)
			})
		}
	})

	t.Run("partitioning strategies", func(t *testing.T) {
		tests := []struct {
			name     string
			strategy PartitionStrategy
		}{
			{"none uses hash", PartitionStrategyNone},
			{"random uses random", PartitionStrategyRandom},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				cfg := &ProducerConfig{
					Brokers:      "localhost:9092",
					Partitioning: tt.strategy,
				}

				sc, err := InitSaramaConfig(cfg)
				require.NoError(t, err)
				assert.NotNil(t, sc.Producer.Partitioner)

				// Verify the partitioner factory produces a valid
				// partitioner instance (types are unexported).
				p := sc.Producer.Partitioner("test-topic")
				assert.NotNil(t, p)
			})
		}
	})

	t.Run("producer settings propagated", func(t *testing.T) {
		cfg := &ProducerConfig{
			Brokers:        "localhost:9092",
			FlushBytes:     2_000_000,
			FlushMessages:  1000,
			FlushFrequency: 5_000_000_000, // 5s as Duration
			MaxRetries:     7,
		}

		sc, err := InitSaramaConfig(cfg)
		require.NoError(t, err)
		assert.Equal(t, 2_000_000, sc.Producer.Flush.Bytes)
		assert.Equal(t, 1000, sc.Producer.Flush.Messages)
		assert.Equal(t, cfg.FlushFrequency, sc.Producer.Flush.Frequency)
		assert.Equal(t, 7, sc.Producer.Retry.Max)
		assert.True(t, sc.Producer.Return.Successes)
		assert.False(t, sc.Metadata.Full)
	})

	t.Run("TLS enabled", func(t *testing.T) {
		cfg := &ProducerConfig{
			Brokers: "localhost:9092",
			TLS:     true,
		}

		sc, err := InitSaramaConfig(cfg)
		require.NoError(t, err)
		assert.True(t, sc.Net.TLS.Enable)
	})

	t.Run("kafka version parsed", func(t *testing.T) {
		cfg := &ProducerConfig{
			Brokers: "localhost:9092",
			Version: "2.8.0",
		}

		sc, err := InitSaramaConfig(cfg)
		require.NoError(t, err)
		assert.Equal(t, sarama.V2_8_0_0, sc.Version)
	})

	t.Run("invalid kafka version", func(t *testing.T) {
		cfg := &ProducerConfig{
			Brokers: "localhost:9092",
			Version: "not-a-version",
		}

		_, err := InitSaramaConfig(cfg)
		require.Error(t, err)
	})

	t.Run("SASL mechanisms", func(t *testing.T) {
		tests := []struct {
			mechanism SASLMechanism
			want      sarama.SASLMechanism
		}{
			{SASLTypeOAuth, sarama.SASLTypeOAuth},
			{SASLTypeSCRAMSHA256, sarama.SASLTypeSCRAMSHA256},
			{SASLTypeSCRAMSHA512, sarama.SASLTypeSCRAMSHA512},
			{SASLTypeGSSAPI, sarama.SASLTypeGSSAPI},
			{SASLTypePlaintext, sarama.SASLTypePlaintext},
			{"UNKNOWN", sarama.SASLTypePlaintext},
		}

		for _, tt := range tests {
			t.Run(string(tt.mechanism), func(t *testing.T) {
				cfg := &ProducerConfig{
					Brokers: "localhost:9092",
					SASLConfig: &SASLConfig{
						Mechanism: tt.mechanism,
						User:      "user",
						Password:  "pass",
					},
				}

				sc, err := InitSaramaConfig(cfg)
				require.NoError(t, err)
				assert.True(t, sc.Net.SASL.Enable)
				assert.Equal(t, tt.want, sc.Net.SASL.Mechanism)
				assert.Equal(t, "user", sc.Net.SASL.User)
				assert.Equal(t, "pass", sc.Net.SASL.Password)
			})
		}
	})
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr string
	}{
		{
			name:    "missing brokers",
			config:  Config{Topic: "test"},
			wantErr: "brokers is required",
		},
		{
			name: "missing topic",
			config: Config{
				ProducerConfig: ProducerConfig{
					Brokers: "localhost:9092",
				},
			},
			wantErr: "topic is required",
		},
		{
			name: "valid config",
			config: Config{
				ProducerConfig: ProducerConfig{
					Brokers: "localhost:9092",
				},
				Topic: "test-topic",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

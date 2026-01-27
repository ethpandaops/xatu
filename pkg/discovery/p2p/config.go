package p2p

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/ethereum"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Type Type `yaml:"type"`

	// Ethereum configuration.
	Ethereum *ethereum.Config `yaml:"ethereum"`

	Config *RawMessage `yaml:"config"`
}

// GetNetworkIDs extracts network IDs from the p2p config based on its type.
func (c *Config) GetNetworkIDs() []uint64 {
	if c.Config == nil {
		return nil
	}

	switch c.Type {
	case TypeXatu:
		conf := &xatu.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return nil
		}

		return conf.NetworkIds
	default:
		// Static and other types don't have network IDs configured
		return nil
	}
}

// GetExecutionConfig extracts execution layer config from the p2p config.
// Returns a config with default values if not configured or on error.
func (c *Config) GetExecutionConfig() *static.ExecutionConfig {
	defaultConfig := &static.ExecutionConfig{
		RetryAttempts: 5,
		RetryDelay:    5 * time.Second,
		DialTimeout:   15 * time.Second,
	}

	if c.Config == nil {
		return defaultConfig
	}

	switch c.Type {
	case TypeStatic:
		conf := &static.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return defaultConfig
		}

		return c.applyExecutionDefaults(&conf.Execution)
	case TypeXatu:
		conf := &xatu.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return defaultConfig
		}

		// Convert xatu.ExecutionConfig to static.ExecutionConfig
		return c.applyExecutionDefaults(&static.ExecutionConfig{
			RetryAttempts: conf.Execution.RetryAttempts,
			RetryDelay:    conf.Execution.RetryDelay,
			DialTimeout:   conf.Execution.DialTimeout,
		})
	default:
		return defaultConfig
	}
}

// applyExecutionDefaults applies default values for zero-valued fields.
func (c *Config) applyExecutionDefaults(exec *static.ExecutionConfig) *static.ExecutionConfig {
	if exec.RetryAttempts == 0 {
		exec.RetryAttempts = 5
	}

	if exec.RetryDelay == 0 {
		exec.RetryDelay = 5 * time.Second
	}

	if exec.DialTimeout == 0 {
		exec.DialTimeout = 15 * time.Second
	}

	return exec
}

// GetConsensusConfig extracts consensus layer config from the p2p config.
// Returns a config with default values if not configured or on error.
func (c *Config) GetConsensusConfig() *static.ConsensusConfig {
	defaultConfig := &static.ConsensusConfig{
		RetryAttempts:     1,
		RetryDelay:        2 * time.Second,
		DialTimeout:       5 * time.Second,
		DialConcurrency:   10,
		CooloffDuration:   10 * time.Second,
		ConnectionTimeout: 30 * time.Second,
	}

	if c.Config == nil {
		return defaultConfig
	}

	switch c.Type {
	case TypeStatic:
		conf := &static.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return defaultConfig
		}

		return c.applyConsensusDefaults(&conf.Consensus)
	case TypeXatu:
		conf := &xatu.Config{}
		if err := c.Config.Unmarshal(conf); err != nil {
			return defaultConfig
		}

		// Convert xatu.ConsensusConfig to static.ConsensusConfig
		return c.applyConsensusDefaults(&static.ConsensusConfig{
			RetryAttempts:     conf.Consensus.RetryAttempts,
			RetryDelay:        conf.Consensus.RetryDelay,
			DialTimeout:       conf.Consensus.DialTimeout,
			DialConcurrency:   conf.Consensus.DialConcurrency,
			CooloffDuration:   conf.Consensus.CooloffDuration,
			ConnectionTimeout: conf.Consensus.ConnectionTimeout,
		})
	default:
		return defaultConfig
	}
}

// applyConsensusDefaults applies default values for zero-valued fields.
func (c *Config) applyConsensusDefaults(cons *static.ConsensusConfig) *static.ConsensusConfig {
	if cons.RetryAttempts == 0 {
		cons.RetryAttempts = 1
	}

	if cons.RetryDelay == 0 {
		cons.RetryDelay = 2 * time.Second
	}

	if cons.DialTimeout == 0 {
		cons.DialTimeout = 5 * time.Second
	}

	if cons.DialConcurrency == 0 {
		cons.DialConcurrency = 10
	}

	if cons.CooloffDuration == 0 {
		cons.CooloffDuration = 10 * time.Second
	}

	if cons.ConnectionTimeout == 0 {
		cons.ConnectionTimeout = 30 * time.Second
	}

	return cons
}

func (c *Config) Validate() error {
	if c.Type == TypeUnknown {
		return errors.New("p2p type is required")
	}

	if c.Ethereum == nil {
		return errors.New("p2p ethereum config is required")
	}

	if err := c.Ethereum.Validate(); err != nil {
		return fmt.Errorf("p2p ethereum config error: %w", err)
	}

	return nil
}

func NewP2P(ctx context.Context, p2pType Type, config *RawMessage, handler func(ctx context.Context, node *enode.Node, source string) error, log logrus.FieldLogger) (P2P, error) {
	if p2pType == TypeUnknown {
		return nil, errors.New("p2p type is required")
	}

	switch p2pType {
	case TypeStatic:
		conf := &static.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return static.New(conf, handler, log)
	case TypeXatu:
		conf := &xatu.Config{}

		if err := config.Unmarshal(conf); err != nil {
			return nil, err
		}

		if err := defaults.Set(conf); err != nil {
			return nil, err
		}

		return xatu.New(ctx, conf, handler, log)
	default:
		return nil, fmt.Errorf("p2p type %s is unknown", p2pType)
	}
}

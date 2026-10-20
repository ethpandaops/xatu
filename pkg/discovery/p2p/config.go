package p2p

import (
	"context"
	"errors"
	"fmt"

	"github.com/creasty/defaults"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethpandaops/ethcore/pkg/ethereum"

	"github.com/ethpandaops/xatu/pkg/discovery/p2p/static"
	"github.com/ethpandaops/xatu/pkg/discovery/p2p/xatu"
	"github.com/ethpandaops/xatu/pkg/internal/configdecode"
	"github.com/ethpandaops/xatu/pkg/observability"
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
// Returns the defaults if not configured or on error.
func (c *Config) GetExecutionConfig() *static.ExecutionConfig {
	switch c.Type {
	case TypeStatic:
		conf := &static.Config{}
		if err := configdecode.Decode(c.Config, conf); err == nil {
			return &conf.Execution
		}
	case TypeXatu:
		conf := &xatu.Config{}
		if err := configdecode.Decode(c.Config, conf); err == nil {
			return &static.ExecutionConfig{
				RetryAttempts:   conf.Execution.RetryAttempts,
				RetryDelay:      conf.Execution.RetryDelay,
				DialTimeout:     conf.Execution.DialTimeout,
				BootstrapRPCURL: conf.Execution.BootstrapRPCURL,
			}
		}
	}

	exec := &static.ExecutionConfig{}
	defaults.MustSet(exec)

	return exec
}

// GetConsensusConfig extracts consensus layer config from the p2p config.
// Returns the defaults if not configured or on error.
func (c *Config) GetConsensusConfig() *static.ConsensusConfig {
	switch c.Type {
	case TypeStatic:
		conf := &static.Config{}
		if err := configdecode.Decode(c.Config, conf); err == nil {
			return &conf.Consensus
		}
	case TypeXatu:
		conf := &xatu.Config{}
		if err := configdecode.Decode(c.Config, conf); err == nil {
			return &static.ConsensusConfig{
				RetryAttempts:     conf.Consensus.RetryAttempts,
				RetryDelay:        conf.Consensus.RetryDelay,
				DialTimeout:       conf.Consensus.DialTimeout,
				DialConcurrency:   conf.Consensus.DialConcurrency,
				CooloffDuration:   conf.Consensus.CooloffDuration,
				ConnectionTimeout: conf.Consensus.ConnectionTimeout,
			}
		}
	}

	cons := &static.ConsensusConfig{}
	defaults.MustSet(cons)

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

func NewP2P(ctx context.Context, p2pType Type, config *RawMessage, handler func(ctx context.Context, node *enode.Node, source string) error, log observability.ContextualLogger) (P2P, error) {
	if p2pType == TypeUnknown {
		return nil, errors.New("p2p type is required")
	}

	switch p2pType {
	case TypeStatic:
		conf := &static.Config{}

		if err := configdecode.Decode(config, conf); err != nil {
			return nil, err
		}

		return static.New(conf, handler, log)
	case TypeXatu:
		conf := &xatu.Config{}

		if err := configdecode.Decode(config, conf); err != nil {
			return nil, err
		}

		return xatu.New(ctx, conf, handler, log)
	default:
		return nil, fmt.Errorf("p2p type %s is unknown", p2pType)
	}
}

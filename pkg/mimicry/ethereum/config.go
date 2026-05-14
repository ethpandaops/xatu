package ethereum

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/ethpandaops/xatu/pkg/observability"
)

type Config struct {
	// OverrideNetworkName is the name of the network to use for mimicry.
	// If not set, the network name will be automatically detected.
	OverrideNetworkName string `yaml:"overrideNetworkName" default:""`

	// BlobTransactionBatchSize is the number of blob transactions to request at a time
	// via GetPooledTransactions. Blob transactions are large (~128KB+), so this should be kept low.
	BlobTransactionBatchSize int `yaml:"blobTransactionBatchSize" default:"1"`

	// TransactionBatchSize is the number of non-blob transactions to request at a time
	// via GetPooledTransactions.
	TransactionBatchSize int `yaml:"transactionBatchSize" default:"10"`

	// BootstrapRPCURL is an optional execution JSON-RPC endpoint used to build
	// our own status message and serve lightweight block/header/body/receipt requests.
	BootstrapRPCURL string `yaml:"bootstrapRpcUrl" default:""`

	// PrivateKey is an optional secp256k1 private key used as the local devp2p
	// node identity for mimicry execution peer connections. If omitted, mimicry
	// generates one process-local identity and reuses it for all connections.
	PrivateKey string `yaml:"privateKey" default:""`
}

func (c *Config) Validate() error {
	if c.PrivateKey != "" {
		if _, err := crypto.HexToECDSA(strings.TrimPrefix(c.PrivateKey, "0x")); err != nil {
			return fmt.Errorf("invalid privateKey: %w", err)
		}
	}

	return nil
}

func (c *Config) ApplyOverrides(overrideNetworkName string, log observability.ContextualLogger) {
	if overrideNetworkName != "" {
		log.WithField("network", overrideNetworkName).Info("Overriding network name")
		c.OverrideNetworkName = overrideNetworkName
	}
}

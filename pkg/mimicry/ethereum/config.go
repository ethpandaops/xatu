package ethereum

import (
	"github.com/sirupsen/logrus"
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
	BootstrapRPCURL string `yaml:"bootstrapRPCURL" default:""` //nolint:tagliatelle // Preserve the documented acronym-heavy config key.

	// PrivateKey is an optional secp256k1 private key used as the local devp2p
	// node identity for mimicry execution peer connections. If omitted, mimicry
	// generates one process-local identity and reuses it for all connections.
	PrivateKey string `yaml:"privateKey" default:""`
}

func (c *Config) Validate() error {
	return nil
}

func (c *Config) ApplyOverrides(overrideNetworkName string, log logrus.FieldLogger) {
	if overrideNetworkName != "" {
		log.WithField("network", overrideNetworkName).Info("Overriding network name")
		c.OverrideNetworkName = overrideNetworkName
	}
}

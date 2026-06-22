package execution

import (
	"errors"

	"github.com/ethpandaops/xatu/pkg/cannon/execution/cryo"
)

// Config configures EL cannon: the execution RPC endpoint, the cryo runner,
// and per-dataset deriver settings. It maps to the top-level `execution:` block
// in cannon config.
type Config struct {
	// Enabled gates the entire EL cannon dimension.
	Enabled bool `yaml:"enabled" default:"false"`
	// RPCAddress is the execution-layer JSON-RPC endpoint cryo collects from.
	// Basic-auth credentials may be embedded (https://user:pass@host).
	RPCAddress string `yaml:"rpcAddress"`
	// Cryo configures the cryo runner.
	Cryo cryo.Config `yaml:"cryo"`
	// Block configures the canonical_execution_block deriver.
	Block BlockDeriverConfig `yaml:"block"`
	// Transaction configures the canonical_execution_transaction deriver.
	Transaction TransactionDeriverConfig `yaml:"transaction"`
	// Logs configures the canonical_execution_logs deriver.
	Logs LogsDeriverConfig `yaml:"logs"`
	// Traces configures the canonical_execution_traces deriver.
	Traces TracesDeriverConfig `yaml:"traces"`
	// NativeTransfers configures the canonical_execution_native_transfers deriver.
	NativeTransfers NativeTransfersDeriverConfig `yaml:"nativeTransfers"`
	// Erc20Transfers configures the canonical_execution_erc20_transfers deriver.
	Erc20Transfers Erc20TransfersDeriverConfig `yaml:"erc20Transfers"`
	// Erc721Transfers configures the canonical_execution_erc721_transfers deriver.
	Erc721Transfers Erc721TransfersDeriverConfig `yaml:"erc721Transfers"`
	// Contracts configures the canonical_execution_contracts deriver.
	Contracts ContractsDeriverConfig `yaml:"contracts"`
	// BalanceDiffs configures the canonical_execution_balance_diffs deriver.
	BalanceDiffs BalanceDiffsDeriverConfig `yaml:"balanceDiffs"`
	// StorageDiffs configures the canonical_execution_storage_diffs deriver.
	StorageDiffs StorageDiffsDeriverConfig `yaml:"storageDiffs"`
	// NonceDiffs configures the canonical_execution_nonce_diffs deriver.
	NonceDiffs NonceDiffsDeriverConfig `yaml:"nonceDiffs"`
	// BalanceReads configures the canonical_execution_balance_reads deriver.
	BalanceReads BalanceReadsDeriverConfig `yaml:"balanceReads"`
	// StorageReads configures the canonical_execution_storage_reads deriver.
	StorageReads StorageReadsDeriverConfig `yaml:"storageReads"`
	// NonceReads configures the canonical_execution_nonce_reads deriver.
	NonceReads NonceReadsDeriverConfig `yaml:"nonceReads"`
	// FourByteCounts configures the canonical_execution_four_byte_counts deriver.
	FourByteCounts FourByteCountsDeriverConfig `yaml:"fourByteCounts"`
	// AddressAppearances configures the canonical_execution_address_appearances deriver.
	AddressAppearances AddressAppearancesDeriverConfig `yaml:"addressAppearances"`
}

// Validate checks the execution config.
func (c *Config) Validate() error {
	if !c.Enabled {
		return nil
	}

	if c.RPCAddress == "" {
		return errors.New("execution.rpcAddress is required when execution is enabled")
	}

	if err := c.Cryo.Validate(); err != nil {
		return err
	}

	return nil
}

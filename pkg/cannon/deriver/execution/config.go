package execution

// Config groups the per-dataset EL (execution-layer) deriver configs. It maps to
// the `derivers.execution` block in cannon config. The shared EL connection
// lives in `ethereum.executionNodeAddress` and the cryo runner in the top-level
// `cryo` config — both are supplied to the derivers by the cannon wiring.
type Config struct {
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

// AnyEnabled reports whether at least one EL deriver is enabled. Cannon uses
// this to decide whether the execution node address + cryo config are required.
func (c *Config) AnyEnabled() bool {
	return c.Block.Enabled || c.Transaction.Enabled || c.Logs.Enabled ||
		c.Traces.Enabled || c.NativeTransfers.Enabled || c.Erc20Transfers.Enabled ||
		c.Erc721Transfers.Enabled || c.Contracts.Enabled || c.BalanceDiffs.Enabled ||
		c.StorageDiffs.Enabled || c.NonceDiffs.Enabled || c.BalanceReads.Enabled ||
		c.StorageReads.Enabled || c.NonceReads.Enabled || c.FourByteCounts.Enabled ||
		c.AddressAppearances.Enabled
}

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_withdrawal_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position_in_block` UInt32 COMMENT 'The index of the withdrawal within the block execution requests' CODEC(ZSTD(1)),
    `source_address` FixedString(42) COMMENT 'The source address that initiated the withdrawal request' CODEC(ZSTD(1)),
    `validator_pubkey` String COMMENT 'The public key of the validator the withdrawal targets' CODEC(ZSTD(1)),
    `amount` UInt64 COMMENT 'The withdrawal amount in gwei' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, block_root, position_in_block)
COMMENT 'Contains an EIP-7002 execution request withdrawal from a beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_withdrawal ON CLUSTER '{cluster}'
AS canonical_beacon_block_execution_request_withdrawal_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_request_withdrawal_local, cityHash64(slot_start_date_time, meta_network_name, block_root, position_in_block))
COMMENT 'Contains an EIP-7002 execution request withdrawal from a beacon block.';

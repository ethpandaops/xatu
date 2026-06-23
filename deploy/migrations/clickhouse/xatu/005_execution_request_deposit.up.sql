CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_deposit_local ON CLUSTER '{cluster}'
(
    `updated_date_time` DateTime COMMENT 'When this row was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `slot` UInt32 COMMENT 'The slot number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `slot_start_date_time` DateTime COMMENT 'The wall clock time when the slot started' CODEC(DoubleDelta, ZSTD(1)),
    `epoch` UInt32 COMMENT 'The epoch number from beacon block payload' CODEC(DoubleDelta, ZSTD(1)),
    `epoch_start_date_time` DateTime COMMENT 'The wall clock time when the epoch started' CODEC(DoubleDelta, ZSTD(1)),
    `block_root` FixedString(66) COMMENT 'The root hash of the beacon block' CODEC(ZSTD(1)),
    `block_version` LowCardinality(String) COMMENT 'The version of the beacon block',
    `position_in_block` UInt32 COMMENT 'The index of the deposit within the block execution requests' CODEC(ZSTD(1)),
    `pubkey` String COMMENT 'The public key of the validator from the deposit request' CODEC(ZSTD(1)),
    `withdrawal_credentials` String COMMENT 'The withdrawal credentials from the deposit request' CODEC(ZSTD(1)),
    `amount` UInt64 COMMENT 'The deposit amount in gwei' CODEC(ZSTD(1)),
    `signature` String COMMENT 'The deposit signature' CODEC(ZSTD(1)),
    `index` UInt64 COMMENT 'The deposit index from the deposit request' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
)
ENGINE = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY (meta_network_name, toYYYYMM(slot_start_date_time))
ORDER BY (meta_network_name, slot_start_date_time, block_root, position_in_block)
COMMENT 'Contains an EIP-6110 execution request deposit from a beacon block.';

CREATE TABLE IF NOT EXISTS canonical_beacon_block_execution_request_deposit ON CLUSTER '{cluster}'
AS canonical_beacon_block_execution_request_deposit_local
ENGINE = Distributed('{cluster}', currentDatabase(), canonical_beacon_block_execution_request_deposit_local, cityHash64(slot_start_date_time, meta_network_name, block_root, position_in_block))
COMMENT 'Contains an EIP-6110 execution request deposit from a beacon block.';

CREATE TABLE default.ethseer_validator_entity_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `event_date_time` DateTime64(3) COMMENT 'When the client fetched the beacon block from ethseer.io' CODEC(DoubleDelta, ZSTD(1)),
    `index` UInt32 COMMENT 'The index of the validator' CODEC(DoubleDelta, ZSTD(1)),
    `pubkey` String COMMENT 'The public key of the validator' CODEC(ZSTD(1)),
    `entity` String COMMENT 'The entity of the validator' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/{database}/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
)
ORDER BY
    (index, pubkey, meta_network_name) COMMENT 'Contains a mapping of validators to entities';

CREATE TABLE default.ethseer_validator_entity ON CLUSTER '{cluster}' AS default.ethseer_validator_entity_local ENGINE = Distributed(
    '{cluster}',
    default,
    ethseer_validator_entity_local,
    cityHash64(`index`, pubkey, meta_network_name)
);

CREATE TABLE default.blob_submitter_local ON CLUSTER '{cluster}' (
    `updated_date_time` DateTime COMMENT 'Timestamp when the record was last updated' CODEC(DoubleDelta, ZSTD(1)),
    `address` FixedString(66) COMMENT 'Ethereum address of the blob submitter' CODEC(ZSTD(1)),
    `name` String COMMENT 'Name of the blob submitter' CODEC(ZSTD(1)),
    `meta_network_name` LowCardinality(String) COMMENT 'Ethereum network name'
) ENGINE = ReplicatedReplacingMergeTree(
    '/clickhouse/{installation}/{cluster}/default/tables/{table}/{shard}',
    '{replica}',
    updated_date_time
)
ORDER BY
    (
        address,
        meta_network_name
    ) COMMENT 'Contains blob submitter address to name mappings.';

CREATE TABLE default.blob_submitter ON CLUSTER '{cluster}' AS default.blob_submitter_local ENGINE = Distributed(
    '{cluster}',
    default,
    blob_submitter_local,
    cityHash64(
        address,
        meta_network_name
    )
);

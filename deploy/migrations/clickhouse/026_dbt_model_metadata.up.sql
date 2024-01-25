CREATE TABLE dbt.model_metadata_local on cluster '{cluster}'
(
  model LowCardinality(String),
  updated_date_time DateTime Codec(ZSTD(1)),
  last_run_date_time DateTime Codec(ZSTD(1))
) Engine = ReplicatedReplacingMergeTree('/clickhouse/{installation}/{cluster}/tables/{shard}/{database}/{table}', '{replica}', updated_date_time)
PARTITION BY model
ORDER BY (model);

ALTER TABLE dbt.model_metadata_local ON CLUSTER '{cluster}'
MODIFY COMMENT 'Stores metadata about the dbt model run',
COMMENT COLUMN model 'Source of the data that was imported',
COMMENT COLUMN updated_date_time 'When this row was last updated',
COMMENT COLUMN last_run_date_time 'The end date the model was last run with';

CREATE TABLE dbt.model_metadata on cluster '{cluster}' AS dbt.model_metadata_local
ENGINE = Distributed('{cluster}', dbt, model_metadata_local, cityHash64(model));

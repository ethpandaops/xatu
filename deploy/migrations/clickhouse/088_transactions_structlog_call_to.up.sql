ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN gas_used UInt64 DEFAULT 0 COMMENT 'Actual gas consumed (computed from consecutive gas values)' CODEC(ZSTD(1)) AFTER gas_cost;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN gas_used UInt64 DEFAULT 0 COMMENT 'Actual gas consumed (computed from consecutive gas values)' CODEC(ZSTD(1)) AFTER gas_cost;

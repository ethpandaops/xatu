-- Drop canonical_execution_transaction_structlog_agg tables
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_agg ON CLUSTER '{cluster}';
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_agg_local ON CLUSTER '{cluster}';

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS task_count;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS task_count;

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS complete;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS complete;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN meta_network_id Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1));

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN meta_network_id Int32 COMMENT 'Ethereum network ID' CODEC(DoubleDelta, ZSTD(1));

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN program_counter UInt32 COMMENT 'The program counter' CODEC(Delta, ZSTD(1));

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN program_counter UInt32 COMMENT 'The program counter' CODEC(Delta, ZSTD(1));

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN call_frame_path;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN call_frame_path;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN gas_self;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN gas_self;

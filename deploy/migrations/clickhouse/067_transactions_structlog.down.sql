DROP TABLE IF EXISTS admin.execution_block ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS admin.execution_block_local ON CLUSTER '{cluster}' SYNC;

DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog ON CLUSTER '{cluster}' SYNC;
DROP TABLE IF EXISTS default.canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}' SYNC;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN gas_used;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN gas_used;

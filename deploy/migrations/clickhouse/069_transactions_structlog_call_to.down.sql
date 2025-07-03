ALTER TABLE canonical_execution_transaction_structlog on cluster '{cluster}'
    DROP COLUMN call_to_address;

ALTER TABLE canonical_execution_transaction_structlog_local on cluster '{cluster}'
    DROP COLUMN call_to_address;

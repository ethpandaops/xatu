ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_to_address Nullable(String) COMMENT 'Address of a CALL operation' CODEC(ZSTD(1)) AFTER error;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_to_address Nullable(String) COMMENT 'Address of a CALL operation' CODEC(ZSTD(1)) AFTER error;

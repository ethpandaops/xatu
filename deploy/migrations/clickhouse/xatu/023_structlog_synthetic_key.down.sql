ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    COMMENT COLUMN `call_to_address` 'Address of a CALL operation';

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `call_to_address` 'Address of a CALL operation';

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS synthetic;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    MODIFY ORDER BY (meta_network_name, block_number, transaction_hash, index);

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS synthetic;

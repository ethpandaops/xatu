ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    COMMENT COLUMN `return_data` 'The return data';

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `return_data` 'The return data';

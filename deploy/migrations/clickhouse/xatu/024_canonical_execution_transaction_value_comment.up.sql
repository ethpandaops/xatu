ALTER TABLE canonical_execution_transaction_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `value` 'The transaction value in wei';

ALTER TABLE canonical_execution_transaction ON CLUSTER '{cluster}'
    COMMENT COLUMN `value` 'The transaction value in wei';

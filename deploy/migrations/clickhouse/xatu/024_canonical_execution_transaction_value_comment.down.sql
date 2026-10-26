ALTER TABLE canonical_execution_transaction ON CLUSTER '{cluster}'
    COMMENT COLUMN `value` 'The transaction value in float64';

ALTER TABLE canonical_execution_transaction_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `value` 'The transaction value in float64';

ALTER TABLE canonical_execution_transaction_structlog_agg_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `memory_words_sum_before`,
    DROP COLUMN IF EXISTS `memory_words_sum_after`,
    DROP COLUMN IF EXISTS `memory_words_sq_sum_before`,
    DROP COLUMN IF EXISTS `memory_words_sq_sum_after`,
    DROP COLUMN IF EXISTS `memory_expansion_gas`,
    DROP COLUMN IF EXISTS `cold_access_count`;

ALTER TABLE canonical_execution_transaction_structlog_agg ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `memory_words_sum_before`,
    DROP COLUMN IF EXISTS `memory_words_sum_after`,
    DROP COLUMN IF EXISTS `memory_words_sq_sum_before`,
    DROP COLUMN IF EXISTS `memory_words_sq_sum_after`,
    DROP COLUMN IF EXISTS `memory_expansion_gas`,
    DROP COLUMN IF EXISTS `cold_access_count`;

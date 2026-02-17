-- Add resource gas building block columns for decomposing EVM gas into categories.
-- These columns enable downstream SQL to compute memory expansion gas and cold access gas
-- without needing per-opcode structlog data.

ALTER TABLE canonical_execution_transaction_structlog_agg_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `memory_words_sum_before` UInt64 DEFAULT 0 COMMENT 'SUM(ceil(memory_bytes/32)) before each opcode executes. Used with sq_sum to compute memory expansion gas.' CODEC(ZSTD(1)) AFTER `max_depth`,
    ADD COLUMN IF NOT EXISTS `memory_words_sum_after` UInt64 DEFAULT 0 COMMENT 'SUM(ceil(memory_bytes/32)) after each opcode executes.' CODEC(ZSTD(1)) AFTER `memory_words_sum_before`,
    ADD COLUMN IF NOT EXISTS `memory_words_sq_sum_before` UInt64 DEFAULT 0 COMMENT 'SUM(words_before²). With sum_before, enables exact memory gas via E[cost(after)] - E[cost(before)].' CODEC(ZSTD(1)) AFTER `memory_words_sum_after`,
    ADD COLUMN IF NOT EXISTS `memory_words_sq_sum_after` UInt64 DEFAULT 0 COMMENT 'SUM(words_after²). With sum_after, enables exact memory gas via E[cost(after)] - E[cost(before)].' CODEC(ZSTD(1)) AFTER `memory_words_sq_sum_before`,
    ADD COLUMN IF NOT EXISTS `memory_expansion_gas` UInt64 DEFAULT 0 COMMENT 'SUM(memory_expansion_gas). Exact per-opcode memory expansion cost, pre-computed to avoid intDiv rounding in SQL reconstruction.' CODEC(ZSTD(1)) AFTER `memory_words_sq_sum_after`,
    ADD COLUMN IF NOT EXISTS `cold_access_count` UInt64 DEFAULT 0 COMMENT 'Number of cold storage/account accesses (EIP-2929). cold_gas = cold_count * (cold_cost - warm_cost).' CODEC(ZSTD(1)) AFTER `memory_expansion_gas`;

ALTER TABLE canonical_execution_transaction_structlog_agg ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `memory_words_sum_before` UInt64 DEFAULT 0 COMMENT 'SUM(ceil(memory_bytes/32)) before each opcode executes.' AFTER `max_depth`,
    ADD COLUMN IF NOT EXISTS `memory_words_sum_after` UInt64 DEFAULT 0 COMMENT 'SUM(ceil(memory_bytes/32)) after each opcode executes.' AFTER `memory_words_sum_before`,
    ADD COLUMN IF NOT EXISTS `memory_words_sq_sum_before` UInt64 DEFAULT 0 COMMENT 'SUM(words_before²).' AFTER `memory_words_sum_after`,
    ADD COLUMN IF NOT EXISTS `memory_words_sq_sum_after` UInt64 DEFAULT 0 COMMENT 'SUM(words_after²).' AFTER `memory_words_sq_sum_before`,
    ADD COLUMN IF NOT EXISTS `memory_expansion_gas` UInt64 DEFAULT 0 COMMENT 'SUM(memory_expansion_gas). Exact per-opcode memory expansion cost.' AFTER `memory_words_sq_sum_after`,
    ADD COLUMN IF NOT EXISTS `cold_access_count` UInt64 DEFAULT 0 COMMENT 'Number of cold storage/account accesses (EIP-2929).' AFTER `memory_expansion_gas`;

ALTER TABLE default.canonical_execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `gas_limit` UInt64 COMMENT 'The block gas limit' CODEC(DoubleDelta, ZSTD(1)) AFTER `gas_used`;

ALTER TABLE default.canonical_execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `gas_limit` UInt64 COMMENT 'The block gas limit' CODEC(DoubleDelta, ZSTD(1)) AFTER `gas_used`;

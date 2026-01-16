ALTER TABLE default.canonical_execution_block ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `gas_limit`;

ALTER TABLE default.canonical_execution_block_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `gas_limit`;

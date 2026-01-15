-- Add execution client metadata columns to execution_engine_new_payload table
ALTER TABLE execution_engine_new_payload_local ON CLUSTER '{cluster}'
  ADD COLUMN IF NOT EXISTS meta_execution_implementation LowCardinality(String) COMMENT 'Implementation of the execution client (e.g., go-ethereum, reth, nethermind)',
  ADD COLUMN IF NOT EXISTS meta_execution_version LowCardinality(String) COMMENT 'Version of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_major LowCardinality(String) COMMENT 'Major version number of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_minor LowCardinality(String) COMMENT 'Minor version number of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_patch LowCardinality(String) COMMENT 'Patch version number of the execution client';

-- Add execution client metadata columns to execution_engine_get_blobs table
ALTER TABLE execution_engine_get_blobs_local ON CLUSTER '{cluster}'
  ADD COLUMN IF NOT EXISTS meta_execution_implementation LowCardinality(String) COMMENT 'Implementation of the execution client (e.g., go-ethereum, reth, nethermind)',
  ADD COLUMN IF NOT EXISTS meta_execution_version LowCardinality(String) COMMENT 'Version of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_major LowCardinality(String) COMMENT 'Major version number of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_minor LowCardinality(String) COMMENT 'Minor version number of the execution client',
  ADD COLUMN IF NOT EXISTS meta_execution_version_patch LowCardinality(String) COMMENT 'Patch version number of the execution client';

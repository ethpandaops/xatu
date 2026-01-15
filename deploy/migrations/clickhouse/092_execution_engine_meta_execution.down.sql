-- Remove execution client metadata columns from execution_engine_new_payload table
ALTER TABLE execution_engine_new_payload_local ON CLUSTER '{cluster}'
  DROP COLUMN IF EXISTS meta_execution_implementation,
  DROP COLUMN IF EXISTS meta_execution_version,
  DROP COLUMN IF EXISTS meta_execution_version_major,
  DROP COLUMN IF EXISTS meta_execution_version_minor,
  DROP COLUMN IF EXISTS meta_execution_version_patch;

-- Remove execution client metadata columns from execution_engine_get_blobs table
ALTER TABLE execution_engine_get_blobs_local ON CLUSTER '{cluster}'
  DROP COLUMN IF EXISTS meta_execution_implementation,
  DROP COLUMN IF EXISTS meta_execution_version,
  DROP COLUMN IF EXISTS meta_execution_version_major,
  DROP COLUMN IF EXISTS meta_execution_version_minor,
  DROP COLUMN IF EXISTS meta_execution_version_patch;

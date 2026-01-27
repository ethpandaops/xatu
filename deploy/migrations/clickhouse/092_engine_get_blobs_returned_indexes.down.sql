ALTER TABLE execution_engine_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS returned_blob_indexes;

ALTER TABLE execution_engine_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS returned_blob_indexes;

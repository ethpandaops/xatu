ALTER TABLE execution_engine_get_blobs_local ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS returned_blob_indexes Array(UInt8)
COMMENT 'Indexes (0-based) of the requested versioned_hashes that were successfully returned'
AFTER returned_count;

ALTER TABLE execution_engine_get_blobs ON CLUSTER '{cluster}'
ADD COLUMN IF NOT EXISTS returned_blob_indexes Array(UInt8)
COMMENT 'Indexes (0-based) of the requested versioned_hashes that were successfully returned'
AFTER returned_count;

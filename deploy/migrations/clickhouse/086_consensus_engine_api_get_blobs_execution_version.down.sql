-- Remove execution client version columns from consensus_engine_api_get_blobs table

ALTER TABLE default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_patch;

ALTER TABLE default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_minor;

ALTER TABLE default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_major;

ALTER TABLE default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_implementation;

ALTER TABLE default.consensus_engine_api_get_blobs ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version;

ALTER TABLE default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_patch;

ALTER TABLE default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_minor;

ALTER TABLE default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version_major;

ALTER TABLE default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_implementation;

ALTER TABLE default.consensus_engine_api_get_blobs_local ON CLUSTER '{cluster}'
DROP COLUMN meta_execution_version;

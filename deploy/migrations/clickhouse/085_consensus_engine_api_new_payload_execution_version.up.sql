-- Add execution client version columns to consensus_engine_api_new_payload table

ALTER TABLE default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version LowCardinality(String) DEFAULT '' COMMENT 'Full execution client version string from web3_clientVersion RPC' AFTER method_version;

ALTER TABLE default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_implementation LowCardinality(String) DEFAULT '' COMMENT 'Execution client implementation name (e.g., Geth, Nethermind, Besu, Reth, Erigon)' AFTER meta_execution_version;

ALTER TABLE default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_major LowCardinality(String) DEFAULT '' COMMENT 'Execution client major version number' AFTER meta_execution_implementation;

ALTER TABLE default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_minor LowCardinality(String) DEFAULT '' COMMENT 'Execution client minor version number' AFTER meta_execution_version_major;

ALTER TABLE default.consensus_engine_api_new_payload_local ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_patch LowCardinality(String) DEFAULT '' COMMENT 'Execution client patch version number' AFTER meta_execution_version_minor;

-- Add to distributed table
ALTER TABLE default.consensus_engine_api_new_payload ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version LowCardinality(String) DEFAULT '' COMMENT 'Full execution client version string from web3_clientVersion RPC' AFTER method_version;

ALTER TABLE default.consensus_engine_api_new_payload ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_implementation LowCardinality(String) DEFAULT '' COMMENT 'Execution client implementation name (e.g., Geth, Nethermind, Besu, Reth, Erigon)' AFTER meta_execution_version;

ALTER TABLE default.consensus_engine_api_new_payload ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_major LowCardinality(String) DEFAULT '' COMMENT 'Execution client major version number' AFTER meta_execution_implementation;

ALTER TABLE default.consensus_engine_api_new_payload ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_minor LowCardinality(String) DEFAULT '' COMMENT 'Execution client minor version number' AFTER meta_execution_version_major;

ALTER TABLE default.consensus_engine_api_new_payload ON CLUSTER '{cluster}'
ADD COLUMN meta_execution_version_patch LowCardinality(String) DEFAULT '' COMMENT 'Execution client patch version number' AFTER meta_execution_version_minor;

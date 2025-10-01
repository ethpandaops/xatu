-- Remove Direction column from distributed tables first
ALTER TABLE libp2p_handle_status ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;

ALTER TABLE libp2p_handle_metadata ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;

-- Remove Direction column from local tables
ALTER TABLE libp2p_handle_status_local ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;

ALTER TABLE libp2p_handle_metadata_local ON CLUSTER '{cluster}'
DROP COLUMN IF EXISTS direction;
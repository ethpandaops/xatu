ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

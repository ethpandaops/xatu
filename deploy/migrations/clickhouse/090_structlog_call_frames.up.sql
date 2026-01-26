ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN gas_self UInt64 DEFAULT 0 COMMENT 'Gas consumed by this opcode only, excludes child frame gas for CALL/CREATE opcodes. sum(gas_self) = total execution gas without double counting' CODEC(ZSTD(1)) AFTER gas_used;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN gas_self UInt64 DEFAULT 0 COMMENT 'Gas consumed by this opcode only, excludes child frame gas for CALL/CREATE opcodes. sum(gas_self) = total execution gas without double counting' CODEC(ZSTD(1)) AFTER gas_used;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_id UInt32 DEFAULT 0 COMMENT 'Sequential identifier for the call frame within the transaction' CODEC(DoubleDelta, ZSTD(1)) AFTER call_to_address;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN call_frame_path Array(UInt32) DEFAULT [0] COMMENT 'Path of frame IDs from root to current frame' CODEC(ZSTD(1)) AFTER call_frame_id;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN program_counter;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN program_counter;

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    DROP COLUMN meta_network_id;

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    DROP COLUMN meta_network_id;

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS complete UInt8;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS complete UInt8;

ALTER TABLE admin.execution_block_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS task_count UInt32;

ALTER TABLE admin.execution_block ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS task_count UInt32;

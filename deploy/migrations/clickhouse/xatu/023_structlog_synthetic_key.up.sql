-- MODIFY ORDER BY may only append a column added without a default in the same ALTER, which keeps this metadata-only.
ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS synthetic Bool
        COMMENT 'True for the row added after a CALL-family opcode whose target ran no code (an EOA or empty account, not a precompile). It has the CALL''s index, depth + 1, its own call_frame_id and an empty operation. Before execution-processor v0.4.0 (2026-10-20) these rows shared the CALL row''s key and one of the two was lost on merge' CODEC(ZSTD(1)),
    MODIFY ORDER BY (meta_network_name, block_number, transaction_hash, index, synthetic);

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS synthetic Bool
        COMMENT 'True for the row added after a CALL-family opcode whose target ran no code (an EOA or empty account, not a precompile). It has the CALL''s index, depth + 1, its own call_frame_id and an empty operation. Before execution-processor v0.4.0 (2026-10-20) these rows shared the CALL row''s key and one of the two was lost on merge' CODEC(ZSTD(1));

ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `call_to_address` 'Target address of a CALL, CALLCODE, DELEGATECALL or STATICCALL, or the address a CREATE or CREATE2 deployed (zero address if it failed). Before execution-processor v0.4.0 (2026-10-20) CREATE and CREATE2 rows from embedded (erigone) tracing had NULL';

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    COMMENT COLUMN `call_to_address` 'Target address of a CALL, CALLCODE, DELEGATECALL or STATICCALL, or the address a CREATE or CREATE2 deployed (zero address if it failed). Before execution-processor v0.4.0 (2026-10-20) CREATE and CREATE2 rows from embedded (erigone) tracing had NULL';

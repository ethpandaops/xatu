ALTER TABLE canonical_execution_transaction_structlog_local ON CLUSTER '{cluster}'
    COMMENT COLUMN `return_data` 'Return data buffer left by the CALL/CREATE-family opcode that just returned in this frame, set only on the first opcode after that call returns (NULL if empty). The buffer at any other opcode is the return_data of the latest such row in the same call_frame_id. Before execution-processor v0.3.0 (2026-10-20) the buffer was repeated on every opcode';

ALTER TABLE canonical_execution_transaction_structlog ON CLUSTER '{cluster}'
    COMMENT COLUMN `return_data` 'Return data buffer left by the CALL/CREATE-family opcode that just returned in this frame, set only on the first opcode after that call returns (NULL if empty). The buffer at any other opcode is the return_data of the latest such row in the same call_frame_id. Before execution-processor v0.3.0 (2026-10-20) the buffer was repeated on every opcode';

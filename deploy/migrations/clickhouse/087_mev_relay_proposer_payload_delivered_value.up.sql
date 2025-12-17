ALTER TABLE default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `value` UInt256 COMMENT 'The bid value in wei' CODEC(ZSTD(1)) AFTER `gas_used`;

ALTER TABLE default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}'
    ADD COLUMN IF NOT EXISTS `value` UInt256 COMMENT 'The bid value in wei' CODEC(ZSTD(1)) AFTER `gas_used`;

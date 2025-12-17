ALTER TABLE default.mev_relay_proposer_payload_delivered_local ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `value`;

ALTER TABLE default.mev_relay_proposer_payload_delivered ON CLUSTER '{cluster}'
    DROP COLUMN IF EXISTS `value`;

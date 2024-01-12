ALTER TABLE canonical_beacon_elaborated_attestation_local on cluster '{cluster}'
    ADD COLUMN inclusion_distance UInt16 CODEC(DoubleDelta, ZSTD(1)) AFTER position_in_block;

ALTER TABLE canonical_beacon_elaborated_attestation on cluster '{cluster}'
   ADD COLUMN inclusion_distance UInt16 CODEC(DoubleDelta, ZSTD(1)) AFTER position_in_block;
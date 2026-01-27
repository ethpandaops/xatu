-- ============================================================================
-- Horizon E2E Validation Queries
-- ============================================================================
-- This file contains SQL queries to validate that the Horizon module is
-- working correctly during E2E tests.
--
-- Usage:
--   cat scripts/e2e-horizon-validate.sql | clickhouse-client -h localhost
--
-- Or run individual queries:
--   docker exec xatu-clickhouse-01 clickhouse-client --query "<query>"
--
-- All queries filter by meta_client_module = 'HORIZON' to verify data
-- specifically came from the Horizon module (not Cannon or other sources).
-- ============================================================================


-- ============================================================================
-- QUERY 1: Count beacon blocks by slot (check for duplicates)
-- ============================================================================
-- Expected: Each slot should have at most 1 block (or 0 for missed slots).
-- If duplicates exist (cnt > 1), deduplication is not working properly.
-- Result should be empty if deduplication is working correctly.
-- ============================================================================
SELECT
    'DUPLICATE_BLOCKS' as check_name,
    slot,
    block_root,
    COUNT(*) as cnt
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
GROUP BY slot, block_root
HAVING cnt > 1
ORDER BY slot DESC
LIMIT 20;


-- ============================================================================
-- QUERY 2: Verify no gaps in slot sequence (FILL iterator working)
-- ============================================================================
-- Expected: No gaps greater than 1 slot between consecutive blocks.
-- Gaps of exactly 1 are normal (consecutive slots).
-- Large gaps (>1) indicate the FILL iterator may not be catching up properly.
-- Note: Some gaps may be acceptable if slots were missed (no block proposed).
-- ============================================================================
WITH slots AS (
    SELECT DISTINCT slot
    FROM beacon_api_eth_v2_beacon_block FINAL
    WHERE meta_client_module = 'HORIZON'
    ORDER BY slot
)
SELECT
    'SLOT_GAPS' as check_name,
    slot as current_slot,
    lagInFrame(slot, 1) OVER (ORDER BY slot) as previous_slot,
    slot - lagInFrame(slot, 1) OVER (ORDER BY slot) as gap
FROM slots
WHERE slot - lagInFrame(slot, 1) OVER (ORDER BY slot) > 1
  AND lagInFrame(slot, 1) OVER (ORDER BY slot) IS NOT NULL
ORDER BY slot
LIMIT 20;


-- ============================================================================
-- QUERY 3: Verify events have module_name = HORIZON
-- ============================================================================
-- Expected: All events should have meta_client_module = 'HORIZON'.
-- This query shows a sample of blocks to confirm the module name is set.
-- ============================================================================
SELECT
    'MODULE_VERIFICATION' as check_name,
    slot,
    block_root,
    meta_client_module,
    meta_client_name
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
ORDER BY slot DESC
LIMIT 10;


-- ============================================================================
-- QUERY 4: Count events per deriver type
-- ============================================================================
-- Expected: Non-zero counts for most deriver types if blocks were processed.
-- beacon_block: Should always have data
-- elaborated_attestation: Should have data (attestations in every block)
-- execution_transaction: May be 0 if no transactions in test blocks
-- attester_slashing, proposer_slashing: Often 0 (rare events)
-- deposit, withdrawal, voluntary_exit, bls_to_execution_change: May be 0
-- ============================================================================
SELECT
    'EVENTS_PER_DERIVER' as check_name,
    event_type,
    event_count
FROM (
    SELECT 'beacon_block' as event_type, COUNT(*) as event_count
    FROM beacon_api_eth_v2_beacon_block FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'attester_slashing', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_attester_slashing FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'proposer_slashing', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_proposer_slashing FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'deposit', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_deposit FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'withdrawal', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_withdrawal FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'voluntary_exit', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_voluntary_exit FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'bls_to_execution_change', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_bls_to_execution_change FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'execution_transaction', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_execution_transaction FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'elaborated_attestation', COUNT(*)
    FROM beacon_api_eth_v2_beacon_block_elaborated_attestation FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'proposer_duty', COUNT(*)
    FROM beacon_api_eth_v1_proposer_duty FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'beacon_blob', COUNT(*)
    FROM beacon_api_eth_v1_beacon_blob_sidecar FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'beacon_validators', COUNT(*)
    FROM beacon_api_eth_v1_beacon_validators FINAL
    WHERE meta_client_module = 'HORIZON'

    UNION ALL

    SELECT 'beacon_committee', COUNT(*)
    FROM beacon_api_eth_v1_beacon_committee FINAL
    WHERE meta_client_module = 'HORIZON'
)
ORDER BY event_count DESC;


-- ============================================================================
-- QUERY 5: Slot coverage summary
-- ============================================================================
-- Expected: Shows the range of slots covered and total unique slots.
-- coverage_percent: Should be close to 100% if no missed slots.
-- total_blocks: Should roughly equal (max_slot - min_slot + 1).
-- ============================================================================
SELECT
    'SLOT_COVERAGE' as check_name,
    MIN(slot) as min_slot,
    MAX(slot) as max_slot,
    MAX(slot) - MIN(slot) + 1 as expected_slots,
    COUNT(DISTINCT slot) as actual_slots,
    ROUND(COUNT(DISTINCT slot) * 100.0 / (MAX(slot) - MIN(slot) + 1), 2) as coverage_percent
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON';


-- ============================================================================
-- QUERY 6: Block latency analysis
-- ============================================================================
-- Expected: Shows how quickly Horizon processed blocks after they were produced.
-- Low latency indicates HEAD iterator is working in real-time.
-- Higher latency may indicate FILL iterator backfilling historical data.
-- ============================================================================
SELECT
    'BLOCK_LATENCY' as check_name,
    COUNT(*) as total_blocks,
    ROUND(AVG(toUnixTimestamp(meta_client_event_date_time) - toUnixTimestamp(slot_start_date_time)), 2) as avg_latency_seconds,
    MIN(toUnixTimestamp(meta_client_event_date_time) - toUnixTimestamp(slot_start_date_time)) as min_latency_seconds,
    MAX(toUnixTimestamp(meta_client_event_date_time) - toUnixTimestamp(slot_start_date_time)) as max_latency_seconds
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
  AND slot_start_date_time IS NOT NULL;


-- ============================================================================
-- QUERY 7: Events per beacon node (multi-node validation)
-- ============================================================================
-- Expected: If Horizon is connected to multiple beacon nodes, events should
-- still be deduplicated (total should match single-node processing).
-- This query shows which beacon node reported each block first.
-- ============================================================================
SELECT
    'EVENTS_BY_NODE' as check_name,
    meta_client_name,
    COUNT(*) as block_count
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
GROUP BY meta_client_name
ORDER BY block_count DESC;


-- ============================================================================
-- QUERY 8: Recent blocks (sanity check)
-- ============================================================================
-- Expected: Shows the 10 most recent blocks processed by Horizon.
-- Useful for quick visual verification that data is flowing.
-- ============================================================================
SELECT
    'RECENT_BLOCKS' as check_name,
    slot,
    LEFT(block_root, 16) as block_root_prefix,
    meta_client_name,
    meta_client_event_date_time
FROM beacon_api_eth_v2_beacon_block FINAL
WHERE meta_client_module = 'HORIZON'
ORDER BY slot DESC
LIMIT 10;


-- ============================================================================
-- VALIDATION SUMMARY
-- ============================================================================
-- This final query provides a pass/fail summary for automated testing.
-- All checks should return 1 (pass) for a successful E2E test.
-- ============================================================================
SELECT
    'VALIDATION_SUMMARY' as check_name,
    -- Check 1: Has beacon blocks
    (SELECT COUNT(*) > 0 FROM beacon_api_eth_v2_beacon_block FINAL WHERE meta_client_module = 'HORIZON') as has_beacon_blocks,
    -- Check 2: No duplicate blocks (by slot+block_root)
    (SELECT COUNT(*) = 0 FROM (
        SELECT slot, block_root, COUNT(*) as cnt
        FROM beacon_api_eth_v2_beacon_block FINAL
        WHERE meta_client_module = 'HORIZON'
        GROUP BY slot, block_root
        HAVING cnt > 1
    )) as no_duplicates,
    -- Check 3: Has elaborated attestations
    (SELECT COUNT(*) > 0 FROM beacon_api_eth_v2_beacon_block_elaborated_attestation FINAL WHERE meta_client_module = 'HORIZON') as has_attestations,
    -- Check 4: Has proposer duties
    (SELECT COUNT(*) > 0 FROM beacon_api_eth_v1_proposer_duty FINAL WHERE meta_client_module = 'HORIZON') as has_proposer_duties,
    -- Check 5: Has beacon committees
    (SELECT COUNT(*) > 0 FROM beacon_api_eth_v1_beacon_committee FINAL WHERE meta_client_module = 'HORIZON') as has_committees,
    -- Check 6: Reasonable slot coverage (>90%)
    (SELECT COUNT(DISTINCT slot) * 100.0 / (MAX(slot) - MIN(slot) + 1) > 90
     FROM beacon_api_eth_v2_beacon_block FINAL
     WHERE meta_client_module = 'HORIZON') as good_coverage;

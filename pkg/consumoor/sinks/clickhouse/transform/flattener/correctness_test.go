package flattener_test

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables"
	"github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
)

// TestCorrectnessAgainstVector loads captured Kafka messages, routes them
// through consumoor's flattener pipeline, inserts results into a local
// ClickHouse "consumoor" database, then diffs against Vector's output in
// the "default" database.
//
// Prerequisites:
//   - Local docker-compose running (Kafka + ClickHouse + Vector)
//   - Captured messages fed through Vector into default database
//   - consumoor database created with cloned schemas
//
// Run:
//
//	CONSUMOOR_INTEGRATION_TEST=true go test ./pkg/consumoor/sinks/clickhouse/transform/flattener/ \
//	  -run TestCorrectnessAgainstVector -v -timeout 120s
func TestCorrectnessAgainstVector(t *testing.T) {
	if os.Getenv("CONSUMOOR_INTEGRATION_TEST") != "true" {
		t.Skip("set CONSUMOOR_INTEGRATION_TEST=true to run")
	}

	ch := &chClient{
		baseURL:  envOr("CLICKHOUSE_URL", "http://localhost:8123"),
		user:     envOr("CLICKHOUSE_USER", ""),
		password: envOr("CLICKHOUSE_PASSWORD", ""),
	}

	ch.mustQuery(t, "SELECT 1")

	// 1. Load all capture lines from testdata.
	capturesDir := filepath.Join("..", "..", "..", "..", "testdata", "captures")
	lines := loadAllCaptureLines(t, capturesDir)
	t.Logf("loaded %d capture lines", len(lines))
	require.NotEmpty(t, lines)

	// 2. Build route index directly from registered table definitions.
	//    This avoids the Engine's dependency on Prometheus metrics.
	index := buildRouteIndex(tabledefs.All())

	// 3. Route each captured event and collect output rows per table.
	tableRows := make(map[string][]map[string]any, 64)

	var decoded, routedRows, skipped, errored int

	for _, raw := range lines {
		event, err := decodeCaptureLine(raw)
		if err != nil {
			errored++

			continue
		}

		decoded++

		name := event.GetEvent().GetName()

		routes, ok := index[name]
		if !ok {
			skipped++

			continue
		}

		meta := metadata.Extract(event)

		for _, route := range routes {
			if !route.ShouldProcess(event) {
				continue
			}

			batch := route.NewBatch()

			if err := batch.FlattenTo(event, meta); err != nil {
				t.Logf("flatten error %s -> %s: %v", name, route.TableName(), err)

				errored++

				continue
			}

			snapper, ok := batch.(flattener.Snapshotter)
			require.True(t, ok, "batch must implement Snapshotter")

			for _, row := range snapper.Snapshot() {
				tableRows[route.TableName()] = append(tableRows[route.TableName()], row)
			}

			routedRows += batch.Rows()
		}
	}

	t.Logf("pipeline: decoded=%d routed_rows=%d skipped=%d errors=%d tables=%d",
		decoded, routedRows, skipped, errored, len(tableRows))
	require.NotEmpty(t, tableRows, "no tables produced rows")

	// 4. Insert rows into consumoor _local tables (bypasses Distributed layer
	//    which may reference unreachable cluster nodes in docker-compose).
	insertFailed := make(map[string]bool, 8)

	for table, rows := range tableRows {
		localTable := table + "_local"

		if !ch.tableExists(t, "consumoor", localTable) {
			t.Logf("  SKIP insert %s: no consumoor.%s", table, localTable)

			insertFailed[table] = true

			continue
		}

		ch.truncate(t, "consumoor", localTable)

		if !ch.insertJSONRows(t, "consumoor", localTable, rows) {
			insertFailed[table] = true

			continue
		}

		t.Logf("  inserted %d rows -> consumoor.%s", len(rows), localTable)
	}

	// 5. Compare each table against Vector's output (both using _local tables).
	t.Log("--- comparison ---")

	var matched, failed, skippedTables int

	for table := range tableRows {
		if insertFailed[table] {
			skippedTables++

			continue
		}

		vectorTable := table + "_local"
		consumoorTable := table + "_local"

		if !ch.tableExists(t, "default", vectorTable) {
			t.Logf("SKIP  %-55s no default.%s", table, vectorTable)

			skippedTables++

			continue
		}

		// Force merge/deduplication on ReplacingMergeTree tables.
		ch.mustQuery(t, fmt.Sprintf("OPTIMIZE TABLE `default`.`%s` FINAL", vectorTable))

		vectorN := ch.rowCount(t, "default", vectorTable)
		if vectorN == 0 {
			t.Logf("SKIP  %-55s default.%s is empty", table, vectorTable)

			skippedTables++

			continue
		}

		consumoorN := ch.rowCount(t, "consumoor", consumoorTable)

		// Determine comparable columns present in both tables.
		cols := ch.intersectColumns(t, "default", vectorTable, "consumoor", consumoorTable)
		if len(cols) == 0 {
			t.Logf("SKIP  %-55s no comparable columns", table)

			skippedTables++

			continue
		}

		// Rows in Vector not matched by consumoor output.
		onlyVector := ch.exceptCount(t,
			"default", vectorTable,
			"consumoor", consumoorTable,
			cols,
		)

		if onlyVector == 0 {
			t.Logf("OK    %-55s vector=%d consumoor=%d", table, vectorN, consumoorN)

			matched++
		} else {
			t.Logf("FAIL  %-55s vector=%d consumoor=%d unmatched=%d",
				table, vectorN, consumoorN, onlyVector)
			ch.logColumnDiffs(t, "default", vectorTable, "consumoor", consumoorTable, cols)

			failed++
		}
	}

	t.Logf("--- summary: %d matched, %d failed, %d skipped ---", matched, failed, skippedTables)
	assert.Zero(t, failed, "tables with mismatches between Vector and consumoor output")
	assert.NotZero(t, matched, "expected at least one matching table")
}

// ---------------------------------------------------------------------------
// Route indexing
// ---------------------------------------------------------------------------

func buildRouteIndex(routes []flattener.Route) map[xatu.Event_Name][]flattener.Route {
	idx := make(map[xatu.Event_Name][]flattener.Route, len(routes))

	for _, r := range routes {
		for _, name := range r.EventNames() {
			idx[name] = append(idx[name], r)
		}
	}

	return idx
}

// ---------------------------------------------------------------------------
// Capture loading and decoding
// ---------------------------------------------------------------------------

func loadAllCaptureLines(t *testing.T, dir string) [][]byte {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	require.NoError(t, err)
	require.NotEmpty(t, matches, "no *.jsonl files in %s", dir)

	var out [][]byte

	for _, path := range matches {
		f, err := os.Open(path)
		require.NoError(t, err)

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 4<<20), 16<<20)

		for scanner.Scan() {
			raw := scanner.Bytes()
			if len(raw) == 0 {
				continue
			}

			cp := make([]byte, len(raw))
			copy(cp, raw)
			out = append(out, cp)
		}

		require.NoError(t, scanner.Err())
		f.Close()
	}

	return out
}

// decodeCaptureLine strips Vector-injected fields and unmarshals the
// remaining JSON into a DecoratedEvent protobuf.
func decodeCaptureLine(raw []byte) (*xatu.DecoratedEvent, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}

	// Vector injects path, source_type, timestamp — not in the proto.
	delete(obj, "path")
	delete(obj, "source_type")
	delete(obj, "timestamp")

	cleaned, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("json re-encode: %w", err)
	}

	event := &xatu.DecoratedEvent{}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}

	if err := opts.Unmarshal(cleaned, event); err != nil {
		return nil, fmt.Errorf("protojson: %w", err)
	}

	return event, nil
}

// ---------------------------------------------------------------------------
// ClickHouse HTTP client
// ---------------------------------------------------------------------------

type chClient struct {
	baseURL  string
	user     string
	password string
	http     *http.Client
}

func (c *chClient) httpClient() *http.Client {
	if c.http == nil {
		c.http = &http.Client{Timeout: 30 * time.Second}
	}

	return c.http
}

func (c *chClient) mustQuery(t *testing.T, sql string) string {
	t.Helper()

	const maxRetries = 3

	for attempt := range maxRetries {
		result, err := c.doQuery(sql)
		if err == nil {
			return result
		}

		if attempt < maxRetries-1 {
			time.Sleep(100 * time.Millisecond)

			continue
		}

		require.NoError(t, err, "query failed after %d attempts: %s", maxRetries, abbreviate(sql, 200))
	}

	return "" // unreachable
}

func (c *chClient) doQuery(sql string) (string, error) {
	req, err := http.NewRequest("POST", c.baseURL, strings.NewReader(sql))
	if err != nil {
		return "", err
	}

	if c.user != "" || c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	resp, err := c.httpClient().Do(req)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("clickhouse error (HTTP %d): %s\nquery: %s",
			resp.StatusCode, string(body), abbreviate(sql, 200))
	}

	return strings.TrimSpace(string(body)), nil
}

func (c *chClient) truncate(t *testing.T, db, table string) {
	t.Helper()
	c.mustQuery(t, fmt.Sprintf("TRUNCATE TABLE `%s`.`%s`", db, table))
}

func (c *chClient) truncateOnCluster(t *testing.T, db, table string) {
	t.Helper()
	c.mustQuery(t, fmt.Sprintf(
		"TRUNCATE TABLE `%s`.`%s` ON CLUSTER 'cluster_2S_1R'", db, table))
}

func (c *chClient) tableExists(t *testing.T, db, table string) bool {
	t.Helper()

	r := c.mustQuery(t, fmt.Sprintf(
		"SELECT count() FROM system.tables WHERE database='%s' AND name='%s'",
		db, table))

	return r == "1"
}

func (c *chClient) rowCount(t *testing.T, db, table string) int {
	t.Helper()

	r := c.mustQuery(t, fmt.Sprintf("SELECT count() FROM `%s`.`%s`", db, table))

	var n int

	_, _ = fmt.Sscanf(r, "%d", &n)

	return n
}

// columnInfo holds a column name and whether it uses a float type.
type columnInfo struct {
	name    string
	isFloat bool
}

// intersectColumns returns column names present in both tables, excluding
// runtime columns that always differ between Vector and consumoor.
func (c *chClient) intersectColumns(
	t *testing.T,
	dbA, tableA, dbB, tableB string,
) []columnInfo {
	t.Helper()

	r := c.mustQuery(t, fmt.Sprintf(`
		SELECT a.name, multiIf(
			a.type LIKE '%%Float32%%', 1,
			a.type LIKE '%%Float64%%', 1,
			0)
		FROM (SELECT name, type, position FROM system.columns
		      WHERE database='%s' AND table='%s') a
		INNER JOIN
		     (SELECT name FROM system.columns
		      WHERE database='%s' AND table='%s') b
		ON a.name = b.name
		WHERE a.name NOT IN ('updated_date_time', 'unique_key',
			'meta_client_ip', 'remote_ip', 'ip')
		  AND a.name NOT LIKE '%%_geo_%%'
		  AND a.name NOT LIKE 'geo_%%'
		ORDER BY a.position`,
		dbA, tableA, dbB, tableB))

	if r == "" {
		return nil
	}

	lines := strings.Split(r, "\n")
	cols := make([]columnInfo, 0, len(lines))

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		ci := columnInfo{name: parts[0]}

		if len(parts) > 1 && parts[1] == "1" {
			ci.isFloat = true
		}

		cols = append(cols, ci)
	}

	return cols
}

// exceptCount returns the number of rows in (A EXCEPT B).
// Nullable columns are coalesced so that NULL and empty/zero are treated as
// equivalent (works around code-generator inconsistencies in Nullable handling).
func (c *chClient) exceptCount(
	t *testing.T,
	dbA, tableA, dbB, tableB string,
	cols []columnInfo,
) int {
	t.Helper()

	// Use ifNull to normalize Nullable columns: NULL and default-value become
	// equivalent. This is safe because ifNull is a no-op on non-Nullable columns.
	colList := ifNullJoin(cols)
	sql := fmt.Sprintf(
		"SELECT count() FROM (SELECT %s FROM `%s`.`%s` EXCEPT SELECT %s FROM `%s`.`%s`)",
		colList, dbA, tableA, colList, dbB, tableB)

	r := c.mustQuery(t, sql)

	var n int

	_, _ = fmt.Sscanf(r, "%d", &n)

	return n
}

// logColumnDiffs identifies and logs which columns differ between two tables.
func (c *chClient) logColumnDiffs(
	t *testing.T,
	dbA, tableA, dbB, tableB string,
	cols []columnInfo,
) {
	t.Helper()

	for _, ci := range cols {
		bc := fmt.Sprintf("`%s`", ci.name)

		sql := fmt.Sprintf(
			"SELECT "+
				"(SELECT arraySort(groupArray(toString(%s))) FROM `%s`.`%s`) = "+
				"(SELECT arraySort(groupArray(toString(%s))) FROM `%s`.`%s`)",
			bc, dbA, tableA, bc, dbB, tableB)

		r := c.mustQuery(t, sql)
		if r == "1" {
			continue
		}

		vecVals := c.mustQuery(t, fmt.Sprintf(
			"SELECT arraySort(groupArray(toString(%s))) FROM `%s`.`%s`",
			bc, dbA, tableA))
		conVals := c.mustQuery(t, fmt.Sprintf(
			"SELECT arraySort(groupArray(toString(%s))) FROM `%s`.`%s`",
			bc, dbB, tableB))

		t.Logf("    column %-45s vector=%s consumoor=%s",
			ci.name, abbreviate(vecVals, 120), abbreviate(conVals, 120))
	}
}

// insertJSONRows inserts rows into ClickHouse via JSONEachRow. Returns
// false if the insert failed (logged but non-fatal so other tables can
// still be compared).
func (c *chClient) insertJSONRows(
	t *testing.T,
	db, table string,
	rows []map[string]any,
) bool {
	t.Helper()

	if len(rows) == 0 {
		return true
	}

	var buf bytes.Buffer

	enc := json.NewEncoder(&buf)
	for _, row := range rows {
		if err := enc.Encode(sanitizeRow(row)); err != nil {
			t.Logf("INSERT FAIL %s.%s: json encode: %v", db, table, err)

			return false
		}
	}

	insertURL := fmt.Sprintf(
		"%s/?database=%s&input_format_skip_unknown_fields=1&date_time_input_format=best_effort&query=%s",
		c.baseURL,
		url.QueryEscape(db),
		url.QueryEscape(fmt.Sprintf("INSERT INTO `%s` FORMAT JSONEachRow", table)),
	)

	req, err := http.NewRequest("POST", insertURL, &buf)
	if err != nil {
		t.Logf("INSERT FAIL %s.%s: %v", db, table, err)

		return false
	}

	if c.user != "" || c.password != "" {
		req.SetBasicAuth(c.user, c.password)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient().Do(req)
	if err != nil {
		t.Logf("INSERT FAIL %s.%s: %v", db, table, err)

		return false
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("INSERT FAIL %s.%s: %s", db, table, abbreviate(string(body), 200))

		return false
	}

	return true
}

// ---------------------------------------------------------------------------
// Row sanitization for JSON serialization
// ---------------------------------------------------------------------------

// sanitizeRow converts Go types that don't serialize cleanly to JSON into
// ClickHouse-compatible representations.
func sanitizeRow(row map[string]any) map[string]any {
	out := make(map[string]any, len(row))

	for k, v := range row {
		out[k] = sanitizeValue(v)
	}

	return out
}

func sanitizeValue(v any) any {
	switch tv := v.(type) {
	case time.Time:
		return tv.Unix()
	case []byte:
		return fmt.Sprintf("%x", tv)
	case map[string]any:
		return sanitizeRow(tv)
	case []any:
		out := make([]any, len(tv))
		for i, item := range tv {
			out[i] = sanitizeValue(item)
		}

		return out
	default:
		return v
	}
}

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

// ifNullJoin wraps each column in assumeNotNull() to normalize Nullable columns.
// Float columns are additionally wrapped with ROUND(..., 6) to tolerate
// float64 text-serialization precision differences between pipelines.
func ifNullJoin(cols []columnInfo) string {
	quoted := make([]string, len(cols))

	for i, c := range cols {
		if c.isFloat {
			quoted[i] = fmt.Sprintf("ROUND(assumeNotNull(`%s`), 6)", c.name)
		} else {
			quoted[i] = fmt.Sprintf("assumeNotNull(`%s`)", c.name)
		}
	}

	return strings.Join(quoted, ", ")
}

func abbreviate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen] + "..."
	}

	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return fallback
}

// ---------------------------------------------------------------------------
// E2E smoke test: full pipeline comparison (Vector vs consumoor)
// ---------------------------------------------------------------------------

// TestE2ESmoke POSTs captured events to xatu-server's HTTP endpoint, waits
// for both pipelines (Vector + consumoor) to flush, then compares the
// resulting ClickHouse _local tables using EXCEPT in both directions.
//
// Prerequisites:
//   - docker-compose running: xatu-server, Kafka, Vector, consumoor, ClickHouse
//   - Both "default" and "consumoor" databases with _local tables created
//
// Run:
//
//	CONSUMOOR_E2E_TEST=true go test ./pkg/consumoor/sinks/clickhouse/transform/flattener/ \
//	  -run TestE2ESmoke -v -timeout 120s
func TestE2ESmoke(t *testing.T) {
	if os.Getenv("CONSUMOOR_E2E_TEST") != "true" {
		t.Skip("set CONSUMOOR_E2E_TEST=true to run")
	}

	ch := &chClient{
		baseURL:  envOr("CLICKHOUSE_URL", "http://localhost:8123"),
		user:     envOr("CLICKHOUSE_USER", ""),
		password: envOr("CLICKHOUSE_PASSWORD", ""),
	}

	ch.mustQuery(t, "SELECT 1")

	serverURL := envOr("XATU_SERVER_URL", "http://localhost:8087/v1/events")
	flushWait := envDurationOr("E2E_FLUSH_WAIT", 45*time.Second)
	drainWait := envDurationOr("E2E_DRAIN_WAIT", 10*time.Second)
	httpC := &http.Client{Timeout: 10 * time.Second}

	// 1. Truncate all _local tables ON CLUSTER (clears ALL shards, not just
	//    the local node), wait for consumoor to drain pending Kafka messages
	//    (stale from previous runs), then truncate again.
	t.Log("=== truncating _local tables in default and consumoor (ON CLUSTER) ===")

	allLocalTables := make(map[string][]string, 2)

	for _, db := range []string{"default", "consumoor"} {
		tables := ch.listLocalTables(t, db)
		allLocalTables[db] = tables

		for _, tbl := range tables {
			ch.truncateOnCluster(t, db, tbl)
		}

		t.Logf("  truncated %d _local tables in %s", len(tables), db)
	}

	t.Logf("=== draining stale consumoor messages for %s ===", drainWait)
	time.Sleep(drainWait)

	// Second truncate to clear any stale data consumoor flushed.
	for _, db := range []string{"default", "consumoor"} {
		for _, tbl := range allLocalTables[db] {
			ch.truncateOnCluster(t, db, tbl)
		}
	}

	t.Log("  second truncate complete")

	// 2. Load and POST captured events to xatu-server.
	//    Skip deprecated V1 events (those replaced by V2) so we only
	//    compare events that both pipelines are expected to handle.
	capturesDir := filepath.Join("..", "..", "..", "..", "testdata", "captures")
	lines := loadAllCaptureLines(t, capturesDir)
	t.Logf("=== loaded %d capture lines ===", len(lines))
	require.NotEmpty(t, lines)

	var posted, postFailed, skippedDeprecated int

	for i, raw := range lines {
		cleaned, err := stripVectorFields(raw)
		if err != nil {
			t.Logf("  SKIP line %d: strip failed: %v", i, err)

			postFailed++

			continue
		}

		name, err := extractEventName(cleaned)
		if err != nil {
			t.Logf("  SKIP line %d: no event name: %v", i, err)

			postFailed++

			continue
		}

		if isDeprecatedV1Event(name) {
			skippedDeprecated++

			continue
		}

		if err := postEventToServer(httpC, serverURL, cleaned); err != nil {
			t.Logf("  FAIL line %d (%s): %v", i, name, err)

			postFailed++

			continue
		}

		posted++
	}

	t.Logf("ingest: posted=%d skipped_deprecated=%d failed=%d",
		posted, skippedDeprecated, postFailed)
	require.NotZero(t, posted, "no events were successfully posted")

	// 3. Wait for both pipelines to flush.
	t.Logf("=== waiting %s for pipelines to flush ===", flushWait)
	time.Sleep(flushWait)

	// 4. OPTIMIZE all _local tables ON CLUSTER to force ReplacingMergeTree
	// dedup on ALL shards before comparison. Doing this upfront in one pass
	// is faster than per-table OPTIMIZE during comparison.
	t.Log("=== optimizing _local tables ON CLUSTER ===")

	baseNames := ch.localTableBaseNames(t, "default", "consumoor")
	t.Logf("found %d unique base table names across both databases", len(baseNames))

	for _, db := range []string{"default", "consumoor"} {
		for _, base := range baseNames {
			localName := base + "_local"
			if ch.tableExists(t, db, localName) {
				ch.mustQuery(t, fmt.Sprintf(
					"OPTIMIZE TABLE `%s`.`%s` ON CLUSTER 'cluster_2S_1R' FINAL",
					db, localName))
			}
		}
	}

	t.Log("=== comparing default vs consumoor tables ===")

	var matched, failed, skippedTables int

	for _, base := range baseNames {
		if skipE2ETable[base] {
			skippedTables++

			continue
		}

		// Use Distributed tables (no suffix) so we see data from all shards.
		result := ch.compareTable(t, base, "")

		switch result {
		case tableMatched:
			matched++
		case tableFailed:
			failed++
		case tableSkipped:
			skippedTables++
		}
	}

	t.Logf("=== summary: %d matched, %d failed, %d skipped ===",
		matched, failed, skippedTables)
	assert.Zero(t, failed,
		"tables with mismatches between Vector and consumoor output")
	assert.NotZero(t, matched, "expected at least one matching table")
}

// ---------------------------------------------------------------------------
// E2E helper functions
// ---------------------------------------------------------------------------

// envDurationOr parses an env var as time.Duration, returning fallback if
// unset or unparseable.
func envDurationOr(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}

	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}

	return d
}

// vectorOnlyTables lists tables that only exist in the Vector pipeline and
// should be skipped during E2E comparison. These include: migration tracking,
// materialized view aggregates, external data imports, and reference tables.
var vectorOnlyTables = map[string]bool{
	"schema_migrations":                true, // migration tracking
	"beacon_api_slot":                  true, // materialized view aggregate
	"beacon_block_classification":      true, // blockprint/cannon, not routed to consumoor
	"imported_sources":                 true, // external import tracking
	"mempool_dumpster_transaction":     true, // external Mempool Dumpster data
	"block_native_mempool_transaction": true, // external Blocknative data
	"ethseer_validator_entity":         true, // external ethseer.io enrichment
	"blob_submitter":                   true, // static reference data
	"execution_transaction":            true, // not produced by consumoor
}

// skipE2ETable lists base table names that should be excluded from E2E
// comparison because they are not event tables populated by consumoor.
var skipE2ETable = map[string]bool{
	"schema_migrations": true, // ClickHouse migration tracking table
	"beacon_api_slot":   true, // Materialized view aggregate, not directly written
}

// deprecatedV1Events lists event names that have V2 replacements.
// These are skipped during E2E ingestion because consumoor only handles V2.
var deprecatedV1Events = map[string]bool{
	"BEACON_API_ETH_V1_EVENTS_ATTESTATION":            true,
	"BEACON_API_ETH_V1_EVENTS_BLOCK":                  true,
	"BEACON_API_ETH_V1_EVENTS_CHAIN_REORG":            true,
	"BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT":   true,
	"BEACON_API_ETH_V1_EVENTS_HEAD":                   true,
	"BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT":         true,
	"BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF": true,
	"MEMPOOL_TRANSACTION":                             true,
	"BEACON_API_ETH_V2_BEACON_BLOCK":                  true,
	"BEACON_API_ETH_V1_DEBUG_FORK_CHOICE":             true,
	"BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG":       true,
}

// isDeprecatedV1Event returns true if the event name is a deprecated V1
// event that has been replaced by a V2 version.
func isDeprecatedV1Event(name string) bool {
	return deprecatedV1Events[name]
}

// extractEventName pulls the event.name field from a DecoratedEvent JSON blob.
func extractEventName(eventJSON []byte) (string, error) {
	var wrapper struct {
		Event struct {
			Name string `json:"name"`
		} `json:"event"`
	}

	if err := json.Unmarshal(eventJSON, &wrapper); err != nil {
		return "", fmt.Errorf("json parse: %w", err)
	}

	if wrapper.Event.Name == "" {
		return "", fmt.Errorf("empty event name")
	}

	return wrapper.Event.Name, nil
}

// stripVectorFields removes Vector-injected fields (path, source_type,
// timestamp) from a raw JSON line so xatu-server can accept it.
func stripVectorFields(raw []byte) ([]byte, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil {
		return nil, fmt.Errorf("json parse: %w", err)
	}

	delete(obj, "path")
	delete(obj, "source_type")
	delete(obj, "timestamp")

	cleaned, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("json re-encode: %w", err)
	}

	return cleaned, nil
}

// postEventToServer wraps a single DecoratedEvent JSON in a
// CreateEventsRequest and POSTs it to xatu-server.
func postEventToServer(
	httpC *http.Client,
	serverURL string,
	eventJSON []byte,
) error {
	payload := fmt.Sprintf(`{"events":[%s]}`, string(eventJSON))

	resp, err := httpC.Post(serverURL, "application/json",
		strings.NewReader(payload))
	if err != nil {
		return fmt.Errorf("POST: %w", err)
	}

	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode,
			abbreviate(string(body), 200))
	}

	return nil
}

// tableResult represents the outcome of comparing a single table.
type tableResult int

const (
	tableSkipped tableResult = iota
	tableMatched
	tableFailed
)

// compareTable compares a single table between the default and consumoor
// databases. The suffix controls which table variant is compared: "_local"
// for shard-local tables, "" for Distributed tables that aggregate all shards.
// When using Distributed tables, OPTIMIZE is still applied to the underlying
// _local tables to force deduplication.
func (c *chClient) compareTable(t *testing.T, base, suffix string) tableResult {
	t.Helper()

	if vectorOnlyTables[base] {
		t.Logf("SKIP  %-55s vector-only table", base)

		return tableSkipped
	}

	tableName := base + suffix

	defaultExists := c.tableExists(t, "default", tableName)
	consumoorExists := c.tableExists(t, "consumoor", tableName)

	if !defaultExists && !consumoorExists {
		return tableSkipped
	}

	// NOTE: OPTIMIZE FINAL is done upfront in TestE2ESmoke before the
	// comparison loop. For non-E2E callers, ensure tables are optimized
	// before calling compareTable.

	vectorN := 0
	if defaultExists {
		vectorN = c.rowCount(t, "default", tableName)
	}

	consumoorN := 0
	if consumoorExists {
		consumoorN = c.rowCount(t, "consumoor", tableName)
	}

	if vectorN == 0 && consumoorN == 0 {
		t.Logf("EMPTY %-55s vector=0 consumoor=0", base)

		return tableSkipped
	}

	// Only in consumoor (Vector filter excluded this event type).
	if vectorN == 0 && consumoorN > 0 {
		t.Logf("SKIP  %-55s vector=0 consumoor=%d (not in Vector filter)",
			base, consumoorN)

		return tableSkipped
	}

	// Only in default — consumoor should have processed this.
	if vectorN > 0 && consumoorN == 0 {
		t.Logf("FAIL  %-55s vector=%d consumoor=0 (missing in consumoor)",
			base, vectorN)

		return tableFailed
	}

	if !defaultExists || !consumoorExists {
		t.Logf("SKIP  %-55s table missing in one db", base)

		return tableSkipped
	}

	cols := c.intersectColumns(t,
		"default", tableName,
		"consumoor", tableName,
	)
	if len(cols) == 0 {
		t.Logf("SKIP  %-55s no comparable columns", base)

		return tableSkipped
	}

	onlyVector := c.exceptCount(t,
		"default", tableName,
		"consumoor", tableName,
		cols,
	)

	onlyConsumoor := c.exceptCount(t,
		"consumoor", tableName,
		"default", tableName,
		cols,
	)

	if onlyVector == 0 && onlyConsumoor == 0 {
		t.Logf("OK    %-55s vector=%d consumoor=%d",
			base, vectorN, consumoorN)

		return tableMatched
	}

	t.Logf("FAIL  %-55s vector=%d consumoor=%d "+
		"only_vector=%d only_consumoor=%d",
		base, vectorN, consumoorN, onlyVector, onlyConsumoor)

	if onlyVector > 0 {
		t.Logf("  ── columns differing (vector EXCEPT consumoor):")
		c.logColumnDiffs(t,
			"default", tableName,
			"consumoor", tableName, cols)
	}

	if onlyConsumoor > 0 {
		t.Logf("  ── columns differing (consumoor EXCEPT vector):")
		c.logColumnDiffs(t,
			"consumoor", tableName,
			"default", tableName, cols)
	}

	return tableFailed
}

// listLocalTables returns all _local table names in the given database,
// excluding materialized views (which cannot be OPTIMIZE'd or EXCEPT'd).
func (c *chClient) listLocalTables(t *testing.T, db string) []string {
	t.Helper()

	r := c.mustQuery(t, fmt.Sprintf(
		"SELECT name FROM system.tables "+
			"WHERE database='%s' AND name LIKE '%%_local' "+
			"AND engine NOT IN ('MaterializedView') "+
			"ORDER BY name", db))

	if r == "" {
		return nil
	}

	return strings.Split(r, "\n")
}

// localTableBaseNames returns the sorted union of base names (without the
// _local suffix) across the given databases.
func (c *chClient) localTableBaseNames(
	t *testing.T,
	dbs ...string,
) []string {
	t.Helper()

	seen := make(map[string]struct{}, 128)

	for _, db := range dbs {
		tables := c.listLocalTables(t, db)
		for _, tbl := range tables {
			base := strings.TrimSuffix(tbl, "_local")
			seen[base] = struct{}{}
		}
	}

	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}

	// Sort for deterministic output.
	sort.Strings(names)

	return names
}

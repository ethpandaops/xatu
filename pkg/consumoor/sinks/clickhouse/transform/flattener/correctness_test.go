package flattener_test

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/sinks/clickhouse/transform/flattener/tables"
)

// TestStagingCorrectness compares consumoor's local ClickHouse output against
// a staging (or production) ClickHouse instance where Vector already writes.
//
// Prerequisites:
//   - Local ClickHouse running (via docker-compose staging overlay) with consumoor data
//   - Staging ClickHouse port-forwarded (kubectl port-forward svc/chendpoint-xatu-clickhouse 8124:8123)
//   - Consumoor has consumed from staging Kafka and written to local CH
//
// Run:
//
//	CONSUMOOR_STAGING_TEST=true \
//	CLICKHOUSE_URL=http://localhost:8123 \
//	STAGING_CLICKHOUSE_URL=http://localhost:8124 \
//	go test ./pkg/consumoor/sinks/clickhouse/transform/flattener/ \
//	  -run TestStagingCorrectness -v -timeout 300s
func TestStagingCorrectness(t *testing.T) {
	if os.Getenv("CONSUMOOR_STAGING_TEST") != "true" {
		t.Skip("set CONSUMOOR_STAGING_TEST=true to run")
	}

	maxRows, err := strconv.Atoi(envOr("MAX_COMPARE_ROWS", "10000"))
	require.NoError(t, err)

	localCH := &chClient{
		baseURL:  envOr("CLICKHOUSE_URL", "http://localhost:8123"),
		user:     envOr("CLICKHOUSE_USER", ""),
		password: envOr("CLICKHOUSE_PASSWORD", ""),
	}

	stagingCH := &chClient{
		baseURL:  envOr("STAGING_CLICKHOUSE_URL", "http://localhost:8124"),
		user:     envOr("STAGING_CLICKHOUSE_USER", ""),
		password: envOr("STAGING_CLICKHOUSE_PASSWORD", ""),
	}

	localCH.mustQuery(t, "SELECT 1")
	stagingCH.mustQuery(t, "SELECT 1")

	t.Log("connected to both ClickHouse instances")

	// Discover consumoor _local tables.
	tables := localCH.listLocalTables(t, "consumoor")
	require.NotEmpty(t, tables, "no _local tables in consumoor database")
	t.Logf("found %d consumoor _local tables", len(tables))

	var matched, failed, skippedTables int

	for _, localTable := range tables {
		base := strings.TrimSuffix(localTable, "_local")

		if skipTables[base] {
			t.Logf("SKIP  %-55s excluded table", base)

			skippedTables++

			continue
		}

		// 0. Fetch local columns.
		localCols := localCH.getColumns(t, "consumoor", localTable)
		hasTimeCol := hasColumn(localCols, "event_date_time")

		consumoorN := localCH.rowCount(t, "consumoor", localTable)
		if consumoorN == 0 {
			t.Logf("SKIP  %-55s no rows", base)

			skippedTables++

			continue
		}

		// 1. Get consumoor's time range (if event_date_time exists).
		var minTime, maxTime string

		if hasTimeCol {
			timeRange := localCH.mustQuery(t, fmt.Sprintf(
				"SELECT min(event_date_time), max(event_date_time) FROM `consumoor`.`%s`",
				localTable))

			parts := strings.Split(timeRange, "\t")
			if len(parts) >= 2 && parts[0] != "" && !strings.HasPrefix(parts[0], "1970") {
				minTime, maxTime = parts[0], parts[1]
			}
		}

		// 2. Check if the staging table exists (in default db, without _local suffix).
		stagingTable := base
		if !stagingCH.tableExists(t, "default", stagingTable) {
			// Try with _local suffix in case staging uses local tables too.
			stagingTable = localTable
			if !stagingCH.tableExists(t, "default", stagingTable) {
				t.Logf("SKIP  %-55s not in staging default db", base)

				skippedTables++

				continue
			}
		}

		// 3. Get comparable columns.
		stagingCols := stagingCH.getColumns(t, "default", stagingTable)
		cols := intersectColumnSets(localCols, stagingCols)

		if len(cols) == 0 {
			t.Logf("SKIP  %-55s no comparable columns", base)

			skippedTables++

			continue
		}

		// 4. Build time filter and ordering for staging query.
		var timeFilter, orderBy string

		if hasTimeCol && minTime != "" {
			timeFilter = fmt.Sprintf(
				"event_date_time >= '%s' AND event_date_time <= '%s'",
				minTime, maxTime)
			orderBy = "event_date_time"
		}

		// 5. Fetch rows from both sides.
		limit := maxRows
		if consumoorN < limit {
			limit = consumoorN
		}

		consumoorRows := fetchHashedRows(t, localCH, "consumoor", localTable, cols, "", orderBy, limit)
		stagingRows := fetchHashedRows(t, stagingCH, "default", stagingTable, cols, timeFilter, orderBy, 0)

		// 6. Set difference: rows in consumoor not found in staging.
		onlyConsumoor := setDifference(consumoorRows, stagingRows)

		if len(onlyConsumoor) == 0 {
			t.Logf("OK    %-55s consumoor=%d staging=%d cols=%d",
				base, len(consumoorRows), len(stagingRows), len(cols))

			matched++
		} else {
			t.Logf("FAIL  %-55s consumoor=%d staging=%d unmatched=%d",
				base, len(consumoorRows), len(stagingRows), len(onlyConsumoor))
			logSampleMismatches(t, onlyConsumoor, 5)

			failed++
		}
	}

	t.Logf("--- summary: %d matched, %d failed, %d skipped ---", matched, failed, skippedTables)
	assert.Zero(t, failed, "tables with consumoor rows not found in staging")
	assert.NotZero(t, matched, "expected at least one matching table")
}

// TestStagingTopicCoverage checks that consumoor populated at least one table
// for each registered route. It connects to the local ClickHouse instance
// and reports per-table row counts as a smoke test.
//
// Run:
//
//	CONSUMOOR_STAGING_TEST=true \
//	CLICKHOUSE_URL=http://localhost:8123 \
//	go test ./pkg/consumoor/sinks/clickhouse/transform/flattener/ \
//	  -run TestStagingTopicCoverage -v -timeout 120s
func TestStagingTopicCoverage(t *testing.T) {
	if os.Getenv("CONSUMOOR_STAGING_TEST") != "true" {
		t.Skip("set CONSUMOOR_STAGING_TEST=true to run")
	}

	ch := &chClient{
		baseURL:  envOr("CLICKHOUSE_URL", "http://localhost:8123"),
		user:     envOr("CLICKHOUSE_USER", ""),
		password: envOr("CLICKHOUSE_PASSWORD", ""),
	}

	ch.mustQuery(t, "SELECT 1")

	db := envOr("CLICKHOUSE_DB", "consumoor")
	suffix := envOr("CLICKHOUSE_TABLE_SUFFIX", "_local")

	routes := tabledefs.All()

	seen := make(map[string]struct{}, len(routes))
	tables := make([]string, 0, len(routes))

	for _, route := range routes {
		name := route.TableName()
		if _, ok := seen[name]; ok {
			continue
		}

		seen[name] = struct{}{}
		tables = append(tables, name)
	}

	sort.Strings(tables)

	var populated, empty, missing int

	for _, base := range tables {
		table := base + suffix

		if !ch.tableExists(t, db, table) {
			t.Logf("MISS  %-55s table not found", base)

			missing++

			continue
		}

		n := ch.rowCount(t, db, table)
		if n == 0 {
			t.Logf("EMPTY %-55s", base)

			empty++

			continue
		}

		t.Logf("OK    %-55s rows=%d", base, n)

		populated++
	}

	t.Logf("--- coverage: %d populated, %d empty, %d missing (of %d registered) ---",
		populated, empty, missing, len(tables))

	assert.NotZero(t, populated, "expected at least one table with data")
}

// ---------------------------------------------------------------------------
// Table skip list
// ---------------------------------------------------------------------------

// skipTables lists base table names that are not populated by consumoor and
// should be excluded from comparison.
var skipTables = map[string]bool{
	"schema_migrations":                true,
	"beacon_api_slot":                  true,
	"beacon_block_classification":      true,
	"imported_sources":                 true,
	"mempool_dumpster_transaction":     true,
	"block_native_mempool_transaction": true,
	"ethseer_validator_entity":         true,
	"blob_submitter":                   true,
	"execution_transaction":            true,
}

// ---------------------------------------------------------------------------
// Row hashing and comparison
// ---------------------------------------------------------------------------

// hashedRow holds the deterministic hash of a row alongside its original
// JSON representation for diagnostic output on mismatches.
type hashedRow struct {
	hash string
	raw  string
}

// fetchHashedRows queries ClickHouse and returns hashed rows for comparison.
// If timeFilter is non-empty it is appended as a WHERE clause. orderBy
// specifies the ORDER BY column (empty means no ordering). Limit of 0 means
// no limit.
func fetchHashedRows(
	t *testing.T,
	ch *chClient,
	db, table string,
	cols []columnInfo,
	timeFilter string,
	orderBy string,
	limit int,
) []hashedRow {
	t.Helper()

	colExpr := normalizedColumnExpr(cols)

	sql := fmt.Sprintf("SELECT %s FROM `%s`.`%s`", colExpr, db, table)
	if timeFilter != "" {
		sql += " WHERE " + timeFilter
	}

	if orderBy != "" {
		sql += " ORDER BY " + orderBy
	}

	if limit > 0 {
		sql += fmt.Sprintf(" LIMIT %d", limit)
	}

	sql += " FORMAT JSONEachRow"

	body := ch.mustQuery(t, sql)
	if body == "" {
		return nil
	}

	lines := strings.Split(body, "\n")
	rows := make([]hashedRow, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		h := hashJSONRow(line)
		rows = append(rows, hashedRow{hash: h, raw: line})
	}

	return rows
}

// hashJSONRow produces a deterministic hash of a JSON row by sorting keys
// and normalizing values.
func hashJSONRow(jsonLine string) string {
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonLine), &m); err != nil {
		return fmt.Sprintf("parse-error:%s", jsonLine)
	}

	// Sort keys for determinism.
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}

	sort.Strings(keys)

	h := sha256.New()

	for _, k := range keys {
		fmt.Fprintf(h, "%s=%v|", k, m[k])
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

// setDifference returns items in a that are not in b (by hash).
func setDifference(a, b []hashedRow) []hashedRow {
	bSet := make(map[string]struct{}, len(b))
	for _, row := range b {
		bSet[row.hash] = struct{}{}
	}

	var diff []hashedRow

	for _, row := range a {
		if _, found := bSet[row.hash]; !found {
			diff = append(diff, row)
		}
	}

	return diff
}

// logSampleMismatches logs up to n sample rows that didn't match.
func logSampleMismatches(t *testing.T, rows []hashedRow, n int) {
	t.Helper()

	if len(rows) > n {
		rows = rows[:n]
	}

	for i, row := range rows {
		t.Logf("    sample[%d]: %s", i, abbreviate(row.raw, 200))
	}
}

// ---------------------------------------------------------------------------
// Column helpers
// ---------------------------------------------------------------------------

// excludedColumns are columns that always differ between pipelines.
// Only truly non-deterministic columns belong here — everything else
// should be compared to catch regressions.
var excludedColumns = map[string]bool{
	"updated_date_time": true,
	"meta_client_ip":    true,
	"ip":                true,
}

// getColumns returns column metadata for a table.
func (c *chClient) getColumns(t *testing.T, db, table string) []columnInfo {
	t.Helper()

	r := c.mustQuery(t, fmt.Sprintf(`
		SELECT name, multiIf(
			type LIKE '%%Float32%%', 1,
			type LIKE '%%Float64%%', 1,
			0),
		position
		FROM system.columns
		WHERE database='%s' AND table='%s'
		ORDER BY position`, db, table))

	if r == "" {
		return nil
	}

	lines := strings.Split(r, "\n")
	cols := make([]columnInfo, 0, len(lines))

	for _, line := range lines {
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			continue
		}

		cols = append(cols, columnInfo{
			name:    parts[0],
			isFloat: len(parts) > 1 && parts[1] == "1",
		})
	}

	return cols
}

// hasColumn returns true if the column list contains a column with the
// given name.
func hasColumn(cols []columnInfo, name string) bool {
	for _, c := range cols {
		if c.name == name {
			return true
		}
	}

	return false
}

// intersectColumnSets returns columns present in both sets, excluding
// columns that always differ between pipelines, geo columns, and
// generated hash columns.
func intersectColumnSets(a, b []columnInfo) []columnInfo {
	bNames := make(map[string]struct{}, len(b))
	for _, c := range b {
		bNames[c.name] = struct{}{}
	}

	result := make([]columnInfo, 0, len(a))

	for _, c := range a {
		if _, ok := bNames[c.name]; !ok {
			continue
		}

		if excludedColumns[c.name] {
			continue
		}

		if strings.Contains(c.name, "_geo_") || strings.HasPrefix(c.name, "geo_") {
			continue
		}

		result = append(result, c)
	}

	return result
}

// normalizedColumnExpr builds a SELECT expression that normalizes Nullable
// and float columns for consistent comparison.
func normalizedColumnExpr(cols []columnInfo) string {
	parts := make([]string, len(cols))

	for i, c := range cols {
		if c.isFloat {
			parts[i] = fmt.Sprintf("ROUND(assumeNotNull(`%s`), 6) AS `%s`", c.name, c.name)
		} else {
			parts[i] = fmt.Sprintf("assumeNotNull(`%s`) AS `%s`", c.name, c.name)
		}
	}

	return strings.Join(parts, ", ")
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
		c.http = &http.Client{Timeout: 60 * time.Second}
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

// listLocalTables returns all _local table names in the given database,
// excluding materialized views.
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

// ---------------------------------------------------------------------------
// Formatting helpers
// ---------------------------------------------------------------------------

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

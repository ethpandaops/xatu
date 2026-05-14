package clickhouse

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
)

// ValidateColumns asserts that every column each registered route writes
// exists in its target ClickHouse table. Catches the schema-vs-code skew
// that otherwise surfaces only at first-INSERT time as
// `DB::Exception: There is no column 'X' in table` and stalls the
// deriver's checkpoint while it retries forever.
//
// Direction asymmetry is intentional: only "route writes a column the
// table is missing" is fatal. The other direction (table has a column
// the route doesn't write) is fine — those columns are simply not
// listed in the INSERT and ClickHouse fills them with DEFAULT. That is
// the supported "cannon writes to legacy schema, leaves geo columns
// empty" semantic.
//
// Honors the same `failOnMissingTables` toggle as ValidateTables — set
// it false to downgrade to warnings.
func (w *Writer) ValidateColumns(ctx context.Context) error {
	if len(w.batchFactories) == 0 {
		return nil
	}

	expected := buildExpectedColumns(w.batchFactories, w.config.TableSuffix)

	tables := make([]string, 0, len(expected))
	for table := range expected {
		tables = append(tables, table)
	}

	actual, err := w.fetchTableColumns(ctx, tables)
	if err != nil {
		return fmt.Errorf("querying table columns: %w", err)
	}

	problems := findMissingColumns(expected, actual)
	if len(problems) == 0 {
		w.log.WithField("tables_checked", len(expected)).WithContext(ctx).
			Info("All registered route columns exist in ClickHouse")

		return nil
	}

	for _, p := range problems {
		w.log.WithFields(logrus.Fields{
			"table":           p.table,
			"missing_columns": p.missing,
			"missing_count":   len(p.missing),
			"database":        w.database,
		}).WithContext(ctx).Error("Registered route writes columns that the target table is missing — INSERTs to this table will fail")
	}

	if !w.config.ShouldFailOnMissingTables() {
		w.log.WithField("tables_with_missing_columns", len(problems)).WithContext(ctx).
			Warn("Continuing despite column mismatch — INSERTs to affected tables will fail (failOnMissingTables is disabled)")

		return nil
	}

	return fmt.Errorf(
		"clickhouse: %d table(s) have route columns missing from the target schema in database %q "+
			"— apply the matching migration before deploying this binary "+
			"(set failOnMissingTables: false to downgrade to warnings)",
		len(problems), w.database,
	)
}

// columnMismatch describes one table whose target schema is missing
// columns that the route would try to INSERT.
type columnMismatch struct {
	table   string
	missing []string
}

// buildExpectedColumns extracts the column-name set each route's batch
// would emit, keyed by the actual ClickHouse table name (base name +
// TableSuffix). Pure function for testability.
func buildExpectedColumns(
	factories map[string]func() route.ColumnarBatch,
	tableSuffix string,
) map[string]map[string]struct{} {
	expected := make(map[string]map[string]struct{}, len(factories))

	for baseTable, factory := range factories {
		target := baseTable + tableSuffix

		cols := make(map[string]struct{}, 32)
		for _, col := range factory().Input() {
			cols[col.Name] = struct{}{}
		}

		expected[target] = cols
	}

	return expected
}

// findMissingColumns computes, for each (table, expected-col-set) pair,
// the sorted list of columns absent from the actual schema. Pure
// function for unit testing. Tables not present in `actual` are skipped
// — ValidateTables is responsible for surfacing missing-table errors.
func findMissingColumns(
	expected map[string]map[string]struct{},
	actual map[string]map[string]struct{},
) []columnMismatch {
	var problems []columnMismatch

	for table, expectedCols := range expected {
		existing, ok := actual[table]
		if !ok {
			continue
		}

		var missing []string

		for col := range expectedCols {
			if _, ok := existing[col]; !ok {
				missing = append(missing, col)
			}
		}

		if len(missing) > 0 {
			sort.Strings(missing)
			problems = append(problems, columnMismatch{table: table, missing: missing})
		}
	}

	sort.Slice(problems, func(i, j int) bool {
		return problems[i].table < problems[j].table
	})

	return problems
}

// fetchTableColumns queries system.columns once for all tables of
// interest and returns a (table → set-of-column-names) map.
func (w *Writer) fetchTableColumns(
	ctx context.Context,
	tables []string,
) (map[string]map[string]struct{}, error) {
	cols := make(map[string]map[string]struct{}, len(tables))

	if len(tables) == 0 {
		return cols, nil
	}

	quoted := make([]string, len(tables))
	for i, t := range tables {
		quoted[i] = quoteCHString(t)
	}

	inClause := strings.Join(quoted, ",")

	var (
		tableCol proto.ColStr
		nameCol  proto.ColStr
	)

	err := w.doWithRetry(ctx, "validate_columns", func(attemptCtx context.Context) error {
		pool := w.getPool()
		if pool == nil {
			return ch.ErrClosed
		}

		// Reset for retry attempts.
		tableCol.Reset()
		nameCol.Reset()

		// Drop accumulated state from any previous attempt.
		for k := range cols {
			delete(cols, k)
		}

		return pool.Do(attemptCtx, ch.Query{
			Body: fmt.Sprintf(
				"SELECT table, name FROM system.columns WHERE database = '%s' AND table IN (%s)",
				escapeCHString(w.database), inClause,
			),
			Result: proto.Results{
				{Name: "table", Data: &tableCol},
				{Name: "name", Data: &nameCol},
			},
			OnResult: func(_ context.Context, block proto.Block) error {
				for i := 0; i < block.Rows; i++ {
					table := tableCol.Row(i)
					name := nameCol.Row(i)

					if cols[table] == nil {
						cols[table] = make(map[string]struct{}, 32)
					}

					cols[table][name] = struct{}{}
				}

				return nil
			},
		})
	})
	if err != nil {
		return nil, err
	}

	return cols, nil
}

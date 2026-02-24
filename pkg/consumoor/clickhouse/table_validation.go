package clickhouse

import (
	"context"
	"fmt"
	"sort"

	"github.com/ClickHouse/ch-go"
	"github.com/ClickHouse/ch-go/proto"
	"github.com/sirupsen/logrus"
)

// ValidateTables queries ClickHouse system.tables and checks that every
// registered route table (with TableSuffix applied) exists. Missing tables
// are logged as warnings. If Config.FailOnMissingTables is true, an error
// is returned instead.
func (w *ChGoWriter) ValidateTables(ctx context.Context, routeTableNames []string) error {
	existing, err := w.fetchExistingTables(ctx)
	if err != nil {
		return fmt.Errorf("querying existing tables: %w", err)
	}

	expected := make([]string, 0, len(routeTableNames))
	for _, base := range routeTableNames {
		expected = append(expected, base+w.config.TableSuffix)
	}

	missing := findMissingTables(expected, existing)
	if len(missing) == 0 {
		w.log.WithField("tables_checked", len(expected)).
			Info("All registered route tables exist in ClickHouse")

		return nil
	}

	for _, table := range missing {
		w.log.WithFields(logrus.Fields{
			"table":        table,
			"table_suffix": w.config.TableSuffix,
			"database":     w.database,
		}).Warn("Registered route table does not exist in ClickHouse — " +
			"INSERTs to this table will be permanently dropped")
	}

	if w.config.FailOnMissingTables {
		return fmt.Errorf(
			"clickhouse: %d registered route table(s) missing in database %q "+
				"(set failOnMissingTables: false to downgrade to warnings)",
			len(missing), w.database,
		)
	}

	w.log.WithField("missing_count", len(missing)).
		WithField("total_checked", len(expected)).
		Warn("Some registered route tables are missing — data for these " +
			"tables will be silently dropped")

	return nil
}

// fetchExistingTables returns the set of table names present in the
// writer's target database by querying system.tables.
func (w *ChGoWriter) fetchExistingTables(ctx context.Context) (map[string]struct{}, error) {
	var nameCol proto.ColStr

	tables := make(map[string]struct{}, 128)

	err := w.doWithRetry(ctx, "validate_tables", func(attemptCtx context.Context) error {
		pool := w.getPool()
		if pool == nil {
			return ch.ErrClosed
		}

		// Reset for retry attempts.
		nameCol.Reset()

		return pool.Do(attemptCtx, ch.Query{
			Body: fmt.Sprintf(
				"SELECT name FROM system.tables WHERE database = '%s'",
				escapeCHString(w.database),
			),
			Result: proto.Results{
				{Name: "name", Data: &nameCol},
			},
			OnResult: func(_ context.Context, block proto.Block) error {
				for i := 0; i < block.Rows; i++ {
					tables[nameCol.Row(i)] = struct{}{}
				}

				return nil
			},
		})
	})
	if err != nil {
		return nil, err
	}

	return tables, nil
}

// findMissingTables returns the sorted list of expected table names that
// are not present in the existing set. This is a pure function to allow
// unit testing without a ClickHouse connection.
func findMissingTables(expected []string, existing map[string]struct{}) []string {
	var missing []string

	for _, name := range expected {
		if _, ok := existing[name]; !ok {
			missing = append(missing, name)
		}
	}

	sort.Strings(missing)

	return missing
}

// escapeCHString escapes single quotes for use in ClickHouse string literals.
func escapeCHString(s string) string {
	quoted := quoteCHString(s)

	return quoted[1 : len(quoted)-1]
}

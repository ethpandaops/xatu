// Command generate spins up a ClickHouse container via testcontainers,
// applies all migrations, and regenerates every .gen.go file using chgo-rowgen.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// clickhouseConfig is a single-node ClickHouse config with embedded Keeper,
// derived from deploy/local/docker-compose/clickhouse-01 but adapted for codegen.
const clickhouseConfig = `<clickhouse replace="true">
    <logger>
        <level>warning</level>
        <console>1</console>
    </logger>
    <listen_host>0.0.0.0</listen_host>
    <http_port>8123</http_port>
    <tcp_port>9000</tcp_port>

    <keeper_server>
        <tcp_port>9181</tcp_port>
        <server_id>1</server_id>
        <raft_configuration>
            <server>
                <id>1</id>
                <hostname>localhost</hostname>
                <port>9234</port>
            </server>
        </raft_configuration>
    </keeper_server>

    <zookeeper>
        <node>
            <host>localhost</host>
            <port>9181</port>
        </node>
    </zookeeper>

    <distributed_ddl>
        <path>/clickhouse/task_queue/ddl</path>
    </distributed_ddl>

    <remote_servers>
        <cluster_2S_1R>
            <secret>supersecret</secret>
            <shard>
                <replica>
                    <host>localhost</host>
                    <port>9000</port>
                </replica>
            </shard>
        </cluster_2S_1R>
    </remote_servers>

    <macros>
        <installation>xatu</installation>
        <cluster>cluster_2S_1R</cluster>
        <shard>01</shard>
        <replica>01</replica>
    </macros>
</clickhouse>
`

func main() {
	opts, err := parseOptions(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(2)
	}

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "generate: %v\n", err)
		os.Exit(1)
	}
}

type options struct {
	migrationMin int
}

func parseOptions(args []string) (*options, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	fs := flag.NewFlagSet("generate", flag.ContinueOnError)

	var opts options

	fs.IntVar(
		&opts.migrationMin,
		"migration-min",
		0,
		"minimum numeric migration prefix to apply (e.g. 105 to skip 001-104)",
	)

	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	if opts.migrationMin < 0 {
		return nil, fmt.Errorf("migration-min must be >= 0")
	}

	return &opts, nil
}

func run(opts *options) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Resolve project root (directory containing go.mod).
	root, err := findProjectRoot()
	if err != nil {
		return fmt.Errorf("find project root: %w", err)
	}

	// Start ClickHouse container.
	fmt.Println("Starting ClickHouse container...")

	container, httpPort, nativePort, startErr := startClickHouse(ctx)
	if startErr != nil {
		return fmt.Errorf("start clickhouse: %w", startErr)
	}

	defer func() {
		_ = container.Terminate(context.Background())
	}()

	fmt.Printf("ClickHouse ready (HTTP :%s, native :%s)\n", httpPort, nativePort)

	// Apply migrations.
	fmt.Println("Applying migrations...")

	if migrateErr := applyMigrations(ctx, httpPort, root, opts.migrationMin); migrateErr != nil {
		return fmt.Errorf("apply migrations: %w", migrateErr)
	}

	// Build chgo-rowgen.
	fmt.Println("Building chgo-rowgen...")

	tmpDir, err := os.MkdirTemp("", "chgo-rowgen-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}

	defer os.RemoveAll(tmpDir)

	rowgenBin := filepath.Join(tmpDir, "chgo-rowgen")

	buildCmd := exec.CommandContext(ctx, "go", "build", "-o", rowgenBin,
		"./pkg/consumoor/route/cmd/chgo-rowgen")
	buildCmd.Dir = root
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build chgo-rowgen: %w", err)
	}

	// Discover all *_local tables from ClickHouse.
	fmt.Println("Discovering tables...")

	chURL := fmt.Sprintf("http://localhost:%s/", httpPort)

	tables, discoverErr := discoverTables(ctx, chURL)
	if discoverErr != nil {
		return fmt.Errorf("discover tables: %w", discoverErr)
	}

	fmt.Printf("Found %d tables.\n", len(tables))

	// Generate all tables.
	dsn := fmt.Sprintf("clickhouse://localhost:%s/default", nativePort)
	tablesDir := filepath.Join(root,
		"pkg", "consumoor", "route")

	generated := 0
	scaffolded := 0

	for _, table := range tables {
		pkg, ok := resolvePackage(table)
		if !ok {
			continue
		}

		generated++
		typeName := toLowerCamel(table)
		outDir := filepath.Join(tablesDir, pkg)
		outPath := filepath.Join(outDir, table+".gen.go")

		if mkErr := os.MkdirAll(outDir, 0o755); mkErr != nil {
			return fmt.Errorf("mkdir %s: %w", outDir, mkErr)
		}

		fmt.Printf("  [%d/%d] %s -> %s\n", generated, len(tables), table, pkg)

		cmd := exec.CommandContext(ctx, rowgenBin,
			"-dsn", dsn,
			"-table", table+"_local",
			"-type", typeName,
			"-package", pkg,
			"-out", outPath,
		)
		cmd.Dir = root
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("generate %s: %w", table, err)
		}

		// Scaffold the hand-written route file if it doesn't exist yet.
		routePath := filepath.Join(outDir, table+".go")
		if _, statErr := os.Stat(routePath); os.IsNotExist(statErr) {
			if scaffoldErr := writeRouteScaffold(routePath, pkg, typeName, table); scaffoldErr != nil {
				return fmt.Errorf("scaffold %s: %w", table, scaffoldErr)
			}

			scaffolded++

			fmt.Printf("    scaffolded %s (needs implementation)\n", routePath)
		}

		// Scaffold the test file if it doesn't exist yet.
		testPath := filepath.Join(outDir, table+"_test.go")
		if _, statErr := os.Stat(testPath); os.IsNotExist(statErr) {
			if scaffoldErr := writeRouteTestScaffold(testPath, pkg, typeName, table); scaffoldErr != nil {
				return fmt.Errorf("scaffold test %s: %w", table, scaffoldErr)
			}

			fmt.Printf("    scaffolded %s (needs assertions)\n", testPath)
		}
	}

	fmt.Printf("Generated %d files, scaffolded %d route files (skipped %d unmatched tables).\n",
		generated, scaffolded, len(tables)-generated)

	return nil
}

// startClickHouse creates a ClickHouse container with embedded Keeper and
// returns the container, mapped HTTP port, and mapped native port.
func startClickHouse(ctx context.Context) (
	container testcontainers.Container, httpPort string, nativePort string, err error,
) {
	req := testcontainers.ContainerRequest{
		Image:        "clickhouse/clickhouse-server:24",
		ExposedPorts: []string{"8123/tcp", "9000/tcp"},
		WaitingFor:   wait.ForHTTP("/").WithPort("8123/tcp").WithStartupTimeout(60 * time.Second),
		Files: []testcontainers.ContainerFile{
			{
				Reader:            strings.NewReader(clickhouseConfig),
				ContainerFilePath: "/etc/clickhouse-server/config.d/codegen.xml",
				FileMode:          0o644,
			},
		},
	}

	container, err = testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, "", "", fmt.Errorf("create container: %w", err)
	}

	mapped, err := container.MappedPort(ctx, "8123/tcp")
	if err != nil {
		_ = container.Terminate(ctx)

		return nil, "", "", fmt.Errorf("map HTTP port: %w", err)
	}

	httpPort = mapped.Port()

	mapped, err = container.MappedPort(ctx, "9000/tcp")
	if err != nil {
		_ = container.Terminate(ctx)

		return nil, "", "", fmt.Errorf("map native port: %w", err)
	}

	nativePort = mapped.Port()

	return container, httpPort, nativePort, nil
}

// applyMigrations reads all *.up.sql files from the migrations directory,
// splits each on semicolons, and POSTs each statement to ClickHouse HTTP.
func applyMigrations(
	ctx context.Context,
	httpPort, root string,
	migrationMin int,
) error {
	migrationsDir := filepath.Join(root, "deploy", "migrations", "clickhouse")

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	// Collect and sort .up.sql files.
	var files []string

	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".up.sql") {
			files = append(files, e.Name())
		}
	}

	sort.Strings(files)

	chURL := fmt.Sprintf("http://localhost:%s/", httpPort)

	appliedFiles := 0
	skippedFiles := 0
	totalStatements := 0

	for _, name := range files {
		version, parseErr := migrationVersion(name)
		if parseErr != nil {
			return parseErr
		}

		if version < migrationMin {
			skippedFiles++
			fmt.Printf("  - skip %s (version %d < migration-min %d)\n", name, version, migrationMin)

			continue
		}

		data, readErr := os.ReadFile(filepath.Join(migrationsDir, name))
		if readErr != nil {
			return fmt.Errorf("read %s: %w", name, readErr)
		}

		sqlText := normalizeEscapedSQL(string(data))
		stmts := splitSQLStatements(sqlText)
		executableCount := 0

		for _, stmt := range stmts {
			if hasExecutableSQL(stmt) {
				executableCount++
			}
		}

		fmt.Printf("  - apply %s (%d statements)\n", name, executableCount)

		for _, stmt := range stmts {
			if !hasExecutableSQL(stmt) {
				continue
			}

			if execErr := execClickHouseHTTP(ctx, chURL, stmt); execErr != nil {
				return fmt.Errorf("exec %s: %w", name, execErr)
			}

			totalStatements++
		}

		appliedFiles++
	}

	fmt.Printf(
		"Migrations complete: applied %d file(s), skipped %d file(s), executed %d statement(s)\n",
		appliedFiles,
		skippedFiles,
		totalStatements,
	)

	return nil
}

func migrationVersion(filename string) (int, error) {
	dot := strings.IndexByte(filename, '_')
	if dot <= 0 {
		return 0, fmt.Errorf("invalid migration filename %q: missing numeric prefix", filename)
	}

	v, err := strconv.Atoi(filename[:dot])
	if err != nil {
		return 0, fmt.Errorf("invalid migration filename %q: %w", filename, err)
	}

	return v, nil
}

// normalizeEscapedSQL decodes SQL dumps that were written with escaped
// newlines/quotes (e.g. "\\n", "\\r\\n", "\\'"), producing executable SQL.
func normalizeEscapedSQL(sql string) string {
	if strings.Contains(sql, `\r\n`) {
		sql = strings.ReplaceAll(sql, `\r\n`, "\n")
	}

	if strings.Contains(sql, `\n`) {
		sql = strings.ReplaceAll(sql, `\n`, "\n")
	}

	if strings.Contains(sql, `\'`) {
		sql = strings.ReplaceAll(sql, `\'`, "'")
	}

	return sql
}

// discoverTables queries ClickHouse for all *_local tables in the default
// database and returns their logical names (with _local suffix stripped),
// sorted alphabetically.
func discoverTables(ctx context.Context, chURL string) ([]string, error) {
	query := "SELECT name FROM system.tables WHERE database = 'default' AND name LIKE '%_local' ORDER BY name"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chURL, strings.NewReader(query))
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	lines := strings.Split(strings.TrimSpace(string(body)), "\n")
	tables := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		tables = append(tables, strings.TrimSuffix(line, "_local"))
	}

	return tables, nil
}

// execClickHouseHTTP posts a single SQL statement to ClickHouse HTTP interface.
func execClickHouseHTTP(ctx context.Context, chURL, stmt string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chURL, strings.NewReader(stmt))
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	_, _ = io.Copy(io.Discard, resp.Body)

	return nil
}

// findProjectRoot walks up from the current working directory to find go.mod.
func findProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, "go.mod")); statErr == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}

		dir = parent
	}
}

// toLowerCamel converts a snake_case string to lowerCamelCase.
// e.g. "beacon_api_eth_v1_events_block" -> "beaconApiEthV1EventsBlock"
func toLowerCamel(s string) string {
	parts := strings.Split(s, "_")
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}

	return strings.Join(parts, "")
}

// splitSQLStatements splits SQL text into statements on semicolons while
// respecting quoted strings and comments.
func splitSQLStatements(sql string) []string {
	stmts := make([]string, 0, 64)
	var buf strings.Builder

	const (
		stateNormal = iota
		stateSingleQuote
		stateDoubleQuote
		stateBacktick
		stateLineComment
		stateBlockComment
	)

	state := stateNormal

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		switch state {
		case stateNormal:
			switch {
			case ch == '\'':
				state = stateSingleQuote
				buf.WriteByte(ch)
			case ch == '"':
				state = stateDoubleQuote
				buf.WriteByte(ch)
			case ch == '`':
				state = stateBacktick
				buf.WriteByte(ch)
			case isLineCommentStart(sql, i):
				state = stateLineComment
				buf.WriteByte(ch)
				if ch == '-' {
					i++
					buf.WriteByte(sql[i])
				}
			case ch == '/' && i+1 < len(sql) && sql[i+1] == '*':
				state = stateBlockComment
				buf.WriteByte(ch)
				i++
				buf.WriteByte(sql[i])
			case ch == ';':
				stmt := strings.TrimSpace(buf.String())
				if stmt != "" {
					stmts = append(stmts, stmt)
				}
				buf.Reset()
			default:
				buf.WriteByte(ch)
			}
		case stateSingleQuote:
			buf.WriteByte(ch)
			if ch == '\\' && i+1 < len(sql) {
				i++
				buf.WriteByte(sql[i])
				continue
			}

			if ch == '\'' {
				// SQL escaped quote: ''.
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
					buf.WriteByte(sql[i])
					continue
				}

				state = stateNormal
			}
		case stateDoubleQuote:
			buf.WriteByte(ch)
			if ch == '\\' && i+1 < len(sql) {
				i++
				buf.WriteByte(sql[i])
				continue
			}

			if ch == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i++
					buf.WriteByte(sql[i])
					continue
				}

				state = stateNormal
			}
		case stateBacktick:
			buf.WriteByte(ch)
			if ch == '`' {
				state = stateNormal
			}
		case stateLineComment:
			buf.WriteByte(ch)
			if ch == '\n' {
				state = stateNormal
			}
		case stateBlockComment:
			buf.WriteByte(ch)
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				i++
				buf.WriteByte(sql[i])
				state = stateNormal
			}
		}
	}

	tail := strings.TrimSpace(buf.String())
	if tail != "" {
		stmts = append(stmts, tail)
	}

	return stmts
}

func hasExecutableSQL(stmt string) bool {
	return strings.TrimSpace(stripSQLComments(stmt)) != ""
}

func stripSQLComments(sql string) string {
	var buf strings.Builder

	const (
		stateNormal = iota
		stateSingleQuote
		stateDoubleQuote
		stateBacktick
		stateLineComment
		stateBlockComment
	)

	state := stateNormal

	for i := 0; i < len(sql); i++ {
		ch := sql[i]

		switch state {
		case stateNormal:
			switch {
			case ch == '\'':
				state = stateSingleQuote
				buf.WriteByte(ch)
			case ch == '"':
				state = stateDoubleQuote
				buf.WriteByte(ch)
			case ch == '`':
				state = stateBacktick
				buf.WriteByte(ch)
			case isLineCommentStart(sql, i):
				state = stateLineComment
				if ch == '-' {
					i++
				}
			case ch == '/' && i+1 < len(sql) && sql[i+1] == '*':
				state = stateBlockComment
				i++
			default:
				buf.WriteByte(ch)
			}
		case stateSingleQuote:
			buf.WriteByte(ch)
			if ch == '\\' && i+1 < len(sql) {
				i++
				buf.WriteByte(sql[i])
				continue
			}

			if ch == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
					buf.WriteByte(sql[i])
					continue
				}

				state = stateNormal
			}
		case stateDoubleQuote:
			buf.WriteByte(ch)
			if ch == '\\' && i+1 < len(sql) {
				i++
				buf.WriteByte(sql[i])
				continue
			}

			if ch == '"' {
				if i+1 < len(sql) && sql[i+1] == '"' {
					i++
					buf.WriteByte(sql[i])
					continue
				}

				state = stateNormal
			}
		case stateBacktick:
			buf.WriteByte(ch)
			if ch == '`' {
				state = stateNormal
			}
		case stateLineComment:
			if ch == '\n' {
				buf.WriteByte(ch)
				state = stateNormal
			}
		case stateBlockComment:
			if ch == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				i++
				state = stateNormal
			}
		}
	}

	return buf.String()
}

func isLineCommentStart(sql string, i int) bool {
	if sql[i] == '#' {
		return true
	}

	if sql[i] != '-' || i+1 >= len(sql) || sql[i+1] != '-' {
		return false
	}

	// Treat "--" as a comment start only when followed by whitespace/line end.
	if i+2 >= len(sql) {
		return true
	}

	next := sql[i+2]

	return next == ' ' || next == '\t' || next == '\n' || next == '\r'
}

package main

import (
	"strings"
	"testing"
)

func TestSplitSQLStatements(t *testing.T) {
	t.Parallel()

	sql := strings.Join([]string{
		"-- migration header;",
		"",
		"CREATE TABLE t1 (",
		"    id UInt64,",
		"    note String COMMENT 'a;b',",
		"    raw String COMMENT '-- not a comment here'",
		");",
		"",
		"/* block comment; with semicolon */",
		"INSERT INTO t1 VALUES (1, 'x;--y', 'z');",
		"# trailing comment;",
	}, "\n")

	got := splitSQLStatements(sql)
	if len(got) != 3 {
		t.Fatalf("expected 3 statements, got %d: %#v", len(got), got)
	}

	executable := make([]string, 0, len(got))
	for _, stmt := range got {
		if hasExecutableSQL(stmt) {
			executable = append(executable, stmt)
		}
	}

	if len(executable) != 2 {
		t.Fatalf("expected 2 executable statements, got %d: %#v", len(executable), executable)
	}

	if !strings.Contains(executable[0], "CREATE TABLE t1") {
		t.Fatalf("expected CREATE TABLE in first executable statement, got: %q", executable[0])
	}

	if !strings.Contains(executable[1], "INSERT INTO t1") {
		t.Fatalf("expected INSERT in second executable statement, got: %q", executable[1])
	}
}

func TestHasExecutableSQL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		sql  string
		want bool
	}{
		{
			name: "line_comment_only",
			sql:  "-- just a comment",
			want: false,
		},
		{
			name: "block_comment_only",
			sql:  "/* just a comment */",
			want: false,
		},
		{
			name: "hash_comment_only",
			sql:  "# just a comment",
			want: false,
		},
		{
			name: "query_with_comment",
			sql:  "SELECT 1 -- inline comment",
			want: true,
		},
		{
			name: "double_dash_no_space_is_not_comment",
			sql:  "SELECT 1--2",
			want: true,
		},
		{
			name: "quoted_comment_marker",
			sql:  "SELECT '-- not comment'",
			want: true,
		},
	}

	for _, tt := range tests {
		tt := tt

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := hasExecutableSQL(tt.sql)
			if got != tt.want {
				t.Fatalf("hasExecutableSQL(%q) = %v, want %v", tt.sql, got, tt.want)
			}
		})
	}
}

func TestMigrationVersion(t *testing.T) {
	t.Parallel()

	v, err := migrationVersion("105_schema_v2.up.sql")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if v != 105 {
		t.Fatalf("expected version 105, got %d", v)
	}

	if _, err := migrationVersion("schema_v2.up.sql"); err == nil {
		t.Fatalf("expected error for filename without numeric prefix")
	}
}

func TestParseOptions(t *testing.T) {
	t.Parallel()

	opts, err := parseOptions([]string{"-migration-min", "105"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if opts.migrationMin != 105 {
		t.Fatalf("expected migrationMin 105, got %d", opts.migrationMin)
	}

	opts, err = parseOptions([]string{"--", "-migration-min", "105"})
	if err != nil {
		t.Fatalf("unexpected error for leading --: %v", err)
	}

	if opts.migrationMin != 105 {
		t.Fatalf("expected migrationMin 105 with leading --, got %d", opts.migrationMin)
	}

	if _, err := parseOptions([]string{"-migration-min", "-1"}); err == nil {
		t.Fatalf("expected error for negative migration-min")
	}
}

func TestNormalizeEscapedSQL(t *testing.T) {
	t.Parallel()

	in := "CREATE TABLE t\\n(\\n  `c` String COMMENT \\'hello\\'\\n);\\r\\n"
	got := normalizeEscapedSQL(in)

	if strings.Contains(got, `\n`) {
		t.Fatalf("expected escaped newlines to be decoded, got: %q", got)
	}

	if strings.Contains(got, `\'`) {
		t.Fatalf("expected escaped quotes to be decoded, got: %q", got)
	}

	if !strings.Contains(got, "COMMENT 'hello'") {
		t.Fatalf("expected decoded comment literal, got: %q", got)
	}
}

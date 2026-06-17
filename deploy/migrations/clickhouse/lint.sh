#!/usr/bin/env bash
# Lint the ClickHouse migration sets for database-agnosticism.
#
# The migration sets under deploy/migrations/clickhouse/<set>/ are applied into a
# database chosen at apply time (golang-migrate `database=`), so the SQL must never
# pin a database itself. This linter enforces three invariants on every *.sql file:
#
#   1. No `CREATE DATABASE`            -- databases are provisioned by the migrator.
#   2. No database-qualified idents    -- e.g. `default.foo`; tables are unqualified
#                                         and resolved via the connection's database=.
#   3. Distributed() uses currentDatabase() as its database (2nd) argument, never a
#      hardcoded db name, a string literal, or the {database} macro.
#
# Rules 1 and 2 are evaluated against a sanitized view of each file in which `--` /
# `/* */` comments and single-quoted string literals are blanked out (line numbers
# preserved), so legitimate dotted text inside COMMENT '...' (e.g. 'ethseer.io')
# never trips the qualifier check. Rule 3 reconstructs statements by splitting the
# sanitized text on `;`.
#
# Usage: lint.sh [MIGRATIONS_DIR]   (default: dir of this script)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MIGRATIONS_DIR="${1:-$SCRIPT_DIR}"

# Emit GitHub Actions inline annotations when running in CI.
CI_ANNOTATE=0
[ "${GITHUB_ACTIONS:-}" = "true" ] && CI_ANNOTATE=1

shopt -s nullglob
files=("$MIGRATIONS_DIR"/*/*.sql)
if [ ${#files[@]} -eq 0 ]; then
  echo "no migration .sql files found under ${MIGRATIONS_DIR}" >&2
  exit 1
fi

status=0
for file in "${files[@]}"; do
  if ! awk -v file="$file" -v ci="$CI_ANNOTATE" '
    BEGIN { Q = sprintf("%c", 39) }                             # single-quote char
    # --- Sanitize one line, carrying string/block-comment state across lines. ---
    # Blanks out comments and the *contents* of single-quoted strings (quotes kept),
    # so dotted text inside literals/comments cannot be mistaken for a db qualifier.
    function sanitize(line,    i, n, c, c2, out) {
      out = ""; i = 1; n = length(line)
      while (i <= n) {
        c = substr(line, i, 1); c2 = substr(line, i + 1, 1)
        if (in_block) {
          if (c == "*" && c2 == "/") { in_block = 0; out = out "  "; i += 2; continue }
          out = out " "; i++; continue
        }
        if (in_string) {
          if (c == "\\") { out = out "  "; i += 2; continue }   # backslash-escaped char
          if (c == Q)    { in_string = 0; out = out Q; i++; continue }
          out = out " "; i++; continue                          # blank string content
        }
        if (c == "-" && c2 == "-") break                        # -- line comment: drop rest
        if (c == "/" && c2 == "*") { in_block = 1; out = out "  "; i += 2; continue }
        if (c == Q) { in_string = 1; out = out Q; i++; continue }
        out = out c; i++
      }
      return out
    }
    function report(line, msg) {
      printf "%s:%d: %s\n", file, line, msg
      if (ci) printf "::error file=%s,line=%d::%s\n", file, line, msg
      violations++
    }
    function check_stmt(stmt, line,    low) {
      gsub(/[ \t]+/, " ", stmt)                                 # collapse whitespace
      low = tolower(stmt)
      if (low ~ /distributed[ \t]*\(/ &&
          low !~ /distributed[ \t]*\([^,]*,[ \t]*currentdatabase\(\)/)
        report(line, "Distributed() must use currentDatabase() as its database argument")
    }
    {
      raw[NR] = $0
      san[NR] = sanitize($0)
    }
    END {
      violations = 0
      for (k = 1; k <= NR; k++) {
        s = san[k]
        # Rule 1: no CREATE DATABASE.
        if (tolower(s) ~ /create[ \t]+database/)
          report(k, "CREATE DATABASE is not allowed (databases are provisioned by the migrator)")
        # Rule 2: no database-qualified identifiers (db.table).
        if (match(s, /[A-Za-z_][A-Za-z0-9_]*[ \t]*\.[ \t]*[A-Za-z_]/)) {
          tok = substr(s, RSTART, RLENGTH)
          gsub(/[ \t]/, "", tok)
          report(k, "database-qualified identifier \"" tok "...\" is not allowed (use an unqualified name; the database is chosen at apply time)")
        }
      }
      # Rule 3: reconstruct statements (split on `;`) and check each Distributed().
      stmt = ""; start = 0
      for (k = 1; k <= NR; k++) {
        rest = san[k]
        while ((p = index(rest, ";")) > 0) {
          seg = substr(rest, 1, p - 1)
          if (start == 0 && seg ~ /[^ \t]/) start = k
          if (start > 0) check_stmt(stmt " " seg, start)
          stmt = ""; start = 0
          rest = substr(rest, p + 1)
        }
        if (start == 0 && rest ~ /[^ \t]/) start = k
        stmt = stmt " " rest
      }
      if (stmt ~ /[^ \t]/ && start > 0) check_stmt(stmt, start)   # trailing stmt, no `;`
      exit (violations > 0)
    }
  ' "$file"; then
    status=1
  fi
done

if [ "$status" -ne 0 ]; then
  echo "" >&2
  echo "ClickHouse migration lint failed: migrations must be database-agnostic." >&2
  echo "See deploy/migrations/clickhouse/lint.sh for the rules." >&2
  exit 1
fi

echo "ClickHouse migration lint passed (${#files[@]} files)."

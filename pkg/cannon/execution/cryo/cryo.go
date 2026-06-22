// Package cryo wraps the third-party `cryo` binary
// (https://github.com/paradigmxyz/cryo) so EL cannon can extract
// execution-layer datasets over a block range and read them back as typed
// rows. The runner is dataset-agnostic: it invokes cryo for one dataset over
// an inclusive block range, writes parquet into a unique temp directory, and
// returns the file paths. Callers parse those files into dataset-specific
// structs with ReadParquet.
//
// Two cryo options matter for round-tripping through parquet-go:
//   - --hex encodes binary columns as 0x-prefixed hex strings (large_string),
//     which is exactly what the canonical_execution_* ClickHouse schema wants.
//   - --compression zstd avoids cryo's default lz4, which parquet-go decodes
//     incorrectly (it produces garbage byte-array offsets at volume). zstd is
//     unambiguous and read cleanly by parquet-go.
package cryo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/parquet-go/parquet-go"
	"github.com/sirupsen/logrus"
)

// Config configures the cryo runner.
type Config struct {
	// BinaryPath is the path to the cryo binary.
	BinaryPath string `yaml:"binaryPath" default:"cryo"`
	// OutputDir is the parent directory for per-invocation temp output.
	OutputDir string `yaml:"outputDir" default:"/tmp/xatu-cannon-cryo"`
	// MaxRangeSize is the default number of blocks per cryo invocation.
	MaxRangeSize uint64 `yaml:"maxRangeSize" default:"50"`
	// Compression is the cryo parquet compression spec (name plus optional
	// level, e.g. "zstd 1"). Avoid cryo's lz4 default — parquet-go mis-decodes
	// it. Set as space-separated tokens passed straight to `cryo --compression`.
	Compression string `yaml:"compression" default:"zstd 1"`
	// RequestsPerSecond rate-limits cryo's RPC calls (0 = cryo default).
	RequestsPerSecond uint64 `yaml:"requestsPerSecond" default:"0"`
	// MaxConcurrentChunks bounds cryo's internal chunk concurrency (0 = default).
	MaxConcurrentChunks uint64 `yaml:"maxConcurrentChunks" default:"0"`
}

// Validate checks the config is usable.
func (c *Config) Validate() error {
	if c.BinaryPath == "" {
		return errors.New("cryo binaryPath is required")
	}

	if c.MaxRangeSize == 0 {
		return errors.New("cryo maxRangeSize must be greater than 0")
	}

	return nil
}

// Runner invokes cryo against a single execution RPC endpoint.
type Runner struct {
	log logrus.FieldLogger
	cfg *Config
	rpc string
}

// New creates a cryo runner. rpc is the execution-layer RPC URL (credentials
// may be embedded as https://user:pass@host).
func New(log logrus.FieldLogger, cfg *Config, rpc string) *Runner {
	return &Runner{
		log: log.WithField("module", "cannon/execution/cryo"),
		cfg: cfg,
		rpc: rpc,
	}
}

// Collection is the result of a cryo invocation. Close removes the temp dir.
type Collection struct {
	Dir   string
	Files []string
}

// Close removes the temporary output directory.
func (c *Collection) Close() error {
	if c.Dir == "" {
		return nil
	}

	return os.RemoveAll(c.Dir)
}

// Collect runs cryo for the given dataset over the inclusive block range
// [from, to], restricting output to the supplied columns (empty = cryo
// defaults). cryo's --blocks end is exclusive, so we pass to+1. The returned
// Collection must be Closed by the caller to clean up disk.
func (r *Runner) Collect(ctx context.Context, dataset string, from, to uint64, columns []string) (*Collection, error) {
	if to < from {
		return nil, fmt.Errorf("invalid range: to (%d) < from (%d)", to, from)
	}

	if err := os.MkdirAll(r.cfg.OutputDir, 0o750); err != nil {
		return nil, fmt.Errorf("failed to create cryo output dir: %w", err)
	}

	dir, err := os.MkdirTemp(r.cfg.OutputDir, fmt.Sprintf("%s-%d-%d-", dataset, from, to))
	if err != nil {
		return nil, fmt.Errorf("failed to create cryo temp dir: %w", err)
	}

	args := []string{
		dataset,
		"--blocks", fmt.Sprintf("%d:%d", from, to+1),
		"--rpc", r.rpc,
		"--output-dir", dir,
		"--no-report",
		"--overwrite",
		"--hex",
	}

	// --compression is variadic ("name [level]"); it must precede the variadic
	// --columns so clap does not conflate their values.
	if comp := strings.Fields(r.cfg.Compression); len(comp) > 0 {
		args = append(args, "--compression")
		args = append(args, comp...)
	}

	if r.cfg.RequestsPerSecond > 0 {
		args = append(args, "--requests-per-second", strconv.FormatUint(r.cfg.RequestsPerSecond, 10))
	}

	if r.cfg.MaxConcurrentChunks > 0 {
		args = append(args, "--max-concurrent-chunks", strconv.FormatUint(r.cfg.MaxConcurrentChunks, 10))
	}

	if len(columns) > 0 {
		args = append(args, "--columns")
		args = append(args, columns...)
	}

	var stderr bytes.Buffer

	//nolint:gosec // cryo binary path and args derive from operator config, not untrusted user input.
	cmd := exec.CommandContext(ctx, r.cfg.BinaryPath, args...)
	cmd.Stderr = &stderr

	r.log.WithFields(logrus.Fields{
		"dataset": dataset,
		"from":    from,
		"to":      to,
	}).Debug("Running cryo")

	if runErr := cmd.Run(); runErr != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("cryo %s [%d:%d] failed: %w: %s", dataset, from, to, runErr, stderr.String())
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.parquet"))
	if err != nil {
		_ = os.RemoveAll(dir)

		return nil, fmt.Errorf("failed to glob cryo output: %w", err)
	}

	return &Collection{Dir: dir, Files: files}, nil
}

// ReadParquet reads every supplied parquet file into a slice of T. T must be a
// struct with `parquet:"<column>"` tags matching the cryo column names.
func ReadParquet[T any](files []string) ([]T, error) {
	out := make([]T, 0)

	for _, file := range files {
		rows, err := parquet.ReadFile[T](file)
		if err != nil {
			return nil, fmt.Errorf("failed to read parquet %s: %w", file, err)
		}

		out = append(out, rows...)
	}

	return out, nil
}

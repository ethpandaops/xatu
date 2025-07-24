package persistence

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	testDbName = "xatu_test"
	testUser   = "postgres"
	testPass   = "password"
)

// TestContainer holds the test database container and client
type TestContainer struct {
	Container testcontainers.Container
	Client    *Client
	DB        *sql.DB
}

// setupTestContainer creates a new PostgreSQL test container with migrations applied
func setupTestContainer(t *testing.T) *TestContainer {
	t.Helper()
	ctx := context.Background()

	// Create PostgreSQL container
	postgresContainer, err := postgres.Run(ctx,
		"postgres:15-alpine",
		postgres.WithDatabase(testDbName),
		postgres.WithUsername(testUser),
		postgres.WithPassword(testPass),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second)),
	)
	require.NoError(t, err)

	// Get connection string
	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	// Create raw database connection for migrations
	db, err := sql.Open("postgres", connStr)
	require.NoError(t, err)

	// Apply migrations
	err = applyMigrations(t, db)
	require.NoError(t, err)

	// Create persistence client
	config := &Config{
		Enabled:          true,
		ConnectionString: connStr,
		DriverName:       DriverNamePostgres,
		MaxIdleConns:     2,
		MaxOpenConns:     5,
	}

	logger := logrus.New()
	logger.SetOutput(io.Discard) // Silence logs during tests

	client, err := NewClient(ctx, logger, config)
	require.NoError(t, err)

	err = client.Start(ctx)
	require.NoError(t, err)

	return &TestContainer{
		Container: postgresContainer,
		Client:    client,
		DB:        db,
	}
}

// applyMigrations applies the PostgreSQL migrations to the test database
func applyMigrations(t *testing.T, db *sql.DB) error {
	t.Helper()
	// Get the migrations directory
	migrationDir := filepath.Join("..", "..", "..", "migrations", "postgres")

	// Auto-discover migration files from directory
	entries, err := os.ReadDir(migrationDir)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	// Filter and sort migration files
	var migrationFiles []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filename := entry.Name()
		// Only include .up.sql files (skip .down.sql files)
		if strings.HasSuffix(filename, ".up.sql") {
			migrationFiles = append(migrationFiles, filename)
		}
	}

	// Sort migration files by name (which includes the numeric prefix)
	sort.Strings(migrationFiles)

	if len(migrationFiles) == 0 {
		t.Logf("No migration files found in %s", migrationDir)

		return nil
	}

	t.Logf("Applying %d migrations: %v", len(migrationFiles), migrationFiles)

	for _, filename := range migrationFiles {
		migrationPath := filepath.Join(migrationDir, filename)

		migrationSQL, err := os.ReadFile(migrationPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", filename, err)
		}

		if len(migrationSQL) == 0 {
			t.Logf("Skipping empty migration: %s", filename)

			continue
		}

		_, err = db.Exec(string(migrationSQL))
		if err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", filename, err)
		}

		t.Logf("Applied migration: %s", filename)
	}

	return nil
}

// cleanup tears down the test container
func (tc *TestContainer) cleanup(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if tc.Client != nil {
		err := tc.Client.Stop(ctx)
		assert.NoError(t, err)
	}

	if tc.DB != nil {
		err := tc.DB.Close()
		assert.NoError(t, err)
	}

	if tc.Container != nil {
		err := tc.Container.Terminate(ctx)
		assert.NoError(t, err)
	}
}

// createTestNodeRecord creates a test node.Record with default values
func createTestNodeRecord(enr string) *node.Record {
	now := time.Now()
	id := "v4"
	ip4 := "192.168.1.1"
	nodeID := "test-node-id-" + enr[len(enr)-10:]
	peerID := "test-peer-id-" + enr[len(enr)-10:]

	return &node.Record{
		Enr:       enr,
		Signature: &[]byte{0x01, 0x02, 0x03},
		Seq: func() *uint64 {
			v := uint64(123)

			return &v
		}(),
		CreateTime:              now,
		LastDialTime:            sql.NullTime{Time: now, Valid: true},
		ConsecutiveDialAttempts: 0,
		LastConnectTime:         sql.NullTime{Time: now, Valid: true},
		ID:                      &id,
		Secp256k1:               &[]byte{0x04, 0x05, 0x06},
		IP4:                     &ip4,
		TCP4: func() *uint32 {
			v := uint32(30303)

			return &v
		}(),
		UDP4: func() *uint32 {
			v := uint32(30303)

			return &v
		}(),
		NodeID: &nodeID,
		PeerID: &peerID,
	}
}

// createTestConsensusRecord creates a test node.Consensus with default values
func createTestConsensusRecord(enr string) *node.Consensus {
	now := time.Now()
	nodeID := "test-consensus-node-" + enr[len(enr)-10:]
	peerID := "test-consensus-peer-" + enr[len(enr)-10:]

	return &node.Consensus{
		Enr:            enr,
		NodeID:         nodeID,
		PeerID:         peerID,
		CreateTime:     now,
		Name:           "Lighthouse",
		ForkDigest:     []byte{0x01, 0x02, 0x03, 0x04},
		NextForkDigest: []byte{0x05, 0x06, 0x07, 0x08},
		FinalizedRoot:  []byte{0x09, 0x0a, 0x0b, 0x0c},
		FinalizedEpoch: []byte{0x0d, 0x0e, 0x0f, 0x10},
		HeadRoot:       []byte{0x11, 0x12, 0x13, 0x14},
		HeadSlot:       []byte{0x15, 0x16, 0x17, 0x18},
		CGC:            []byte{0x19, 0x1a, 0x1b, 0x1c},
		NetworkID:      "1", // Mainnet
	}
}

// createTestExecutionRecord creates a test node.Execution with default values
func createTestExecutionRecord(enr string) *node.Execution {
	return &node.Execution{
		Enr:             enr,
		Name:            "Geth",
		Capabilities:    "eth/66,eth/67",
		ProtocolVersion: "66",
		NetworkID:       "1", // Mainnet
		TotalDifficulty: "123456789",
		Head:            []byte{0x20, 0x21, 0x22, 0x23},
		Genesis:         []byte{0x24, 0x25, 0x26, 0x27},
		ForkIDHash:      []byte{0x28, 0x29, 0x2a, 0x2b},
		ForkIDNext:      "12345",
	}
}

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

// TestPersistenceIntegration tests all persistence operations using a single PostgreSQL container
func TestPersistenceIntegration(t *testing.T) {
	tc := setupTestContainer(t)
	defer tc.cleanup(t)

	ctx := context.Background()

	// Test basic node record operations
	t.Run("NodeRecordOperations", func(t *testing.T) {
		t.Run("InsertNodeRecords", func(t *testing.T) {
			// Create test records
			records := []*node.Record{
				createTestNodeRecord("enr:-IS4QHCYrYZbAKWCBRlAy5zzaDZXJBGkcnh4MHcBFZntXNFrdvJjX04jRzjzCBOonrkTfj499SZuOh8R33Ls8RRcy5wBgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQPKY0yuDUmEJHqpE8qU4GCe1-dqmJL67uGYDKQo4o3Wdg"),
				createTestNodeRecord("enr:-IS4QJ2d11eu6dC7E7LoXeLMgMP3kom1u3SE8esFSWvaHoo0dP1jg8O3-nx9ht-EO3CmG7L6OkHcMmoIh00IYWB92QABgmlkgnY0gmlwhH8AAAGJc2VjcDI1NmsxoQIB_c-jQMOXn0GJbVdpfCnGNRTt5wXPtyaxWJN5pbZ6sg"),
			}

			err := tc.Client.InsertNodeRecords(ctx, records)
			require.NoError(t, err)

			// Verify records exist by attempting to insert again (should do nothing due to ON CONFLICT)
			err = tc.Client.InsertNodeRecords(ctx, records)
			require.NoError(t, err)
		})

		t.Run("UpdateNodeRecord", func(t *testing.T) {
			// Create and insert a record first
			enr := "enr:-IS4QUpdateTest1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"
			record := createTestNodeRecord(enr)

			err := tc.Client.InsertNodeRecords(ctx, []*node.Record{record})
			require.NoError(t, err)

			// Update the record
			record.ConsecutiveDialAttempts = 5
			newSeq := uint64(456)
			record.Seq = &newSeq

			err = tc.Client.UpdateNodeRecord(ctx, record)
			require.NoError(t, err)
		})

		t.Run("CheckoutStalledExecutionNodeRecords", func(t *testing.T) {
			// Create records without eth2 field (execution nodes)
			records := []*node.Record{
				createTestNodeRecord("enr:-IS4QExecTest1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"),
				createTestNodeRecord("enr:-IS4QExecTest2-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"),
			}

			// Remove eth2 field to make them execution nodes and make them stalled
			for _, record := range records {
				record.ETH2 = nil
				record.LastDialTime = sql.NullTime{Valid: false}    // Make them stalled (NULL)
				record.LastConnectTime = sql.NullTime{Valid: false} // Make them stalled (NULL)
				record.ConsecutiveDialAttempts = 0
			}

			err := tc.Client.InsertNodeRecords(ctx, records)
			require.NoError(t, err)

			// Check out stalled records
			stalledRecords, err := tc.Client.CheckoutStalledExecutionNodeRecords(ctx, 10)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(stalledRecords), 2, "Should find at least 2 stalled execution records")

			// Verify the records are execution nodes (eth2 is NULL)
			for _, record := range stalledRecords {
				assert.Nil(t, record.ETH2, "Execution nodes should not have eth2 field")
			}
		})

		t.Run("CheckoutStalledConsensusNodeRecords", func(t *testing.T) {
			// Create records with eth2 field (consensus nodes)
			records := []*node.Record{
				createTestNodeRecord("enr:-IS4QConsTest1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"),
				createTestNodeRecord("enr:-IS4QConsTest2-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqrstuvwxyz"),
			}

			// Set eth2 field and make them stalled
			eth2Key := []byte{0x01, 0x02, 0x03, 0x04}
			for _, record := range records {
				record.ETH2 = &eth2Key
				record.LastDialTime = sql.NullTime{Valid: false}    // Make them stalled (NULL)
				record.LastConnectTime = sql.NullTime{Valid: false} // Make them stalled (NULL)
				record.ConsecutiveDialAttempts = 0
			}

			err := tc.Client.InsertNodeRecords(ctx, records)
			require.NoError(t, err)

			// Check out stalled records
			stalledRecords, err := tc.Client.CheckoutStalledConsensusNodeRecords(ctx, 10)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(stalledRecords), 2, "Should find at least 2 stalled consensus records")

			// Verify the records are consensus nodes (eth2 is NOT NULL)
			for _, record := range stalledRecords {
				assert.NotNil(t, record.ETH2, "Consensus nodes should have eth2 field")
			}
		})
	})

	// Test consensus record operations
	t.Run("NodeRecordConsensusOperations", func(t *testing.T) {
		t.Run("InsertAndListNodeRecordConsensus", func(t *testing.T) {
			// First insert the parent node record
			enr := "enr:-IS4QConsensusOpTest-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmnopqr"
			nodeRecord := createTestNodeRecord(enr)
			err := tc.Client.InsertNodeRecords(ctx, []*node.Record{nodeRecord})
			require.NoError(t, err)

			// Create and insert consensus record
			consensusRecord := createTestConsensusRecord(enr)
			err = tc.Client.InsertNodeRecordConsensus(ctx, consensusRecord)
			require.NoError(t, err)

			// List consensus records
			records, err := tc.Client.ListNodeRecordConsensus(ctx, []uint64{1}, nil, 10)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(records), 1)

			// Verify the record data
			found := false
			for _, record := range records {
				if record.Enr == enr {
					assert.Equal(t, "Lighthouse", record.Name)
					assert.Equal(t, "1", record.NetworkID)
					found = true

					break
				}
			}
			assert.True(t, found, "Consensus record not found in list")
		})

		t.Run("ListNodeRecordConsensusWithFilters", func(t *testing.T) {
			// Insert multiple consensus records with different network IDs and fork digests
			testData := []struct {
				enr        string
				networkID  string
				forkDigest []byte
			}{
				{"enr:-IS4QFilter1-networkID1-fork1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefgh", "1", []byte{0x01, 0x01, 0x01, 0x01}},
				{"enr:-IS4QFilter2-networkID1-fork2-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefgh", "1", []byte{0x02, 0x02, 0x02, 0x02}},
				{"enr:-IS4QFilter3-networkID2-fork1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefgh", "2", []byte{0x01, 0x01, 0x01, 0x01}},
			}

			for _, td := range testData {
				// Insert node record first
				nodeRecord := createTestNodeRecord(td.enr)
				err := tc.Client.InsertNodeRecords(ctx, []*node.Record{nodeRecord})
				require.NoError(t, err)

				// Insert consensus record
				consensusRecord := createTestConsensusRecord(td.enr)
				consensusRecord.NetworkID = td.networkID
				consensusRecord.ForkDigest = td.forkDigest
				err = tc.Client.InsertNodeRecordConsensus(ctx, consensusRecord)
				require.NoError(t, err)
			}

			// Test filtering by network ID
			records, err := tc.Client.ListNodeRecordConsensus(ctx, []uint64{1}, nil, 10)
			require.NoError(t, err)
			networkID1Count := 0
			for _, record := range records {
				if record.NetworkID == "1" {
					networkID1Count++
				}
			}
			assert.GreaterOrEqual(t, networkID1Count, 2)

			// Test filtering by fork digest
			records, err = tc.Client.ListNodeRecordConsensus(ctx, nil, [][]byte{{0x01, 0x01, 0x01, 0x01}}, 10)
			require.NoError(t, err)
			forkDigest1Count := 0
			for _, record := range records {
				if len(record.ForkDigest) == 4 && record.ForkDigest[0] == 0x01 {
					forkDigest1Count++
				}
			}
			assert.GreaterOrEqual(t, forkDigest1Count, 2)
		})

		t.Run("BulkInsertNodeRecordConsensus", func(t *testing.T) {
			// Create test data for bulk insert
			bulkTestData := []struct {
				enr       string
				networkID string
				name      string
			}{
				{"enr:-IS4QBulk1-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghij", "1", "Prysm"},
				{"enr:-IS4QBulk2-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghij", "5", "Lighthouse"},
				{"enr:-IS4QBulk3-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghij", "1", "Teku"},
				{"enr:-IS4QBulk4-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghij", "5", "Nimbus"},
				{"enr:-IS4QBulk5-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghij", "1", "Lodestar"},
			}

			// First insert parent node records
			nodeRecords := make([]*node.Record, len(bulkTestData))
			for i, td := range bulkTestData {
				nodeRecords[i] = createTestNodeRecord(td.enr)
			}
			err := tc.Client.InsertNodeRecords(ctx, nodeRecords)
			require.NoError(t, err)

			// Create consensus records for bulk insert
			consensusRecords := make([]*node.Consensus, len(bulkTestData))
			for i, td := range bulkTestData {
				consensusRecord := createTestConsensusRecord(td.enr)
				consensusRecord.NetworkID = td.networkID
				consensusRecord.Name = td.name
				consensusRecords[i] = consensusRecord
			}

			// Perform bulk insert
			err = tc.Client.BulkInsertNodeRecordConsensus(ctx, consensusRecords)
			require.NoError(t, err)

			// Verify all records were inserted
			allRecords, err := tc.Client.ListNodeRecordConsensus(ctx, []uint64{1, 5}, nil, 100)
			require.NoError(t, err)

			// Count inserted records by checking ENR prefix
			bulkRecordCount := 0
			clientNames := make(map[string]int)
			for _, record := range allRecords {
				if strings.HasPrefix(record.Enr, "enr:-IS4QBulk") {
					bulkRecordCount++
					clientNames[record.Name]++
				}
			}

			assert.Equal(t, len(bulkTestData), bulkRecordCount, "All bulk records should be inserted")
			assert.Equal(t, 1, clientNames["Prysm"])
			assert.Equal(t, 1, clientNames["Lighthouse"])
			assert.Equal(t, 1, clientNames["Teku"])
			assert.Equal(t, 1, clientNames["Nimbus"])
			assert.Equal(t, 1, clientNames["Lodestar"])

			// Test bulk insert with empty slice (should not error)
			err = tc.Client.BulkInsertNodeRecordConsensus(ctx, []*node.Consensus{})
			require.NoError(t, err)

			// Test bulk insert with records that reference non-existent ENR (should fail due to foreign key)
			invalidRecords := []*node.Consensus{
				createTestConsensusRecord("enr:-IS4QInvalid-consensus-record-that-does-not-exist-in-node-record-table-ABCDEFGHIJKLMNOPQRSTUVWXYZ"),
			}
			err = tc.Client.BulkInsertNodeRecordConsensus(ctx, invalidRecords)
			assert.Error(t, err, "Should fail due to foreign key constraint")

			// Test bulk insert with more than 500 records (should work due to batching)
			t.Run("LargeBatch", func(t *testing.T) {
				// Create 600 records to test batching
				largeENRs := make([]string, 600)
				for i := 0; i < 600; i++ {
					largeENRs[i] = fmt.Sprintf("enr:-IS4QLarge%04d-consensus-record-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abc", i)
				}

				// First insert parent node records
				largeNodeRecords := make([]*node.Record, len(largeENRs))
				for i, enr := range largeENRs {
					largeNodeRecords[i] = createTestNodeRecord(enr)
				}

				// Insert node records in batches to avoid overwhelming the test
				batchSize := 100
				for i := 0; i < len(largeNodeRecords); i += batchSize {
					end := i + batchSize
					if end > len(largeNodeRecords) {
						end = len(largeNodeRecords)
					}
					err := tc.Client.InsertNodeRecords(ctx, largeNodeRecords[i:end])
					require.NoError(t, err)
				}

				// Create consensus records
				largeConsensusRecords := make([]*node.Consensus, len(largeENRs))
				for i, enr := range largeENRs {
					consensusRecord := createTestConsensusRecord(enr)
					consensusRecord.Name = fmt.Sprintf("Client-%d", i%5) // Vary the client names
					largeConsensusRecords[i] = consensusRecord
				}

				// Perform bulk insert (should handle batching internally)
				err := tc.Client.BulkInsertNodeRecordConsensus(ctx, largeConsensusRecords)
				require.NoError(t, err)

				// Verify all records were inserted
				allRecords, err := tc.Client.ListNodeRecordConsensus(ctx, []uint64{1}, nil, 1000)
				require.NoError(t, err)

				// Count large batch records
				largeBatchCount := 0
				for _, record := range allRecords {
					if strings.HasPrefix(record.Enr, "enr:-IS4QLarge") {
						largeBatchCount++
					}
				}

				assert.Equal(t, 600, largeBatchCount, "All 600 records should be inserted via batching")
			})
		})
	})

	// Test execution record operations
	t.Run("NodeRecordExecutionOperations", func(t *testing.T) {
		t.Run("InsertAndListNodeRecordExecution", func(t *testing.T) {
			// First insert the parent node record
			enr := "enr:-IS4QExecutionOpTest-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890abcdefghijklmn"
			nodeRecord := createTestNodeRecord(enr)
			err := tc.Client.InsertNodeRecords(ctx, []*node.Record{nodeRecord})
			require.NoError(t, err)

			// Create and insert execution record
			executionRecord := createTestExecutionRecord(enr)
			err = tc.Client.InsertNodeRecordExecution(ctx, executionRecord)
			require.NoError(t, err)

			// List execution records
			records, err := tc.Client.ListNodeRecordExecutions(ctx, []uint64{1}, nil, 10)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, len(records), 1)

			// Verify the record data
			found := false
			for _, record := range records {
				if record.Enr == enr {
					assert.Equal(t, "Geth", record.Name)
					assert.Equal(t, "1", record.NetworkID)
					assert.Equal(t, "eth/66,eth/67", record.Capabilities)
					found = true

					break
				}
			}
			assert.True(t, found, "Execution record not found in list")
		})

		t.Run("ListNodeRecordExecutionWithFilters", func(t *testing.T) {
			// Insert multiple execution records with different network IDs and fork ID hashes
			testData := []struct {
				enr        string
				networkID  string
				forkIDHash []byte
			}{
				{"enr:-IS4QExecFilter1-networkID1-fork1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789", "1", []byte{0x11, 0x11, 0x11, 0x11}},
				{"enr:-IS4QExecFilter2-networkID1-fork2-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789", "1", []byte{0x22, 0x22, 0x22, 0x22}},
				{"enr:-IS4QExecFilter3-networkID2-fork1-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ123456789", "2", []byte{0x11, 0x11, 0x11, 0x11}},
			}

			for _, td := range testData {
				// Insert node record first
				nodeRecord := createTestNodeRecord(td.enr)
				err := tc.Client.InsertNodeRecords(ctx, []*node.Record{nodeRecord})
				require.NoError(t, err)

				// Insert execution record
				executionRecord := createTestExecutionRecord(td.enr)
				executionRecord.NetworkID = td.networkID
				executionRecord.ForkIDHash = td.forkIDHash
				err = tc.Client.InsertNodeRecordExecution(ctx, executionRecord)
				require.NoError(t, err)
			}

			// Test filtering by network ID
			records, err := tc.Client.ListNodeRecordExecutions(ctx, []uint64{1}, nil, 10)
			require.NoError(t, err)
			networkID1Count := 0
			for _, record := range records {
				if record.NetworkID == "1" {
					networkID1Count++
				}
			}
			assert.GreaterOrEqual(t, networkID1Count, 2)

			// Test filtering by fork ID hash
			records, err = tc.Client.ListNodeRecordExecutions(ctx, nil, [][]byte{{0x11, 0x11, 0x11, 0x11}}, 10)
			require.NoError(t, err)
			forkHash1Count := 0
			for _, record := range records {
				if len(record.ForkIDHash) == 4 && record.ForkIDHash[0] == 0x11 {
					forkHash1Count++
				}
			}
			assert.GreaterOrEqual(t, forkHash1Count, 2)
		})
	})

	// Test database constraints
	t.Run("DatabaseConstraints", func(t *testing.T) {
		t.Run("ForeignKeyConstraintConsensus", func(t *testing.T) {
			// Try to insert consensus record without parent node record
			consensusRecord := createTestConsensusRecord("enr:-IS4QNonExistentENR-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

			err := tc.Client.InsertNodeRecordConsensus(ctx, consensusRecord)
			assert.Error(t, err, "Should fail due to foreign key constraint")
		})

		t.Run("ForeignKeyConstraintExecution", func(t *testing.T) {
			// Try to insert execution record without parent node record
			executionRecord := createTestExecutionRecord("enr:-IS4QNonExistentENR2-abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

			err := tc.Client.InsertNodeRecordExecution(ctx, executionRecord)
			assert.Error(t, err, "Should fail due to foreign key constraint")
		})
	})
}

// setupTestContainer creates a new PostgreSQL test container with migrations applied
func setupTestContainer(t *testing.T) *TestContainer {
	t.Helper()
	ctx := context.Background()

	// Create PostgreSQL container
	t.Logf("Creating PostgreSQL container...")
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
	t.Logf("PostgreSQL container created successfully")

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

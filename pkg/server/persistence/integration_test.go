package persistence

import (
	"context"
	"database/sql"
	"testing"

	"github.com/ethpandaops/xatu/pkg/server/persistence/node"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

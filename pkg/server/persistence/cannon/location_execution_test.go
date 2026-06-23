package cannon

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func TestLocation_ExecutionCanonicalBlock_RoundTrip(t *testing.T) {
	msg := &xatu.CannonLocation{
		NetworkId: "1",
		Type:      xatu.CannonType_EXECUTION_CANONICAL_BLOCK,
		Data: &xatu.CannonLocation_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: &xatu.CannonLocationExecutionCanonicalBlock{
				BackfillingBlockMarker: &xatu.BackfillingBlockMarker{
					FinalizedBlock: 22000000,
					BackfillBlock:  21999900,
				},
			},
		},
	}

	l := &Location{}
	require.NoError(t, l.Marshal(msg))

	assert.Equal(t, "EXECUTION_CANONICAL_BLOCK", l.Type)
	assert.Equal(t, "1", l.NetworkID)
	assert.NotEmpty(t, l.Value)

	out, err := l.Unmarshal()
	require.NoError(t, err)

	assert.Equal(t, xatu.CannonType_EXECUTION_CANONICAL_BLOCK, out.GetType())

	marker := out.GetExecutionCanonicalBlock().GetBackfillingBlockMarker()
	assert.Equal(t, uint64(22000000), marker.GetFinalizedBlock())
	assert.Equal(t, int64(21999900), marker.GetBackfillBlock())
}

func TestLocation_ExecutionCanonicalTransaction_RoundTrip(t *testing.T) {
	msg := &xatu.CannonLocation{
		NetworkId: "1",
		Type:      xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION,
		Data: &xatu.CannonLocation_ExecutionCanonicalTransaction{
			ExecutionCanonicalTransaction: &xatu.CannonLocationExecutionCanonicalTransaction{
				BackfillingBlockMarker: &xatu.BackfillingBlockMarker{
					FinalizedBlock: 22000000,
					BackfillBlock:  21999900,
				},
			},
		},
	}

	l := &Location{}
	require.NoError(t, l.Marshal(msg))

	assert.Equal(t, "EXECUTION_CANONICAL_TRANSACTION", l.Type)

	out, err := l.Unmarshal()
	require.NoError(t, err)

	assert.Equal(t, xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION, out.GetType())

	marker := out.GetExecutionCanonicalTransaction().GetBackfillingBlockMarker()
	assert.Equal(t, uint64(22000000), marker.GetFinalizedBlock())
	assert.Equal(t, int64(21999900), marker.GetBackfillBlock())
}

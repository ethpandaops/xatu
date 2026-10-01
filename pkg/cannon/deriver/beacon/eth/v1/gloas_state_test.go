package v1

import (
	"testing"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec/bellatrix"
	"github.com/ethpandaops/go-eth2-client/spec/gloas"
	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"github.com/ethpandaops/xatu/pkg/cannon/ethereum"
	"github.com/ethpandaops/xatu/pkg/clickhouse/route"
	routeall "github.com/ethpandaops/xatu/pkg/clickhouse/route/all"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

const (
	testSlotsPerEpoch = 32
	testPTCSize       = 512
	testEpoch         = phase0.Epoch(100)
	testForkEpoch     = phase0.Epoch(90)
)

var testGenesis = time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

func newTestDeriver() *gloasStateDeriver {
	return &gloasStateDeriver{
		clientMeta: &xatu.ClientMeta{
			Name: "cannon",
			Ethereum: &xatu.ClientMeta_Ethereum{
				Network: &xatu.ClientMeta_Ethereum_Network{Name: "testnet", Id: 1},
			},
		},
		epochData: func(epoch phase0.Epoch) *xatu.EpochV2 {
			return &xatu.EpochV2{
				Number:        wrapperspb.UInt64(uint64(epoch)),
				StartDateTime: timestamppb.New(testGenesis.Add(time.Duration(uint64(epoch)*testSlotsPerEpoch*12) * time.Second)),
			}
		},
		slotData: func(slot phase0.Slot) *xatu.SlotV2 {
			return &xatu.SlotV2{
				Number:        wrapperspb.UInt64(uint64(slot)),
				StartDateTime: timestamppb.New(testGenesis.Add(time.Duration(uint64(slot)*12) * time.Second)),
			}
		},
	}
}

// newTestEpochState returns a state at the first slot of testEpoch. Every cell
// of the ptc window is distinct: window index*1000+position.
func newTestEpochState() *ethereum.GloasEpochState {
	window := make([][]phase0.ValidatorIndex, 96)
	for i := range window {
		window[i] = make([]phase0.ValidatorIndex, testPTCSize)

		for j := range window[i] {
			window[i][j] = phase0.ValidatorIndex(i*1000 + j)
		}
	}

	payments := make([]*gloas.BuilderPendingPayment, 2*testSlotsPerEpoch)
	for i := range payments {
		payments[i] = &gloas.BuilderPendingPayment{Withdrawal: &gloas.BuilderPendingWithdrawal{}}
	}

	firstSlot := phase0.Slot(uint64(testEpoch) * testSlotsPerEpoch)

	return &ethereum.GloasEpochState{
		Slot:                         firstSlot,
		LatestBlockHeaderSlot:        firstSlot,
		FinalizedEpoch:               testEpoch - 2,
		PTCWindow:                    window,
		BuilderPendingPayments:       payments,
		ExecutionPayloadAvailability: make([]byte, 1024),
	}
}

func newTestInput(state *ethereum.GloasEpochState) *gloasStateInput {
	return &gloasStateInput{
		epoch:         testEpoch,
		stateID:       "3200",
		state:         state,
		slotsPerEpoch: testSlotsPerEpoch,
		forkEpoch:     testForkEpoch,
	}
}

func TestPtcMemberDeriveEpoch(t *testing.T) {
	d := &BeaconStatePtcMemberDeriver{newTestDeriver()}

	events, err := d.deriveEpoch(newTestInput(newTestEpochState()))
	require.NoError(t, err)
	require.Len(t, events, testSlotsPerEpoch*testPTCSize)

	firstSlot := uint64(testEpoch) * testSlotsPerEpoch

	// The epoch's own committees (window[32:64]) are emitted slot by slot, each
	// in committee order.
	for i, event := range events {
		slot := firstSlot + uint64(i/testPTCSize)
		position := uint64(i % testPTCSize)
		payload := event.GetEthV1BeaconStatePtcMember()
		extra := event.GetMeta().GetClient().GetEthV1BeaconStatePtcMember()

		require.Equal(t, xatu.Event_BEACON_API_ETH_V1_BEACON_STATE_PTC_MEMBER, event.GetEvent().GetName())
		require.Equal(t, slot, extra.GetSlot().GetNumber().GetValue())
		require.Equal(t, uint64(testEpoch), extra.GetEpoch().GetNumber().GetValue())
		require.Equal(t, "3200", extra.GetStateId())
		require.Equal(t, position, payload.GetPosition().GetValue())
		require.Equal(t, uint64(32+i/testPTCSize)*1000+position, payload.GetValidatorIndex().GetValue())
	}
}

func TestPtcMemberDeriveEpochRejectsEmptyCommittee(t *testing.T) {
	state := newTestEpochState()
	state.PTCWindow[40] = nil

	_, err := (&BeaconStatePtcMemberDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.Error(t, err)
}

func TestBuilderDeriveEpoch(t *testing.T) {
	state := newTestEpochState()
	state.FinalizedEpoch = 50
	state.Builders = []*gloas.Builder{
		{PublicKey: phase0.BLSPubKey{1}, Version: 0, ExecutionAddress: bellatrix.ExecutionAddress{2}, Balance: 40, DepositEpoch: 10, WithdrawableEpoch: farFutureEpoch},
		{PublicKey: phase0.BLSPubKey{3}, Version: 1, ExecutionAddress: bellatrix.ExecutionAddress{4}, Balance: 0, DepositEpoch: 50, WithdrawableEpoch: farFutureEpoch},
		{PublicKey: phase0.BLSPubKey{5}, Version: 0, ExecutionAddress: bellatrix.ExecutionAddress{6}, Balance: 7, DepositEpoch: 10, WithdrawableEpoch: 90},
	}

	events, err := (&BeaconStateBuilderDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.NoError(t, err)
	require.Len(t, events, 3)

	wantStatus := []string{"active", "pending", "exited"}

	for i, event := range events {
		payload := event.GetEthV1BeaconStateBuilder()
		extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilder()

		assert.Equal(t, uint64(i), payload.GetIndex().GetValue())
		assert.Equal(t, wantStatus[i], payload.GetStatus())
		assert.Equal(t, uint64(testEpoch), extra.GetEpoch().GetNumber().GetValue())
		assert.Equal(t, "3200", extra.GetStateId())
	}

	first := events[0].GetEthV1BeaconStateBuilder()
	assert.Equal(t, "0x01"+repeat("00", 47), first.GetPubkey())
	assert.Equal(t, "0x02"+repeat("00", 19), first.GetExecutionAddress())
	assert.Equal(t, uint64(40), first.GetBalance().GetValue())
	assert.Equal(t, uint64(10), first.GetDepositEpoch().GetValue())
	assert.Equal(t, uint64(farFutureEpoch), first.GetWithdrawableEpoch().GetValue())
	assert.Equal(t, uint32(1), events[1].GetEthV1BeaconStateBuilder().GetVersion().GetValue())
}

func TestBuilderDeriveEpochWithoutBuilders(t *testing.T) {
	events, err := (&BeaconStateBuilderDeriver{newTestDeriver()}).deriveEpoch(newTestInput(newTestEpochState()))
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestBuilderPendingPaymentDeriveEpoch(t *testing.T) {
	state := newTestEpochState()
	// Previous epoch's slot 5 and the current epoch's slot 2.
	state.BuilderPendingPayments[5] = &gloas.BuilderPendingPayment{
		Weight:        100,
		Withdrawal:    &gloas.BuilderPendingWithdrawal{FeeRecipient: bellatrix.ExecutionAddress{9}, Amount: 200, BuilderIndex: 3},
		ProposerIndex: 77,
	}
	state.BuilderPendingPayments[testSlotsPerEpoch+2] = &gloas.BuilderPendingPayment{
		Withdrawal: &gloas.BuilderPendingWithdrawal{Amount: 1, BuilderIndex: 0},
	}
	// Only a proposer index is still not empty.
	state.BuilderPendingPayments[testSlotsPerEpoch+3] = &gloas.BuilderPendingPayment{
		Withdrawal:    &gloas.BuilderPendingWithdrawal{},
		ProposerIndex: 1,
	}

	events, err := (&BeaconStateBuilderPendingPaymentDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.NoError(t, err)
	require.Len(t, events, 3)

	firstSlot := uint64(testEpoch) * testSlotsPerEpoch

	first := events[0].GetEthV1BeaconStateBuilderPendingPayment()
	firstExtra := events[0].GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingPayment()

	assert.Equal(t, uint64(5), first.GetPaymentIndex().GetValue())
	assert.Equal(t, uint64(100), first.GetWeight().GetValue())
	assert.Equal(t, "0x09"+repeat("00", 19), first.GetFeeRecipient())
	assert.Equal(t, uint64(200), first.GetAmount().GetValue())
	assert.Equal(t, uint64(3), first.GetBuilderIndex().GetValue())
	assert.Equal(t, uint64(77), first.GetProposerIndex().GetValue())
	assert.Equal(t, firstSlot-testSlotsPerEpoch+5, firstExtra.GetSlot().GetNumber().GetValue(), "previous epoch half")
	assert.Equal(t, uint64(testEpoch), firstExtra.GetEpoch().GetNumber().GetValue())

	secondExtra := events[1].GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingPayment()
	assert.Equal(t, uint64(testSlotsPerEpoch+2), events[1].GetEthV1BeaconStateBuilderPendingPayment().GetPaymentIndex().GetValue())
	assert.Equal(t, firstSlot+2, secondExtra.GetSlot().GetNumber().GetValue(), "current epoch half")

	assert.Equal(t, uint64(1), events[2].GetEthV1BeaconStateBuilderPendingPayment().GetProposerIndex().GetValue())
}

func TestBuilderPendingPaymentDeriveEpochSkipsEmptyEntries(t *testing.T) {
	events, err := (&BeaconStateBuilderPendingPaymentDeriver{newTestDeriver()}).deriveEpoch(newTestInput(newTestEpochState()))
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestBuilderPendingPaymentDeriveEpochRejectsShortVector(t *testing.T) {
	state := newTestEpochState()
	state.BuilderPendingPayments = state.BuilderPendingPayments[:10]

	_, err := (&BeaconStateBuilderPendingPaymentDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.Error(t, err)
}

func TestBuilderPendingPaymentSlot(t *testing.T) {
	slot, ok := builderPendingPaymentSlot(100, 0, 32)
	require.True(t, ok)
	assert.Equal(t, phase0.Slot(99*32), slot)

	slot, ok = builderPendingPaymentSlot(100, 31, 32)
	require.True(t, ok)
	assert.Equal(t, phase0.Slot(100*32-1), slot)

	slot, ok = builderPendingPaymentSlot(100, 32, 32)
	require.True(t, ok)
	assert.Equal(t, phase0.Slot(100*32), slot)

	slot, ok = builderPendingPaymentSlot(100, 63, 32)
	require.True(t, ok)
	assert.Equal(t, phase0.Slot(101*32-1), slot)

	_, ok = builderPendingPaymentSlot(0, 3, 32)
	assert.False(t, ok, "no previous epoch before epoch 0")
}

func TestBuilderPendingWithdrawalDeriveEpoch(t *testing.T) {
	state := newTestEpochState()
	state.BuilderPendingWithdrawals = []*gloas.BuilderPendingWithdrawal{
		{FeeRecipient: bellatrix.ExecutionAddress{7}, Amount: 11, BuilderIndex: 2},
		{FeeRecipient: bellatrix.ExecutionAddress{8}, Amount: 12, BuilderIndex: 3},
	}

	events, err := (&BeaconStateBuilderPendingWithdrawalDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.NoError(t, err)
	require.Len(t, events, 2)

	for i, event := range events {
		payload := event.GetEthV1BeaconStateBuilderPendingWithdrawal()
		extra := event.GetMeta().GetClient().GetEthV1BeaconStateBuilderPendingWithdrawal()

		assert.Equal(t, uint64(i), extra.GetPositionInQueue().GetValue())
		assert.Equal(t, uint64(11+i), payload.GetAmount().GetValue())
		assert.Equal(t, uint64(2+i), payload.GetBuilderIndex().GetValue())
		assert.Equal(t, uint64(testEpoch), extra.GetEpoch().GetNumber().GetValue())
	}

	assert.Equal(t, "0x07"+repeat("00", 19), events[0].GetEthV1BeaconStateBuilderPendingWithdrawal().GetFeeRecipient())
}

func TestExecutionPayloadAvailabilityDeriveEpoch(t *testing.T) {
	state := newTestEpochState()

	// Epoch 98 is emitted for a state in epoch 100. Mark every other slot of it
	// available.
	firstSlot := uint64(testEpoch-availabilityEpochLag) * testSlotsPerEpoch
	for i := uint64(0); i < testSlotsPerEpoch; i += 2 {
		bit := (firstSlot + i) % 8192
		state.ExecutionPayloadAvailability[bit/8] |= 1 << (bit % 8)
	}

	events, err := (&BeaconStateExecutionPayloadAvailabilityDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.NoError(t, err)
	require.Len(t, events, testSlotsPerEpoch)

	for i, event := range events {
		extra := event.GetMeta().GetClient().GetEthV1BeaconStateExecutionPayloadAvailability()

		assert.Equal(t, firstSlot+uint64(i), extra.GetSlot().GetNumber().GetValue())
		assert.Equal(t, uint64(testEpoch-availabilityEpochLag), extra.GetEpoch().GetNumber().GetValue())
		assert.Equal(t, "3200", extra.GetStateId())
		assert.Equal(t, i%2 == 0, event.GetEthV1BeaconStateExecutionPayloadAvailability().GetAvailable().GetValue(), "slot %d", firstSlot+uint64(i))
	}
}

func TestExecutionPayloadAvailabilityDeriveEpochBeforeFork(t *testing.T) {
	d := &BeaconStateExecutionPayloadAvailabilityDeriver{newTestDeriver()}

	// Epoch 91 reads epoch 89, which predates the fork at epoch 90.
	in := newTestInput(newTestEpochState())
	in.epoch = testForkEpoch + 1

	events, err := d.deriveEpoch(in)
	require.NoError(t, err)
	assert.Empty(t, events)

	// Epoch 92 reads the fork epoch, whose first slot is left out.
	in.epoch = testForkEpoch + availabilityEpochLag
	in.state.Slot = phase0.Slot(uint64(in.epoch) * testSlotsPerEpoch)
	in.state.LatestBlockHeaderSlot = in.state.Slot

	events, err = d.deriveEpoch(in)
	require.NoError(t, err)
	require.Len(t, events, testSlotsPerEpoch-1)
	assert.Equal(t, uint64(testForkEpoch)*testSlotsPerEpoch+1,
		events[0].GetMeta().GetClient().GetEthV1BeaconStateExecutionPayloadAvailability().GetSlot().GetNumber().GetValue())

	// Nothing to emit before the first epoch that has two epochs behind it.
	in.epoch = 1
	in.forkEpoch = 0
	events, err = d.deriveEpoch(in)
	require.NoError(t, err)
	assert.Empty(t, events)
}

func TestExecutionPayloadAvailabilityDeriveEpochRejectsUnsettledBits(t *testing.T) {
	state := newTestEpochState()
	// No block was applied since before the emitted epoch ended.
	state.LatestBlockHeaderSlot = phase0.Slot(uint64(testEpoch-availabilityEpochLag)*testSlotsPerEpoch + 5)

	_, err := (&BeaconStateExecutionPayloadAvailabilityDeriver{newTestDeriver()}).deriveEpoch(newTestInput(state))
	require.Error(t, err)
}

// TestDerivedEventsFlattenThroughRoutes feeds every deriver's events through
// the ClickHouse route registered for them, so the event contract between the
// cannon and the routes cannot drift apart.
func TestDerivedEventsFlattenThroughRoutes(t *testing.T) {
	base := newTestDeriver()

	state := newTestEpochState()
	state.Builders = []*gloas.Builder{{Balance: 5, WithdrawableEpoch: farFutureEpoch}}
	state.BuilderPendingWithdrawals = []*gloas.BuilderPendingWithdrawal{{Amount: 3}}
	state.BuilderPendingPayments[4] = &gloas.BuilderPendingPayment{
		Weight:     1,
		Withdrawal: &gloas.BuilderPendingWithdrawal{Amount: 2, BuilderIndex: 1},
	}

	input := newTestInput(state)

	cases := []struct {
		derive func(*gloasStateInput) ([]*xatu.DecoratedEvent, error)
		table  string
		rows   int
	}{
		{(&BeaconStatePtcMemberDeriver{base}).deriveEpoch, "canonical_beacon_state_ptc_member", testSlotsPerEpoch * testPTCSize},
		{(&BeaconStateBuilderDeriver{base}).deriveEpoch, "canonical_beacon_state_builder", 1},
		{(&BeaconStateBuilderPendingPaymentDeriver{base}).deriveEpoch, "canonical_beacon_state_builder_pending_payment", 1},
		{(&BeaconStateBuilderPendingWithdrawalDeriver{base}).deriveEpoch, "canonical_beacon_state_builder_pending_withdrawal", 1},
		{(&BeaconStateExecutionPayloadAvailabilityDeriver{base}).deriveEpoch, "canonical_beacon_state_execution_payload_availability", testSlotsPerEpoch},
	}

	routes, err := routeall.All()
	require.NoError(t, err)

	byTable := make(map[string]route.Route, len(routes))
	for _, r := range routes {
		byTable[r.TableName()] = r
	}

	for _, tc := range cases {
		t.Run(tc.table, func(t *testing.T) {
			r, ok := byTable[tc.table]
			require.True(t, ok, "no route registered for %s", tc.table)

			events, err := tc.derive(input)
			require.NoError(t, err)
			require.Len(t, events, tc.rows)

			batch := r.NewBatch()

			for _, event := range events {
				require.True(t, r.ShouldProcess(event))
				require.NoError(t, batch.FlattenTo(event))
			}

			require.Equal(t, tc.rows, batch.Rows())

			for _, col := range batch.Input() {
				require.Equalf(t, tc.rows, col.Data.Rows(), "column %q", col.Name)
			}
		})
	}
}

func repeat(s string, n int) string {
	out := ""
	for i := 0; i < n; i++ {
		out += s
	}

	return out
}

package v1

import (
	"context"
	"fmt"
	"slices"

	"github.com/ethpandaops/go-eth2-client/spec/phase0"
	zcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	zphase0 "github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/zrnt/eth2/util/math"
)

// phase0AttesterReward is one validator's attestation reward components for an epoch in gwei, as get_attestation_deltas computes them.
type phase0AttesterReward struct {
	ValidatorIndex phase0.ValidatorIndex
	Head           int64
	Target         int64
	Source         int64
	// InclusionDelay is the attester's own inclusion reward; the proposer's share is in phase0EpochRewards.ProposerCredits.
	InclusionDelay uint64
	Inactivity     int64
}

// phase0Inclusion identifies the block that included attestations: its slot and the proposer the attestations name.
type phase0Inclusion struct {
	Slot          phase0.Slot
	ProposerIndex phase0.ValidatorIndex
}

// phase0EpochRewards is what the epoch transition after a phase0 state pays, read from the state at the last slot of its epoch.
type phase0EpochRewards struct {
	// StateEpoch is the epoch of the state; its transition rewards the attestations of StateEpoch-1.
	StateEpoch phase0.Epoch
	// Attesters holds every validator eligible for attestation rewards. Empty at genesis, whose transition pays nothing.
	Attesters []phase0AttesterReward
	// ProposerCredits is the inclusion reward in gwei each including block's proposer earns.
	ProposerCredits map[phase0Inclusion]uint64
	// Slashed maps every slashed validator to the whistleblower reward its slashing paid the block proposer, if it happened
	// during StateEpoch: the effective balance it is computed from only changes at epoch transitions.
	Slashed map[phase0.ValidatorIndex]uint64
}

// computePhase0EpochRewards runs the consensus spec's phase0 get_attestation_deltas on state and splits the
// inclusion delay deltas into the attester's reward and the including proposer's reward.
func computePhase0EpochRewards(ctx context.Context, spec *zcommon.Spec, state *zphase0.BeaconStateView) (*phase0EpochRewards, error) {
	epc, err := zcommon.NewEpochsContext(spec, state)
	if err != nil {
		return nil, fmt.Errorf("failed to build epochs context: %w", err)
	}

	vals, err := state.Validators()
	if err != nil {
		return nil, err
	}

	flats, err := zcommon.FlattenValidators(vals)
	if err != nil {
		return nil, err
	}

	currentEpoch := epc.CurrentEpoch.Epoch

	out := &phase0EpochRewards{
		StateEpoch:      phase0.Epoch(currentEpoch),
		ProposerCredits: map[phase0Inclusion]uint64{},
		Slashed:         map[phase0.ValidatorIndex]uint64{},
	}

	for i := range flats {
		if flats[i].Slashed {
			out.Slashed[phase0.ValidatorIndex(i)] = uint64(flats[i].EffectiveBalance) / uint64(spec.WHISTLEBLOWER_REWARD_QUOTIENT)
		}
	}

	// process_rewards_and_penalties skips the genesis epoch.
	if currentEpoch == zcommon.GENESIS_EPOCH {
		return out, nil
	}

	attesterData, err := zphase0.ComputeEpochAttesterData(ctx, spec, epc, flats, state)
	if err != nil {
		return nil, fmt.Errorf("failed to compute attester data: %w", err)
	}

	deltas, err := zphase0.AttestationRewardsAndPenalties(ctx, spec, epc, attesterData, state)
	if err != nil {
		return nil, fmt.Errorf("failed to compute attestation deltas: %w", err)
	}

	inclusions, err := earliestInclusions(spec, epc, state, len(flats))
	if err != nil {
		return nil, err
	}

	sqrtTotal := zcommon.Gwei(math.IntegerSquareroot(uint64(epc.TotalActiveStake)))
	attesterInclusion := make([]uint64, len(flats))
	proposerTotals := make([]uint64, len(flats))

	for i := range attesterData.Statuses {
		status := &attesterData.Statuses[i]
		if !status.Flags.HasMarkers(zphase0.PrevSourceAttester | zphase0.UnslashedAttester) {
			continue
		}

		included := inclusions[i]
		if !included.ok || included.proposer != status.AttestedProposer || included.delay != status.InclusionDelay {
			return nil, fmt.Errorf("validator %d: earliest inclusion does not match the attester data", i)
		}

		// get_inclusion_delay_deltas: the proposer earns base/PROPOSER_REWARD_QUOTIENT, the attester the rest divided by the delay.
		baseReward := attesterData.Flats[i].EffectiveBalance * zcommon.Gwei(spec.BASE_REWARD_FACTOR) / sqrtTotal / zcommon.BASE_REWARDS_PER_EPOCH
		proposerReward := baseReward / zcommon.Gwei(spec.PROPOSER_REWARD_QUOTIENT)

		attesterInclusion[i] = uint64((baseReward - proposerReward) / zcommon.Gwei(status.InclusionDelay))
		proposerTotals[status.AttestedProposer] += uint64(proposerReward)
		out.ProposerCredits[phase0Inclusion{Slot: included.slot, ProposerIndex: phase0.ValidatorIndex(status.AttestedProposer)}] += uint64(proposerReward)
	}

	out.Attesters = make([]phase0AttesterReward, 0, len(flats))

	for i := range attesterData.Statuses {
		if got, want := attesterInclusion[i]+proposerTotals[i], uint64(deltas.InclusionDelay.Rewards[i]); got != want {
			return nil, fmt.Errorf("validator %d: inclusion split %d does not add up to the inclusion delay reward %d", i, got, want)
		}

		if attesterData.Statuses[i].Flags&zphase0.EligibleAttester == 0 {
			continue
		}

		out.Attesters = append(out.Attesters, phase0AttesterReward{
			ValidatorIndex: phase0.ValidatorIndex(i),
			Head:           netDelta(deltas.Head, i),
			Target:         netDelta(deltas.Target, i),
			Source:         netDelta(deltas.Source, i),
			InclusionDelay: attesterInclusion[i],
			Inactivity:     netDelta(deltas.Inactivity, i),
		})
	}

	return out, nil
}

type earliestInclusion struct {
	ok       bool
	slot     phase0.Slot
	delay    zcommon.Slot
	proposer zcommon.ValidatorIndex
}

// earliestInclusions finds, per validator, the previous epoch attestation get_inclusion_delay_deltas rewards: the first one with the
// smallest inclusion delay. Its slot plus delay is the slot of the block whose proposer earns the proposer reward.
func earliestInclusions(spec *zcommon.Spec, epc *zcommon.EpochsContext, state *zphase0.BeaconStateView, count int) ([]earliestInclusion, error) {
	out := make([]earliestInclusion, count)

	atts, err := state.PreviousEpochAttestations()
	if err != nil {
		return nil, err
	}

	iter := atts.ReadonlyIter()
	participants := make([]zcommon.ValidatorIndex, 0, spec.MAX_VALIDATORS_PER_COMMITTEE)

	for {
		el, ok, err := iter.Next()
		if err != nil {
			return nil, err
		}

		if !ok {
			return out, nil
		}

		view, err := zphase0.AsPendingAttestation(el, nil)
		if err != nil {
			return nil, err
		}

		att, err := view.Raw()
		if err != nil {
			return nil, err
		}

		committee, err := epc.GetBeaconCommittee(att.Data.Slot, att.Data.Index)
		if err != nil {
			return nil, err
		}

		participants = append(participants[:0], committee...)
		participants = att.AggregationBits.FilterParticipants(participants)

		for _, p := range participants {
			if out[p].ok && out[p].delay <= att.InclusionDelay {
				continue
			}

			out[p] = earliestInclusion{
				ok:       true,
				slot:     phase0.Slot(att.Data.Slot + att.InclusionDelay),
				delay:    att.InclusionDelay,
				proposer: att.ProposerIndex,
			}
		}
	}
}

// phase0BlockReward is a phase0 block's proposer reward breakdown in gwei. Phase0 has no sync aggregate.
type phase0BlockReward struct {
	Slot              phase0.Slot
	ProposerIndex     phase0.ValidatorIndex
	Attestations      uint64
	ProposerSlashings uint64
	AttesterSlashings uint64
}

func (r *phase0BlockReward) Total() uint64 {
	return r.Attestations + r.ProposerSlashings + r.AttesterSlashings
}

// computePhase0BlockRewards attributes proposer rewards to the canonical blocks of one epoch, given in slot order,
// from the rewards read at the last slot of the previous epoch, this epoch and the next one.
// A block's attestation reward is paid by two transitions: the one ending its epoch (endOfEpoch) for previous epoch
// attestations, and the next one for its own epoch's attestations. next is nil when that transition is not phase0;
// Altair's first transition pays proposers nothing for phase0 attestations. Slashing rewards are paid in the block, for
// the validators slashed since the previous epoch's state; previous is nil for the genesis epoch.
func computePhase0BlockRewards(blocks []*phase0.BeaconBlock, slotsPerEpoch uint64, previous, endOfEpoch, next *phase0EpochRewards) ([]phase0BlockReward, error) {
	epoch := endOfEpoch.StateEpoch
	if (previous != nil && previous.StateEpoch+1 != epoch) || (next != nil && next.StateEpoch != epoch+1) {
		return nil, fmt.Errorf("rewards around state epoch %d are not for consecutive epochs", epoch)
	}

	inEpoch := func(slot phase0.Slot) bool { return uint64(slot)/slotsPerEpoch == uint64(epoch) }

	credits := map[phase0Inclusion]uint64{}

	for _, rewards := range []*phase0EpochRewards{endOfEpoch, next} {
		if rewards == nil {
			continue
		}

		for inclusion, gwei := range rewards.ProposerCredits {
			if inEpoch(inclusion.Slot) {
				credits[inclusion] += gwei
			}
		}
	}

	// A validator is slashed by the first operation in the epoch that names it; later ones skip it as no longer slashable.
	unclaimed := map[phase0.ValidatorIndex]uint64{}

	for index, reward := range endOfEpoch.Slashed {
		if previous != nil {
			if _, before := previous.Slashed[index]; before {
				continue
			}
		}

		unclaimed[index] = reward
	}

	claim := func(index phase0.ValidatorIndex) uint64 {
		reward, ok := unclaimed[index]
		if !ok {
			return 0
		}

		delete(unclaimed, index)

		return reward
	}

	out := make([]phase0BlockReward, 0, len(blocks))

	for _, block := range blocks {
		if !inEpoch(block.Slot) {
			return nil, fmt.Errorf("block at slot %d is outside epoch %d", block.Slot, epoch)
		}

		reward := phase0BlockReward{Slot: block.Slot, ProposerIndex: block.ProposerIndex}

		inclusion := phase0Inclusion{Slot: block.Slot, ProposerIndex: block.ProposerIndex}
		reward.Attestations = credits[inclusion]

		delete(credits, inclusion)

		for _, slashing := range block.Body.ProposerSlashings {
			reward.ProposerSlashings += claim(slashing.SignedHeader1.Message.ProposerIndex)
		}

		for _, slashing := range block.Body.AttesterSlashings {
			for _, index := range slashableIntersection(slashing.Attestation1.AttestingIndices, slashing.Attestation2.AttestingIndices) {
				reward.AttesterSlashings += claim(index)
			}
		}

		out = append(out, reward)
	}

	// Leftovers mean the blocks or states are not the canonical chain the rewards were computed from.
	for inclusion := range credits {
		return nil, fmt.Errorf("attestations included at slot %d by proposer %d earned a reward but no such block was given", inclusion.Slot, inclusion.ProposerIndex)
	}

	for index := range unclaimed {
		return nil, fmt.Errorf("validator %d was slashed in epoch %d but no given block slashed it", index, epoch)
	}

	return out, nil
}

// slashableIntersection returns the indices in both attestations in ascending order, the order process_attester_slashing slashes them in.
func slashableIntersection(a, b []uint64) []phase0.ValidatorIndex {
	inB := make(map[uint64]struct{}, len(b))
	for _, index := range b {
		inB[index] = struct{}{}
	}

	out := make([]phase0.ValidatorIndex, 0, len(a))

	for _, index := range a {
		if _, ok := inB[index]; ok {
			out = append(out, phase0.ValidatorIndex(index))
		}
	}

	slices.Sort(out)

	return out
}

//nolint:gosec // G115: per-epoch phase0 deltas are far below MaxInt64 gwei.
func netDelta(d *zcommon.Deltas, i int) int64 {
	return int64(d.Rewards[i]) - int64(d.Penalties[i])
}

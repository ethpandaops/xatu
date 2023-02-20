package event

import (
	"context"
	"fmt"
	"time"

	"github.com/attestantio/go-eth2-client/spec"
	"github.com/attestantio/go-eth2-client/spec/bellatrix"
	"github.com/attestantio/go-eth2-client/spec/capella"
	"github.com/attestantio/go-eth2-client/spec/phase0"
	xatuethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	xatuethv2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/sentry/ethereum"
	hashstructure "github.com/mitchellh/hashstructure/v2"
	ttlcache "github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type BeaconBlock struct {
	log logrus.FieldLogger

	now time.Time

	event          *spec.VersionedSignedBeaconBlock
	beacon         *ethereum.BeaconNode
	duplicateCache *ttlcache.Cache[string, time.Time]
	clientMeta     *xatu.ClientMeta
}

func NewBeaconBlock(log logrus.FieldLogger, event *spec.VersionedSignedBeaconBlock, now time.Time, beacon *ethereum.BeaconNode, duplicateCache *ttlcache.Cache[string, time.Time], clientMeta *xatu.ClientMeta) *BeaconBlock {
	return &BeaconBlock{
		log:            log.WithField("event", "BEACON_API_ETH_V2_BEACON_BLOCK"),
		now:            now,
		event:          event,
		beacon:         beacon,
		duplicateCache: duplicateCache,
		clientMeta:     clientMeta,
	}
}

func (e *BeaconBlock) Decorate(ctx context.Context) (*xatu.DecoratedEvent, error) {
	ignore, err := e.shouldIgnore(ctx)
	if err != nil {
		return nil, err
	}

	if ignore {
		return nil, nil
	}

	var data *xatuethv2.EventBlock

	switch e.event.Version {
	case spec.DataVersionPhase0:
		data = e.getPhase0Data()
	case spec.DataVersionAltair:
		data = e.getAltairData()
	case spec.DataVersionBellatrix:
		data = e.getBellatrixData()
	case spec.DataVersionCapella:
		data = e.getCapellaData()
	default:
		return nil, fmt.Errorf("unsupported block version: %v", e.event.Version)
	}

	decoratedEvent := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V2_BEACON_BLOCK,
			DateTime: timestamppb.New(e.now),
		},
		Meta: &xatu.Meta{
			Client: e.clientMeta,
		},
		Data: &xatu.DecoratedEvent_EthV2BeaconBlock{
			EthV2BeaconBlock: data,
		},
	}

	additionalData, err := e.getAdditionalData(ctx)
	if err != nil {
		e.log.WithError(err).Error("Failed to get extra beacon block data")
	} else {
		decoratedEvent.Meta.Client.AdditionalData = &xatu.ClientMeta_EthV2BeaconBlock{
			EthV2BeaconBlock: additionalData,
		}
	}

	return decoratedEvent, nil
}

func (e *BeaconBlock) shouldIgnore(ctx context.Context) (bool, error) {
	if e.event == nil {
		return true, nil
	}

	if err := e.beacon.Synced(ctx); err != nil {
		return true, nil
	}

	hash, err := hashstructure.Hash(e.event, hashstructure.FormatV2, nil)
	if err != nil {
		return true, err
	}

	item, retrieved := e.duplicateCache.GetOrSet(fmt.Sprint(hash), e.now, ttlcache.DefaultTTL)
	if retrieved {
		e.log.WithFields(logrus.Fields{
			"hash":                  hash,
			"time_since_first_item": time.Since(item.Value()),
			"slot":                  e.event.Slot,
		}).Debug("Duplicate beacon block event received")

		return true, nil
	}

	currentSlot, _, err := e.beacon.Node().Wallclock().Now()
	if err != nil {
		return true, err
	}

	// ignore blocks that are more than 16 slots old
	slotLimit := currentSlot.Number() - 16

	slot, err := e.event.Slot()
	if err != nil {
		return true, err
	}

	if uint64(slot) < slotLimit {
		return true, nil
	}

	return false, nil
}

func getProposerSlashings(data []*phase0.ProposerSlashing) []*xatuethv1.ProposerSlashing {
	slashings := []*xatuethv1.ProposerSlashing{}

	if data != nil {
		return slashings
	}

	for _, slashing := range data {
		slashings = append(slashings, &xatuethv1.ProposerSlashing{
			SignedHeader_1: &xatuethv1.SignedBeaconBlockHeader{
				Message: &xatuethv1.BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader1.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader1.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader1.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader1.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader1.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader1.Signature.String(),
			},
			SignedHeader_2: &xatuethv1.SignedBeaconBlockHeader{
				Message: &xatuethv1.BeaconBlockHeader{
					Slot:          uint64(slashing.SignedHeader2.Message.Slot),
					ProposerIndex: uint64(slashing.SignedHeader2.Message.ProposerIndex),
					ParentRoot:    slashing.SignedHeader2.Message.ParentRoot.String(),
					StateRoot:     slashing.SignedHeader2.Message.StateRoot.String(),
					BodyRoot:      slashing.SignedHeader2.Message.BodyRoot.String(),
				},
				Signature: slashing.SignedHeader2.Signature.String(),
			},
		})
	}

	return slashings
}

func getAttesterSlashings(data []*phase0.AttesterSlashing) []*xatuethv1.AttesterSlashing {
	slashings := []*xatuethv1.AttesterSlashing{}

	if data == nil {
		return slashings
	}

	for _, slashing := range data {
		slashings = append(slashings, &xatuethv1.AttesterSlashing{
			Attestation_1: &xatuethv1.IndexedAttestation{
				AttestingIndices: slashing.Attestation1.AttestingIndices,
				Data: &xatuethv1.AttestationData{
					Slot:            uint64(slashing.Attestation1.Data.Slot),
					Index:           uint64(slashing.Attestation1.Data.Index),
					BeaconBlockRoot: slashing.Attestation1.Data.BeaconBlockRoot.String(),
					Source: &xatuethv1.Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Source.Epoch),
						Root:  slashing.Attestation1.Data.Source.Root.String(),
					},
					Target: &xatuethv1.Checkpoint{
						Epoch: uint64(slashing.Attestation1.Data.Target.Epoch),
						Root:  slashing.Attestation1.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation1.Signature.String(),
			},
			Attestation_2: &xatuethv1.IndexedAttestation{
				AttestingIndices: slashing.Attestation2.AttestingIndices,
				Data: &xatuethv1.AttestationData{
					Slot:            uint64(slashing.Attestation2.Data.Slot),
					Index:           uint64(slashing.Attestation2.Data.Index),
					BeaconBlockRoot: slashing.Attestation2.Data.BeaconBlockRoot.String(),
					Source: &xatuethv1.Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Source.Epoch),
						Root:  slashing.Attestation2.Data.Source.Root.String(),
					},
					Target: &xatuethv1.Checkpoint{
						Epoch: uint64(slashing.Attestation2.Data.Target.Epoch),
						Root:  slashing.Attestation2.Data.Target.Root.String(),
					},
				},
				Signature: slashing.Attestation2.Signature.String(),
			},
		})
	}

	return slashings
}

func getAttestations(data []*phase0.Attestation) []*xatuethv1.Attestation {
	attestations := []*xatuethv1.Attestation{}

	if data == nil {
		return attestations
	}

	for _, attestation := range data {
		attestations = append(attestations, &xatuethv1.Attestation{
			AggregationBits: fmt.Sprintf("0x%x", attestation.AggregationBits),
			Data: &xatuethv1.AttestationData{
				Slot:            uint64(attestation.Data.Slot),
				Index:           uint64(attestation.Data.Index),
				BeaconBlockRoot: attestation.Data.BeaconBlockRoot.String(),
				Source: &xatuethv1.Checkpoint{
					Epoch: uint64(attestation.Data.Source.Epoch),
					Root:  attestation.Data.Source.Root.String(),
				},
				Target: &xatuethv1.Checkpoint{
					Epoch: uint64(attestation.Data.Target.Epoch),
					Root:  attestation.Data.Target.Root.String(),
				},
			},
			Signature: attestation.Signature.String(),
		})
	}

	return attestations
}

func getDeposits(data []*phase0.Deposit) []*xatuethv1.Deposit {
	deposits := []*xatuethv1.Deposit{}

	if data == nil {
		return deposits
	}

	for _, deposit := range data {
		proof := []string{}
		for _, p := range deposit.Proof {
			proof = append(proof, fmt.Sprintf("0x%x", p))
		}

		deposits = append(deposits, &xatuethv1.Deposit{
			Proof: proof,
			Data: &xatuethv1.Deposit_Data{
				Pubkey:                deposit.Data.PublicKey.String(),
				WithdrawalCredentials: fmt.Sprintf("0x%x", deposit.Data.WithdrawalCredentials),
				Amount:                uint64(deposit.Data.Amount),
				Signature:             deposit.Data.Signature.String(),
			},
		})
	}

	return deposits
}

func getVoluntaryExits(data []*phase0.SignedVoluntaryExit) []*xatuethv1.SignedVoluntaryExit {
	exits := []*xatuethv1.SignedVoluntaryExit{}

	if data == nil {
		return exits
	}

	for _, exit := range data {
		exits = append(exits, &xatuethv1.SignedVoluntaryExit{
			Message: &xatuethv1.VoluntaryExit{
				Epoch:          uint64(exit.Message.Epoch),
				ValidatorIndex: uint64(exit.Message.ValidatorIndex),
			},
			Signature: exit.Signature.String(),
		})
	}

	return exits
}

func getTransactions(data []bellatrix.Transaction) []string {
	transactions := []string{}

	for _, tx := range data {
		transactions = append(transactions, fmt.Sprintf("0x%x", tx))
	}

	return transactions
}

func getBlsToExecutionChanges(data []*capella.SignedBLSToExecutionChange) []*xatuethv2.SignedBLSToExecutionChange {
	changes := []*xatuethv2.SignedBLSToExecutionChange{}

	if data == nil {
		return changes
	}

	for _, change := range data {
		changes = append(changes, &xatuethv2.SignedBLSToExecutionChange{
			Message: &xatuethv2.BLSToExecutionChange{
				ValidatorIndex:     uint64(change.Message.ValidatorIndex),
				FromBlsPubkey:      change.Message.FromBLSPubkey.String(),
				ToExecutionAddress: change.Message.ToExecutionAddress.String(),
			},
			Signature: change.Signature.String(),
		})
	}

	return changes
}

func getWithdrawals(data []*capella.Withdrawal) []*xatuethv1.Withdrawal {
	withdrawals := []*xatuethv1.Withdrawal{}

	if data == nil {
		return withdrawals
	}

	for _, withdrawal := range data {
		withdrawals = append(withdrawals, &xatuethv1.Withdrawal{
			Index:          uint64(withdrawal.Index),
			ValidatorIndex: uint64(withdrawal.ValidatorIndex),
			Address:        withdrawal.Address.String(),
			Amount:         uint64(withdrawal.Amount),
		})
	}

	return withdrawals
}

func (e *BeaconBlock) getPhase0Data() *xatuethv2.EventBlock {
	return &xatuethv2.EventBlock{
		Message: &xatuethv2.EventBlock_Phase0Block{
			Phase0Block: &xatuethv1.BeaconBlock{
				Slot:          uint64(e.event.Phase0.Message.Slot),
				ProposerIndex: uint64(e.event.Phase0.Message.ProposerIndex),
				ParentRoot:    e.event.Phase0.Message.ParentRoot.String(),
				StateRoot:     e.event.Phase0.Message.StateRoot.String(),
				Body: &xatuethv1.BeaconBlockBody{
					RandaoReveal: e.event.Phase0.Message.Body.RANDAOReveal.String(),
					Eth1Data: &xatuethv1.Eth1Data{
						DepositRoot:  e.event.Phase0.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: e.event.Phase0.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", e.event.Phase0.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", e.event.Phase0.Message.Body.Graffiti[:]),
					ProposerSlashings: getProposerSlashings(e.event.Phase0.Message.Body.ProposerSlashings),
					AttesterSlashings: getAttesterSlashings(e.event.Phase0.Message.Body.AttesterSlashings),
					Attestations:      getAttestations(e.event.Phase0.Message.Body.Attestations),
					Deposits:          getDeposits(e.event.Phase0.Message.Body.Deposits),
					VoluntaryExits:    getVoluntaryExits(e.event.Phase0.Message.Body.VoluntaryExits),
				},
			},
		},
		Signature: e.event.Phase0.Signature.String(),
	}
}

func (e *BeaconBlock) getAltairData() *xatuethv2.EventBlock {
	return &xatuethv2.EventBlock{
		Message: &xatuethv2.EventBlock_AltairBlock{
			AltairBlock: &xatuethv2.BeaconBlockAltair{
				Slot:          uint64(e.event.Altair.Message.Slot),
				ProposerIndex: uint64(e.event.Altair.Message.ProposerIndex),
				ParentRoot:    e.event.Altair.Message.ParentRoot.String(),
				StateRoot:     e.event.Altair.Message.StateRoot.String(),
				Body: &xatuethv2.BeaconBlockBodyAltair{
					RandaoReveal: e.event.Altair.Message.Body.RANDAOReveal.String(),
					Eth1Data: &xatuethv1.Eth1Data{
						DepositRoot:  e.event.Altair.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: e.event.Altair.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", e.event.Altair.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", e.event.Altair.Message.Body.Graffiti[:]),
					ProposerSlashings: getProposerSlashings(e.event.Altair.Message.Body.ProposerSlashings),
					AttesterSlashings: getAttesterSlashings(e.event.Altair.Message.Body.AttesterSlashings),
					Attestations:      getAttestations(e.event.Altair.Message.Body.Attestations),
					Deposits:          getDeposits(e.event.Altair.Message.Body.Deposits),
					VoluntaryExits:    getVoluntaryExits(e.event.Altair.Message.Body.VoluntaryExits),
					SyncAggregate: &xatuethv1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", e.event.Altair.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: e.event.Altair.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
				},
			},
		},
		Signature: e.event.Altair.Signature.String(),
	}
}

func (e *BeaconBlock) getBellatrixData() *xatuethv2.EventBlock {
	return &xatuethv2.EventBlock{
		Message: &xatuethv2.EventBlock_BellatrixBlock{
			BellatrixBlock: &xatuethv2.BeaconBlockBellatrix{
				Slot:          uint64(e.event.Bellatrix.Message.Slot),
				ProposerIndex: uint64(e.event.Bellatrix.Message.ProposerIndex),
				ParentRoot:    e.event.Bellatrix.Message.ParentRoot.String(),
				StateRoot:     e.event.Bellatrix.Message.StateRoot.String(),
				Body: &xatuethv2.BeaconBlockBodyBellatrix{
					RandaoReveal: e.event.Bellatrix.Message.Body.RANDAOReveal.String(),
					Eth1Data: &xatuethv1.Eth1Data{
						DepositRoot:  e.event.Bellatrix.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: e.event.Bellatrix.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.Graffiti[:]),
					ProposerSlashings: getProposerSlashings(e.event.Bellatrix.Message.Body.ProposerSlashings),
					AttesterSlashings: getAttesterSlashings(e.event.Bellatrix.Message.Body.AttesterSlashings),
					Attestations:      getAttestations(e.event.Bellatrix.Message.Body.Attestations),
					Deposits:          getDeposits(e.event.Bellatrix.Message.Body.Deposits),
					VoluntaryExits:    getVoluntaryExits(e.event.Bellatrix.Message.Body.VoluntaryExits),
					SyncAggregate: &xatuethv1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: e.event.Bellatrix.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &xatuethv1.ExecutionPayload{
						ParentHash:    e.event.Bellatrix.Message.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  e.event.Bellatrix.Message.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   e.event.Bellatrix.Message.Body.ExecutionPayload.BlockNumber,
						GasLimit:      e.event.Bellatrix.Message.Body.ExecutionPayload.GasLimit,
						GasUsed:       e.event.Bellatrix.Message.Body.ExecutionPayload.GasUsed,
						Timestamp:     e.event.Bellatrix.Message.Body.ExecutionPayload.Timestamp,
						ExtraData:     fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: fmt.Sprintf("0x%x", e.event.Bellatrix.Message.Body.ExecutionPayload.BaseFeePerGas[:]),
						BlockHash:     e.event.Bellatrix.Message.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(e.event.Bellatrix.Message.Body.ExecutionPayload.Transactions),
					},
				},
			},
		},
		Signature: e.event.Bellatrix.Signature.String(),
	}
}

func (e *BeaconBlock) getCapellaData() *xatuethv2.EventBlock {
	return &xatuethv2.EventBlock{
		Message: &xatuethv2.EventBlock_CapellaBlock{
			CapellaBlock: &xatuethv2.BeaconBlockCapella{
				Slot:          uint64(e.event.Capella.Message.Slot),
				ProposerIndex: uint64(e.event.Capella.Message.ProposerIndex),
				ParentRoot:    e.event.Capella.Message.ParentRoot.String(),
				StateRoot:     e.event.Capella.Message.StateRoot.String(),
				Body: &xatuethv2.BeaconBlockBodyCapella{
					RandaoReveal: e.event.Capella.Message.Body.RANDAOReveal.String(),
					Eth1Data: &xatuethv1.Eth1Data{
						DepositRoot:  e.event.Capella.Message.Body.ETH1Data.DepositRoot.String(),
						DepositCount: e.event.Capella.Message.Body.ETH1Data.DepositCount,
						BlockHash:    fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ETH1Data.BlockHash),
					},
					Graffiti:          fmt.Sprintf("0x%x", e.event.Capella.Message.Body.Graffiti[:]),
					ProposerSlashings: getProposerSlashings(e.event.Capella.Message.Body.ProposerSlashings),
					AttesterSlashings: getAttesterSlashings(e.event.Capella.Message.Body.AttesterSlashings),
					Attestations:      getAttestations(e.event.Capella.Message.Body.Attestations),
					Deposits:          getDeposits(e.event.Capella.Message.Body.Deposits),
					VoluntaryExits:    getVoluntaryExits(e.event.Capella.Message.Body.VoluntaryExits),
					SyncAggregate: &xatuethv1.SyncAggregate{
						SyncCommitteeBits:      fmt.Sprintf("0x%x", e.event.Capella.Message.Body.SyncAggregate.SyncCommitteeBits),
						SyncCommitteeSignature: e.event.Capella.Message.Body.SyncAggregate.SyncCommitteeSignature.String(),
					},
					ExecutionPayload: &xatuethv1.ExecutionPayloadCapella{
						ParentHash:    e.event.Capella.Message.Body.ExecutionPayload.ParentHash.String(),
						FeeRecipient:  e.event.Capella.Message.Body.ExecutionPayload.FeeRecipient.String(),
						StateRoot:     fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.StateRoot[:]),
						ReceiptsRoot:  fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.ReceiptsRoot[:]),
						LogsBloom:     fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.LogsBloom[:]),
						PrevRandao:    fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.PrevRandao[:]),
						BlockNumber:   e.event.Capella.Message.Body.ExecutionPayload.BlockNumber,
						GasLimit:      e.event.Capella.Message.Body.ExecutionPayload.GasLimit,
						GasUsed:       e.event.Capella.Message.Body.ExecutionPayload.GasUsed,
						Timestamp:     e.event.Capella.Message.Body.ExecutionPayload.Timestamp,
						ExtraData:     fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.ExtraData),
						BaseFeePerGas: fmt.Sprintf("0x%x", e.event.Capella.Message.Body.ExecutionPayload.BaseFeePerGas[:]),
						BlockHash:     e.event.Capella.Message.Body.ExecutionPayload.BlockHash.String(),
						Transactions:  getTransactions(e.event.Capella.Message.Body.ExecutionPayload.Transactions),
						Withdrawals:   getWithdrawals(e.event.Capella.Message.Body.ExecutionPayload.Withdrawals),
					},
					BlsToExecutionChanges: getBlsToExecutionChanges(e.event.Capella.Message.Body.BLSToExecutionChanges),
				},
			},
		},
		Signature: e.event.Capella.Signature.String(),
	}
}

func (e *BeaconBlock) getAdditionalData(ctx context.Context) (*xatu.ClientMeta_AdditionalEthV2BeaconBlockData, error) {
	extra := &xatu.ClientMeta_AdditionalEthV2BeaconBlockData{}

	slotI, err := e.event.Slot()
	if err != nil {
		return nil, err
	}

	slot := e.beacon.Metadata().Wallclock().Slots().FromNumber(uint64(slotI))
	epoch := e.beacon.Metadata().Wallclock().Epochs().FromSlot(uint64(slotI))

	extra.Slot = &xatu.Slot{
		StartDateTime: timestamppb.New(slot.TimeWindow().Start()),
		Number:        uint64(slotI),
	}

	extra.Epoch = &xatu.Epoch{
		Number:        epoch.Number(),
		StartDateTime: timestamppb.New(epoch.TimeWindow().Start()),
	}

	extra.Version = e.event.Version.String()

	return extra, nil
}

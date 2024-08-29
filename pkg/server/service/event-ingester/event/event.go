package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	v1 "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/blockprint"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/libp2p"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/mempool"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/mevrelay"
	"github.com/ethpandaops/xatu/pkg/server/store"
)

type Type string

var (
	TypeUnknown                                 Type = "unknown"
	TypeBeaconETHV1EventsBlock                  Type = v1.EventsBlockType
	TypeBeaconETHV1EventsBlockV2                Type = v1.EventsBlockV2Type
	TypeBeaconETHV1EventsChainReorg             Type = v1.EventsChainReorgType
	TypeBeaconETHV1EventsChainReorgV2           Type = v1.EventsChainReorgV2Type
	TypeBeaconETHV1EventsFinalizedCheckpoint    Type = v1.EventsFinalizedCheckpointType
	TypeBeaconETHV1EventsFinalizedCheckpointV2  Type = v1.EventsFinalizedCheckpointV2Type
	TypeBeaconETHV1EventsHead                   Type = v1.EventsHeadType
	TypeBeaconETHV1EventsHeadV2                 Type = v1.EventsHeadV2Type
	TypeBeaconETHV1EventsVoluntaryExit          Type = v1.EventsVoluntaryExitType
	TypeBeaconETHV1EventsVoluntaryExitV2        Type = v1.EventsVoluntaryExitV2Type
	TypeBeaconETHV1EventsAttestation            Type = v1.EventsAttestationType
	TypeBeaconETHV1EventsAttestationV2          Type = v1.EventsAttestationV2Type
	TypeBeaconETHV1EventsContributionAndProof   Type = v1.EventsContributionAndProofType
	TypeBeaconETHV1EventsContributionAndProofV2 Type = v1.EventsContributionAndProofV2Type
	TypeMempoolTransaction                      Type = mempool.TransactionType
	TypeMempoolTransactionV2                    Type = mempool.TransactionV2Type
	TypeBeaconETHV2BeaconBlock                  Type = v2.BeaconBlockType
	TypeBeaconETHV2BeaconBlockV2                Type = v2.BeaconBlockV2Type
	TypeDebugForkChoice                         Type = v1.DebugForkChoiceType
	TypeDebugForkChoiceV2                       Type = v1.DebugForkChoiceV2Type
	TypeDebugForkChoiceReorg                    Type = v1.DebugForkChoiceReorgType
	TypeDebugForkChoiceReorgV2                  Type = v1.DebugForkChoiceReorgV2Type
	TypeBeaconEthV1BeaconCommittee              Type = v1.BeaconCommitteeType
	TypeBeaconEthV1ValidatorAttestationData     Type = v1.ValidatorAttestationDataType
	TypeBeaconEthV2BeaconBlockAttesterSlashing  Type = v2.BeaconBlockAttesterSlashingType
	TypeBeaconEthV2BeaconBlockProposerSlashing  Type = v2.BeaconBlockProposerSlashingType
	TypeBeaconEthV2BeaconBlockVoluntaryExit     Type = v2.BeaconBlockVoluntaryExitType
	TypeBeaconEthV2BeaconBlockDeposit           Type = v2.BeaconBlockDepositType
	TypeBeaconEthV2BeaconExecutionTransaction   Type = v2.BeaconBlockExecutionTransactionType
	TypeBeaconEthV2BeaconBLSToExecutionChange   Type = v2.BeaconBlockBLSToExecutionChangeType
	TypeBeaconEthV2BeaconWithdrawal             Type = v2.BeaconBlockWithdrawalType
	TypeBlockprintBlockClassification           Type = blockprint.BlockClassificationType
	TypeBeaconETHV1EventsBlobSidecar            Type = v1.EventsBlobSidecarType
	TypeBeaconETHV1BeaconBlobSidecar            Type = v1.BeaconBlobSidecarType
	TypeBeaconEthV1ProposerDuty                 Type = v1.BeaconProposerDutyType
	TypeBeaconP2PAttestation                    Type = v1.BeaconP2PAttestationType
	TypeBeaconEthV2BeaconElaboratedAttestation  Type = v2.BeaconBlockElaboratedAttestationType
	TypeLibP2PTraceAddPeer                      Type = Type(libp2p.TraceAddPeerType)
	TypeLibP2PTraceConnected                    Type = Type(libp2p.TraceConnectedType)
	TypeLibP2PTraceJoin                         Type = Type(libp2p.TraceJoinType)
	TypeLibP2PTraceDisconnected                 Type = Type(libp2p.TraceDisconnectedType)
	TypeLibP2PTraceRemovePeer                   Type = Type(libp2p.TraceRemovePeerType)
	TypeLibP2PTraceRecvRPC                      Type = Type(libp2p.TraceRecvRPCType)
	TypeLibP2PTraceSendRPC                      Type = Type(libp2p.TraceSendRPCType)
	TypeLibP2PTraceHandleStatus                 Type = Type(libp2p.TraceHandleStatusType)
	TypeLibP2PTraceHandleMetadata               Type = Type(libp2p.TraceHandleMetadataType)
	TypeLibP2PTraceGossipSubBeaconBlock         Type = Type(libp2p.TraceGossipSubBeaconBlockType)
	TypeLibP2PTraceGossipSubBeaconAttestation   Type = Type(libp2p.TraceGossipSubBeaconAttestationType)
	TypeLibP2PTraceGossipSubBlobSidecar         Type = Type(libp2p.TraceGossipSubBlobSidecarType)
	TypeBeaconETHV1BeaconValidators             Type = Type(v1.BeaconValidatorsType)
	TypeMEVRelayBidTraceBuilderBlockSubmission  Type = Type(mevrelay.BidTraceBuilderBlockSubmissionType)
)

type Event interface {
	Type() string
	Validate(ctx context.Context) error
	Filter(ctx context.Context) bool
	AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta
}

//nolint:gocyclo //not that complex
func New(eventType Type, log logrus.FieldLogger, event *xatu.DecoratedEvent, cache store.Cache, geoipProvider geoip.Provider) (Event, error) {
	if eventType == TypeUnknown {
		return nil, errors.New("event type is required")
	}

	switch eventType {
	case TypeBeaconETHV1EventsAttestationV2:
		return v1.NewEventsAttestationV2(log, event), nil
	case TypeLibP2PTraceGossipSubBeaconAttestation:
		return libp2p.NewTraceGossipSubBeaconAttestation(log, event), nil
	case TypeBeaconETHV1BeaconValidators:
		return v1.NewBeaconValidators(log, event), nil
	case TypeBeaconP2PAttestation:
		return v1.NewBeaconP2PAttestation(log, event, geoipProvider), nil
	case TypeBeaconETHV1EventsAttestation:
		return v1.NewEventsAttestation(log, event), nil
	case TypeBeaconETHV1EventsBlock:
		return v1.NewEventsBlock(log, event), nil
	case TypeBeaconETHV1EventsBlockV2:
		return v1.NewEventsBlockV2(log, event), nil
	case TypeBeaconETHV1EventsChainReorg:
		return v1.NewEventsChainReorg(log, event), nil
	case TypeBeaconETHV1EventsChainReorgV2:
		return v1.NewEventsChainReorgV2(log, event), nil
	case TypeBeaconETHV1EventsFinalizedCheckpoint:
		return v1.NewEventsFinalizedCheckpoint(log, event), nil
	case TypeBeaconETHV1EventsFinalizedCheckpointV2:
		return v1.NewEventsFinalizedCheckpointV2(log, event), nil
	case TypeBeaconETHV1EventsHead:
		return v1.NewEventsHead(log, event), nil
	case TypeBeaconETHV1EventsHeadV2:
		return v1.NewEventsHeadV2(log, event), nil
	case TypeBeaconETHV1EventsVoluntaryExit:
		return v1.NewEventsVoluntaryExit(log, event), nil
	case TypeBeaconETHV1EventsVoluntaryExitV2:
		return v1.NewEventsVoluntaryExitV2(log, event), nil
	case TypeBeaconETHV1EventsContributionAndProof:
		return v1.NewEventsContributionAndProof(log, event), nil
	case TypeBeaconETHV1EventsContributionAndProofV2:
		return v1.NewEventsContributionAndProofV2(log, event), nil
	case TypeMempoolTransaction:
		return mempool.NewTransaction(log, event), nil
	case TypeMempoolTransactionV2:
		return mempool.NewTransactionV2(log, event), nil
	case TypeBeaconETHV2BeaconBlock:
		return v2.NewBeaconBlock(log, event, cache), nil
	case TypeBeaconETHV2BeaconBlockV2:
		return v2.NewBeaconBlockV2(log, event, cache), nil
	case TypeDebugForkChoice:
		return v1.NewDebugForkChoice(log, event), nil
	case TypeDebugForkChoiceV2:
		return v1.NewDebugForkChoiceV2(log, event), nil
	case TypeDebugForkChoiceReorg:
		return v1.NewDebugForkChoiceReorg(log, event), nil
	case TypeDebugForkChoiceReorgV2:
		return v1.NewDebugForkChoiceReorgV2(log, event), nil
	case TypeBeaconEthV1BeaconCommittee:
		return v1.NewBeaconCommittee(log, event), nil
	case TypeBeaconEthV1ValidatorAttestationData:
		return v1.NewValidatorAttestationData(log, event), nil
	case TypeBeaconEthV2BeaconBlockAttesterSlashing:
		return v2.NewBeaconBlockAttesterSlashing(log, event), nil
	case TypeBeaconEthV2BeaconBlockProposerSlashing:
		return v2.NewBeaconBlockProposerSlashing(log, event), nil
	case TypeBeaconEthV2BeaconBlockVoluntaryExit:
		return v2.NewBeaconBlockVoluntaryExit(log, event), nil
	case TypeBeaconEthV2BeaconBlockDeposit:
		return v2.NewBeaconBlockDeposit(log, event), nil
	case TypeBeaconEthV2BeaconExecutionTransaction:
		return v2.NewBeaconBlockExecutionTransaction(log, event), nil
	case TypeBeaconEthV2BeaconBLSToExecutionChange:
		return v2.NewBeaconBlockBLSToExecutionChange(log, event), nil
	case TypeBeaconEthV2BeaconWithdrawal:
		return v2.NewBeaconBlockWithdrawal(log, event), nil
	case TypeBlockprintBlockClassification:
		return blockprint.NewBlockClassification(log, event), nil
	case TypeBeaconETHV1EventsBlobSidecar:
		return v1.NewEventsBlobSidecar(log, event), nil
	case TypeBeaconETHV1BeaconBlobSidecar:
		return v1.NewBeaconBlobSidecar(log, event), nil
	case TypeBeaconEthV1ProposerDuty:
		return v1.NewBeaconProposerDuty(log, event), nil
	case TypeBeaconEthV2BeaconElaboratedAttestation:
		return v2.NewBeaconBlockElaboratedAttestation(log, event), nil
	case TypeLibP2PTraceAddPeer:
		return libp2p.NewTraceAddPeer(log, event), nil
	case TypeLibP2PTraceConnected:
		return libp2p.NewTraceConnected(log, event, geoipProvider), nil
	case TypeLibP2PTraceJoin:
		return libp2p.NewTraceJoin(log, event), nil
	case TypeLibP2PTraceDisconnected:
		return libp2p.NewTraceDisconnected(log, event, geoipProvider), nil
	case TypeLibP2PTraceRemovePeer:
		return libp2p.NewTraceRemovePeer(log, event), nil
	case TypeLibP2PTraceRecvRPC:
		return libp2p.NewTraceRecvRPC(log, event), nil
	case TypeLibP2PTraceSendRPC:
		return libp2p.NewTraceSendRPC(log, event), nil
	case TypeLibP2PTraceHandleStatus:
		return libp2p.NewTraceHandleStatus(log, event), nil
	case TypeLibP2PTraceHandleMetadata:
		return libp2p.NewTraceHandleMetadata(log, event), nil
	case TypeLibP2PTraceGossipSubBeaconBlock:
		return libp2p.NewTraceGossipSubBeaconBlock(log, event), nil
	case TypeLibP2PTraceGossipSubBlobSidecar:
		return libp2p.NewTraceGossipSubBlobSidecar(log, event), nil
	case TypeMEVRelayBidTraceBuilderBlockSubmission:
		return mevrelay.NewBidTraceBuilderBlockSubmission(log, event), nil
	default:
		return nil, fmt.Errorf("event type %s is unknown", eventType)
	}
}

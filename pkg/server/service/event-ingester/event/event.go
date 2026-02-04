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
	v3 "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/beacon/eth/v3"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/consensus"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/execution"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/libp2p"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/mempool"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/mevrelay"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/noderecord"
	"github.com/ethpandaops/xatu/pkg/server/store"
)

type Type string

var (
	TypeUnknown                                 Type = "unknown"
	TypeBeaconETHV1EventsBlock                  Type = v1.EventsBlockType
	TypeBeaconETHV1EventsBlockV2                Type = v1.EventsBlockV2Type
	TypeBeaconETHV1EventsBlockGossip            Type = v1.EventsBlockGossipType
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
	TypeExecutionStateSize                      Type = execution.ExecutionStateSizeType
	TypeExecutionEngineNewPayload               Type = execution.EngineNewPayloadType
	TypeExecutionEngineGetBlobs                 Type = execution.EngineGetBlobsType
	TypeConsensusEngineAPINewPayload            Type = consensus.EngineAPINewPayloadType
	TypeConsensusEngineAPIGetBlobs              Type = consensus.EngineAPIGetBlobsType
	TypeBeaconETHV2BeaconBlock                  Type = v2.BeaconBlockType
	TypeBeaconETHV2BeaconBlockV2                Type = v2.BeaconBlockV2Type
	TypeDebugForkChoice                         Type = v1.DebugForkChoiceType
	TypeDebugForkChoiceV2                       Type = v1.DebugForkChoiceV2Type
	TypeDebugForkChoiceReorg                    Type = v1.DebugForkChoiceReorgType
	TypeDebugForkChoiceReorgV2                  Type = v1.DebugForkChoiceReorgV2Type
	TypeBeaconEthV1BeaconCommittee              Type = v1.BeaconCommitteeType
	TypeBeaconEthV1BeaconSyncCommittee          Type = v1.BeaconSyncCommitteeType
	TypeBeaconEthV1ValidatorAttestationData     Type = v1.ValidatorAttestationDataType
	TypeBeaconEthV2BeaconBlockAttesterSlashing  Type = v2.BeaconBlockAttesterSlashingType
	TypeBeaconEthV2BeaconBlockProposerSlashing  Type = v2.BeaconBlockProposerSlashingType
	TypeBeaconEthV2BeaconBlockVoluntaryExit     Type = v2.BeaconBlockVoluntaryExitType
	TypeBeaconEthV2BeaconBlockDeposit           Type = v2.BeaconBlockDepositType
	TypeBeaconEthV2BeaconExecutionTransaction   Type = v2.BeaconBlockExecutionTransactionType
	TypeBeaconEthV2BeaconBLSToExecutionChange   Type = v2.BeaconBlockBLSToExecutionChangeType
	TypeBeaconEthV2BeaconWithdrawal             Type = v2.BeaconBlockWithdrawalType
	TypeBeaconETHV1EventsBlobSidecar            Type = v1.EventsBlobSidecarType
	TypeBeaconETHV1EventsDataColumnSidecar      Type = v1.EventsDataColumnSidecarType
	TypeBeaconETHV1BeaconBlobSidecar            Type = v1.BeaconBlobSidecarType
	TypeBeaconETHV1BeaconBlob                   Type = v1.BeaconBlobType
	TypeBeaconEthV1ProposerDuty                 Type = v1.BeaconProposerDutyType
	TypeBeaconP2PAttestation                    Type = v1.BeaconP2PAttestationType
	TypeBeaconEthV2BeaconElaboratedAttestation  Type = v2.BeaconBlockElaboratedAttestationType
	TypeLibP2PTraceDuplicateMessage             Type = Type(libp2p.TraceDuplicateMessageType)
	TypeLibP2PTraceDeliverMessage               Type = Type(libp2p.TraceDeliverMessageType)
	TypeLibP2PTraceRejectMessage                Type = Type(libp2p.TraceRejectMessageType)
	TypeLibP2PTracePublishMessage               Type = Type(libp2p.TracePublishMessageType)
	TypeLibP2PTraceAddPeer                      Type = Type(libp2p.TraceAddPeerType)
	TypeLibP2PTraceConnected                    Type = Type(libp2p.TraceConnectedType)
	TypeLibP2PTraceJoin                         Type = Type(libp2p.TraceJoinType)
	TypeLibP2PTraceLeave                        Type = Type(libp2p.TraceLeaveType)
	TypeLibP2PTraceGraft                        Type = Type(libp2p.TraceGraftType)
	TypeLibP2PTracePrune                        Type = Type(libp2p.TracePruneType)
	TypeLibP2PTraceDisconnected                 Type = Type(libp2p.TraceDisconnectedType)
	TypeLibP2PTraceRemovePeer                   Type = Type(libp2p.TraceRemovePeerType)
	TypeLibP2PTraceRecvRPC                      Type = Type(libp2p.TraceRecvRPCType)
	TypeLibP2PTraceSendRPC                      Type = Type(libp2p.TraceSendRPCType)
	TypeLibP2PTraceDropRPC                      Type = Type(libp2p.TraceDropRPCType)
	TypeLibP2PTraceHandleStatus                 Type = Type(libp2p.TraceHandleStatusType)
	TypeLibP2PTraceHandleMetadata               Type = Type(libp2p.TraceHandleMetadataType)
	TypeLibP2PTraceGossipSubBeaconBlock         Type = Type(libp2p.TraceGossipSubBeaconBlockType)
	TypeLibP2PTraceGossipSubBeaconAttestation   Type = Type(libp2p.TraceGossipSubBeaconAttestationType)
	TypeLibP2PTraceGossipSubAggregateAndProof   Type = Type(libp2p.TraceGossipSubAggregateAndProofType)
	TypeLibP2PTraceGossipSubBlobSidecar         Type = Type(libp2p.TraceGossipSubBlobSidecarType)
	TypeLibP2PTraceGossipSubDataColumnSidecar   Type = Type(libp2p.TraceGossipSubDataColumnSidecarType)
	TypeBeaconETHV1BeaconValidators             Type = Type(v1.BeaconValidatorsType)
	TypeMEVRelayBidTraceBuilderBlockSubmission  Type = Type(mevrelay.BidTraceBuilderBlockSubmissionType)
	TypeMEVRelayProposerPayloadDelivered        Type = Type(mevrelay.ProposerPayloadDeliveredType)
	TypeMEVRelayValidatorRegistration           Type = Type(mevrelay.ValidatorRegistrationType)
	TypeBeaconETHV3ValidatorBlock               Type = v3.ValidatorBlockType
	TypeLibP2PTraceRPCMetaControlIHave          Type = Type(libp2p.TraceRPCMetaControlIHaveType)
	TypeLibP2PTraceRPCMetaControlIWant          Type = Type(libp2p.TraceRPCMetaControlIWantType)
	TypeLibP2PTraceRPCMetaControlIDontWant      Type = Type(libp2p.TraceRPCMetaControlIDontWantType)
	TypeLibP2PTraceRPCMetaControlGraft          Type = Type(libp2p.TraceRPCMetaControlGraftType)
	TypeLibP2PTraceRPCMetaControlPrune          Type = Type(libp2p.TraceRPCMetaControlPruneType)
	TypeLibP2PTraceRPCMetaSubscription          Type = Type(libp2p.TraceRPCMetaSubscriptionType)
	TypeLibP2PTraceRPCMetaMessage               Type = Type(libp2p.TraceRPCMetaMessageType)
	TypeNodeRecordExecution                     Type = Type(noderecord.ExecutionType)
	TypeNodeRecordConsensus                     Type = Type(noderecord.ConsensusType)
	TypeLibP2PTraceSyntheticHeartbeat           Type = Type(libp2p.TraceSyntheticHeartbeatType)
	TypeLibP2PRPCDataColumnCustodyProbe         Type = Type(libp2p.TypeLibp2pRPCDataColumnCustodyProbe)
)

type Event interface {
	Type() string
	Validate(ctx context.Context) error
	Filter(ctx context.Context) bool
	AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta
}

type EventRouter struct {
	log           logrus.FieldLogger
	cache         store.Cache
	geoipProvider geoip.Provider
	routes        map[Type]func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error)
}

func NewEventRouter(log logrus.FieldLogger, cache store.Cache, geoipProvider geoip.Provider) *EventRouter {
	router := &EventRouter{
		log:           log,
		cache:         cache,
		geoipProvider: geoipProvider,
		routes:        make(map[Type]func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error)),
	}

	router.RegisterHandler(TypeBeaconETHV1EventsAttestationV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsAttestationV2(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGossipSubBeaconAttestation, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGossipSubBeaconAttestation(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGossipSubAggregateAndProof, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGossipSubAggregateAndProof(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1BeaconValidators, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconValidators(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconP2PAttestation, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconP2PAttestation(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsAttestation, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsAttestation(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsBlock, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsBlock(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsBlockV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsBlockV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsBlockGossip, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsBlockGossip(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsChainReorg, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsChainReorg(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsChainReorgV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsChainReorgV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsFinalizedCheckpoint, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsFinalizedCheckpoint(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsFinalizedCheckpointV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsFinalizedCheckpointV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsHead, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsHead(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsHeadV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsHeadV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsVoluntaryExit, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsVoluntaryExit(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsVoluntaryExitV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsVoluntaryExitV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsContributionAndProof, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsContributionAndProof(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsContributionAndProofV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsContributionAndProofV2(router.log, event), nil
	})
	router.RegisterHandler(TypeMempoolTransaction, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return mempool.NewTransaction(router.log, event), nil
	})
	router.RegisterHandler(TypeMempoolTransactionV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return mempool.NewTransactionV2(router.log, event), nil
	})
	router.RegisterHandler(TypeExecutionStateSize, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return execution.NewExecutionStateSize(router.log, event), nil
	})
	router.RegisterHandler(TypeExecutionEngineNewPayload, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return execution.NewEngineNewPayload(router.log, event), nil
	})
	router.RegisterHandler(TypeExecutionEngineGetBlobs, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return execution.NewEngineGetBlobs(router.log, event), nil
	})
	router.RegisterHandler(TypeConsensusEngineAPINewPayload, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return consensus.NewEngineAPINewPayload(router.log, event), nil
	})
	router.RegisterHandler(TypeConsensusEngineAPIGetBlobs, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return consensus.NewEngineAPIGetBlobs(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV2BeaconBlock, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlock(router.log, event, router.cache), nil
	})
	router.RegisterHandler(TypeBeaconETHV2BeaconBlockV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockV2(router.log, event, router.cache), nil
	})
	router.RegisterHandler(TypeDebugForkChoice, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewDebugForkChoice(router.log, event), nil
	})
	router.RegisterHandler(TypeDebugForkChoiceV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewDebugForkChoiceV2(router.log, event), nil
	})
	router.RegisterHandler(TypeDebugForkChoiceReorg, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewDebugForkChoiceReorg(router.log, event), nil
	})
	router.RegisterHandler(TypeDebugForkChoiceReorgV2, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewDebugForkChoiceReorgV2(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV1BeaconCommittee, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconCommittee(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV1BeaconSyncCommittee, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconSyncCommittee(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV1ValidatorAttestationData, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewValidatorAttestationData(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconBlockAttesterSlashing, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockAttesterSlashing(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconBlockProposerSlashing, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockProposerSlashing(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconBlockVoluntaryExit, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockVoluntaryExit(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconBlockDeposit, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockDeposit(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconExecutionTransaction, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockExecutionTransaction(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconBLSToExecutionChange, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockBLSToExecutionChange(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconWithdrawal, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockWithdrawal(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsBlobSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsBlobSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsDataColumnSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsDataColumnSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1BeaconBlobSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconBlobSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV1BeaconBlob, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconBlob(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV1ProposerDuty, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewBeaconProposerDuty(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconEthV2BeaconElaboratedAttestation, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v2.NewBeaconBlockElaboratedAttestation(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceDuplicateMessage, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceDuplicateMessage(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceDeliverMessage, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceDeliverMessage(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRejectMessage, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRejectMessage(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTracePublishMessage, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTracePublishMessage(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceAddPeer, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceAddPeer(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceConnected, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceConnected(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeLibP2PTraceJoin, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceJoin(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceLeave, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceLeave(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGraft, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGraft(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTracePrune, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTracePrune(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceDisconnected, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceDisconnected(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRemovePeer, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRemovePeer(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRecvRPC, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRecvRPC(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceSendRPC, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceSendRPC(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceDropRPC, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceDropRPC(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaControlIHave, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaControlIHave(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaControlIWant, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaControlIWant(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaControlIDontWant, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaControlIDontWant(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaControlGraft, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaControlGraft(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaControlPrune, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaControlPrune(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaSubscription, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaSubscription(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceRPCMetaMessage, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceRPCMetaMessage(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceHandleStatus, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceHandleStatus(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceHandleMetadata, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceHandleMetadata(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGossipSubBeaconBlock, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGossipSubBeaconBlock(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGossipSubBlobSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGossipSubBlobSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceGossipSubDataColumnSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceGossipSubDataColumnSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PTraceSyntheticHeartbeat, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewTraceSyntheticHeartbeat(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeMEVRelayBidTraceBuilderBlockSubmission, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return mevrelay.NewBidTraceBuilderBlockSubmission(router.log, event), nil
	})
	router.RegisterHandler(TypeMEVRelayProposerPayloadDelivered, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return mevrelay.NewProposerPayloadDelivered(router.log, event), nil
	})
	router.RegisterHandler(TypeBeaconETHV3ValidatorBlock, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v3.NewValidatorBlock(router.log, event), nil
	})
	router.RegisterHandler(TypeMEVRelayValidatorRegistration, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return mevrelay.NewValidatorRegistration(router.log, event), nil
	})
	router.RegisterHandler(TypeNodeRecordExecution, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return noderecord.NewExecution(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeNodeRecordConsensus, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return noderecord.NewConsensus(router.log, event, router.geoipProvider), nil
	})
	router.RegisterHandler(TypeBeaconETHV1EventsDataColumnSidecar, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return v1.NewEventsDataColumnSidecar(router.log, event), nil
	})
	router.RegisterHandler(TypeLibP2PRPCDataColumnCustodyProbe, func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error) {
		return libp2p.NewDataColumnCustodyProbe(router.log, event), nil
	})

	return router
}

func (er *EventRouter) RegisterHandler(eventType Type, handler func(event *xatu.DecoratedEvent, router *EventRouter) (Event, error)) {
	er.routes[eventType] = handler
}

func (er *EventRouter) HasRoute(eventType Type) bool {
	_, exists := er.routes[eventType]

	return exists
}

func (er *EventRouter) Route(eventType Type, event *xatu.DecoratedEvent) (Event, error) {
	if eventType == TypeUnknown {
		return nil, errors.New("event type is required")
	}

	handler, exists := er.routes[eventType]
	if !exists {
		return nil, fmt.Errorf("event type %s is unknown", eventType)
	}

	return handler(event, er)
}

package event

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	v1 "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/beacon/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/beacon/eth/v2"
	"github.com/ethpandaops/xatu/pkg/server/service/event-ingester/event/mempool"
	"github.com/ethpandaops/xatu/pkg/server/store"
	"github.com/sirupsen/logrus"
)

type Type string

const (
	TypeUnknown                               Type = "unknown"
	TypeBeaconETHV1EventsBlock                Type = v1.EventsBlockType
	TypeBeaconETHV1EventsChainReorg           Type = v1.EventsChainReorgType
	TypeBeaconETHV1EventsFinalizedCheckpoint  Type = v1.EventsFinalizedCheckpointType
	TypeBeaconETHV1EventsHead                 Type = v1.EventsHeadType
	TypeBeaconETHV1EventsVoluntaryExit        Type = v1.EventsVoluntaryExitType
	TypeBeaconETHV1EventsAttestation          Type = v1.EventsAttestationType
	TypeBeaconETHV1EventsContributionAndProof Type = v1.EventsContributionAndProofType
	TypeMempoolTransaction                    Type = mempool.TransactionType
	TypeBeaconETHV2BeaconBlock                Type = v2.BeaconBlockType
	TypeDebugForkChoice                       Type = v1.DebugForkChoiceType
	TypeDebugForkChoiceReorg                  Type = v1.DebugForkChoiceReorgType
)

type Event interface {
	Type() string
	Validate(ctx context.Context) error
	Filter(ctx context.Context) bool
}

func New(eventType Type, log logrus.FieldLogger, event *xatu.DecoratedEvent, cache store.Cache, networkID uint64) (Event, error) {
	if eventType == TypeUnknown {
		return nil, errors.New("event type is required")
	}

	switch eventType {
	case TypeBeaconETHV1EventsBlock:
		return v1.NewEventsBlock(log, event, networkID), nil
	case TypeBeaconETHV1EventsChainReorg:
		return v1.NewEventsChainReorg(log, event, networkID), nil
	case TypeBeaconETHV1EventsFinalizedCheckpoint:
		return v1.NewEventsFinalizedCheckpoint(log, event, networkID), nil
	case TypeBeaconETHV1EventsHead:
		return v1.NewEventsHead(log, event, networkID), nil
	case TypeBeaconETHV1EventsVoluntaryExit:
		return v1.NewEventsVoluntaryExit(log, event, networkID), nil
	case TypeBeaconETHV1EventsAttestation:
		return v1.NewEventsAttestation(log, event, networkID), nil
	case TypeBeaconETHV1EventsContributionAndProof:
		return v1.NewEventsContributionAndProof(log, event, networkID), nil
	case TypeMempoolTransaction:
		return mempool.NewTransaction(log, event, networkID), nil
	case TypeBeaconETHV2BeaconBlock:
		return v2.NewBeaconBlock(log, event, networkID, cache), nil
	case TypeDebugForkChoice:
		return v1.NewDebugForkChoice(log, event, networkID), nil
	case TypeDebugForkChoiceReorg:
		return v1.NewDebugForkChoiceReorg(log, event, networkID), nil
	default:
		return nil, fmt.Errorf("event type %s is unknown", eventType)
	}
}

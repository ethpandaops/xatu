package v1

import (
	eth2v1 "github.com/attestantio/go-eth2-client/api/v1"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func NewReorgEventFromGoEth2ClientEvent(e *eth2v1.ChainReorgEvent) *EventChainReorg {
	return &EventChainReorg{
		Slot:         uint64(e.Slot),
		Epoch:        uint64(e.Epoch),
		OldHeadBlock: RootAsString(e.OldHeadBlock),
		OldHeadState: RootAsString(e.OldHeadState),
		NewHeadBlock: RootAsString(e.NewHeadBlock),
		NewHeadState: RootAsString(e.NewHeadState),
		Depth:        e.Depth,
	}
}

func NewReorgEventV2FromGoEth2ClientEvent(e *eth2v1.ChainReorgEvent) *EventChainReorgV2 {
	return &EventChainReorgV2{
		Slot:         &wrapperspb.UInt64Value{Value: uint64(e.Slot)},
		Epoch:        &wrapperspb.UInt64Value{Value: uint64(e.Epoch)},
		OldHeadBlock: RootAsString(e.OldHeadBlock),
		OldHeadState: RootAsString(e.OldHeadState),
		NewHeadBlock: RootAsString(e.NewHeadBlock),
		NewHeadState: RootAsString(e.NewHeadState),
		Depth:        &wrapperspb.UInt64Value{Value: e.Depth},
	}
}

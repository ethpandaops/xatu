package ethereum

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/pkg/errors"
	zcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	zphase0 "github.com/protolambda/zrnt/eth2/beacon/phase0"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/codec"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/ethpandaops/xatu/pkg/observability"
)

// WithPhase0BeaconState downloads the phase0 beacon state stateID and calls fn with it and the matching consensus spec.
// The state is only referenced while fn runs; callers bound how many run at once, each holding a full state.
func (b *BeaconNode) WithPhase0BeaconState(
	ctx context.Context,
	stateID string,
	fn func(spec *zcommon.Spec, state *zphase0.BeaconStateView) error,
) error {
	ctx, span := observability.Tracer().Start(ctx, "ethereum.beacon.WithPhase0BeaconState",
		trace.WithAttributes(attribute.String("state_id", stateID)))
	defer span.End()

	zspec, err := b.phase0Spec()
	if err != nil {
		return err
	}

	start := time.Now()

	state, err := b.fetchPhase0State(ctx, zspec, stateID)
	if err != nil {
		return err
	}

	b.metrics.ObserveBeaconStateFetch(string(b.Metadata().Network.Name), time.Since(start))

	return fn(zspec, state)
}

func (b *BeaconNode) fetchPhase0State(ctx context.Context, zspec *zcommon.Spec, stateID string) (*zphase0.BeaconStateView, error) {
	rsp, err := b.beacon.OpenRawBeaconState(ctx, stateID, sszContentType)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to open beacon state %s", stateID)
	}

	defer rsp.Close()

	if version := rsp.Header.Get("Eth-Consensus-Version"); version != spec.DataVersionPhase0.String() {
		return nil, fmt.Errorf("beacon state %s has consensus version %q, expected %s", stateID, version, spec.DataVersionPhase0)
	}

	raw, err := io.ReadAll(io.LimitReader(rsp.Body, maxBeaconStateSSZSize+1))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read beacon state %s", stateID)
	}

	if len(raw) > maxBeaconStateSSZSize {
		return nil, fmt.Errorf("beacon state %s is larger than %d bytes", stateID, maxBeaconStateSSZSize)
	}

	state, err := zphase0.AsBeaconStateView(zphase0.BeaconStateType(zspec).Deserialize(
		codec.NewDecodingReader(bytes.NewReader(raw), uint64(len(raw)))))
	if err != nil {
		return nil, errors.Wrapf(err, "failed to decode beacon state %s", stateID)
	}

	return state, nil
}

// phase0Spec returns the zrnt spec for the node's preset. Phase0 state decoding, committees and rewards only depend on
// preset values, which zrnt ships for mainnet and minimal.
func (b *BeaconNode) phase0Spec() (*zcommon.Spec, error) {
	nodeSpec, err := b.beacon.Spec()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get spec")
	}

	switch nodeSpec.PresetBase {
	case "mainnet":
		return configs.Mainnet, nil
	case "minimal":
		return configs.Minimal, nil
	default:
		return nil, fmt.Errorf("phase0 state decoding does not support preset %q", nodeSpec.PresetBase)
	}
}

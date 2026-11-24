package v3

import (
	"context"
	"testing"

	"github.com/ethpandaops/go-eth2-client/spec"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

func validatorBlockEvent(version string, block *v2.EventBlockV2) *xatu.DecoratedEvent {
	return &xatu.DecoratedEvent{
		Meta: &xatu.Meta{Client: &xatu.ClientMeta{
			AdditionalData: &xatu.ClientMeta_EthV3ValidatorBlock{
				EthV3ValidatorBlock: &xatu.ClientMeta_AdditionalEthV3ValidatorBlockData{Version: version},
			},
		}},
		Data: &xatu.DecoratedEvent_EthV3ValidatorBlock{EthV3ValidatorBlock: block},
	}
}

func TestValidatorBlockFilter(t *testing.T) {
	tests := []struct {
		name    string
		version string
		block   *v2.EventBlockV2
		filter  bool
	}{
		{
			name:    "fulu",
			version: spec.DataVersionFulu.String(),
			block:   &v2.EventBlockV2{Message: &v2.EventBlockV2_FuluBlock{FuluBlock: &v2.BeaconBlockFulu{StateRoot: "0x01"}}},
		},
		{
			name:    "gloas",
			version: spec.DataVersionGloas.String(),
			block:   &v2.EventBlockV2{Message: &v2.EventBlockV2_GloasBlock{GloasBlock: &v2.BeaconBlockGloas{StateRoot: "0x01"}}},
		},
		{
			name:    "gloas version with a fulu block",
			version: spec.DataVersionGloas.String(),
			block:   &v2.EventBlockV2{Message: &v2.EventBlockV2_FuluBlock{FuluBlock: &v2.BeaconBlockFulu{StateRoot: "0x01"}}},
			filter:  true,
		},
		{
			name:    "unknown version",
			version: "heze",
			block:   &v2.EventBlockV2{Message: &v2.EventBlockV2_GloasBlock{GloasBlock: &v2.BeaconBlockGloas{StateRoot: "0x01"}}},
			filter:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewValidatorBlock(logrus.New(), validatorBlockEvent(tt.version, tt.block))
			assert.Equal(t, tt.filter, b.Filter(context.Background()))
		})
	}
}

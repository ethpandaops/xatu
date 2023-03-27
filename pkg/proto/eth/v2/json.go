package v2

import (
	"encoding/json"

	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

func (b *EventBlock) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Message   json.RawMessage `json:"message"`
		Version   BlockVersion    `json:"version,omitempty"`
		Signature string          `json:"signature,omitempty"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	switch tmp.Version {
	case BlockVersion_PHASE0:
		pb := new(v1.BeaconBlock)

		if err := protojson.Unmarshal(tmp.Message, pb); err != nil {
			return errors.Wrap(err, "failed to unmarshal EventBlock.Phase0Block")
		}

		b.Message = &EventBlock_Phase0Block{Phase0Block: pb}

	case BlockVersion_ALTAIR:
		pb := new(BeaconBlockAltair)

		if err := protojson.Unmarshal(tmp.Message, pb); err != nil {
			return errors.Wrap(err, "failed to unmarshal EventBlock.AltairBlock")
		}

		b.Message = &EventBlock_AltairBlock{AltairBlock: pb}

	case BlockVersion_BELLATRIX:
		pb := new(BeaconBlockBellatrix)

		if err := protojson.Unmarshal(tmp.Message, pb); err != nil {
			return errors.Wrap(err, "failed to unmarshal EventBlock.BellatrixBlock")
		}

		b.Message = &EventBlock_BellatrixBlock{BellatrixBlock: pb}

	case BlockVersion_CAPELLA:
		pb := new(BeaconBlockCapella)

		if err := protojson.Unmarshal(tmp.Message, pb); err != nil {
			return errors.Wrap(err, "failed to unmarshal EventBlock.CapellaBlock")
		}

		b.Message = &EventBlock_CapellaBlock{CapellaBlock: pb}

	default:
		return errors.Errorf("unknown block version: %d", tmp.Version)
	}

	b.Version = tmp.Version
	b.Signature = tmp.Signature

	return nil
}

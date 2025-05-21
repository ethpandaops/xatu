package cannon

import (
	"errors"
	"fmt"
	"time"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/encoding/protojson"
)

type Location struct {
	// LocationID is the location id.
	LocationID interface{} `json:"locationId" db:"location_id"`
	// CreateTime is the timestamp of when the execution record was created.
	CreateTime time.Time `json:"createTime" db:"create_time" fieldopt:"omitempty"`
	// UpdateTime is the timestamp of when the activity record was updated.
	UpdateTime time.Time `json:"updateTime" db:"update_time" fieldopt:"omitempty"`
	// NetworkId is the network id of the location.
	NetworkID string `json:"networkId" db:"network_id"`
	// Type is the type of the location.
	Type string `json:"type" db:"type"`
	// Value is the value of the location.
	Value string `json:"value" db:"value"`
}

var (
	ErrFailedToMarshal   = errors.New("failed to marshal location")
	ErrFailedToUnmarshal = errors.New("failed to unmarshal location")
)

// MarshalValueFromProto marshals a proto message into the Value field.
func (l *Location) Marshal(msg *xatu.CannonLocation) error {
	l.NetworkID = msg.NetworkId

	switch msg.Type {
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT"

		data := msg.GetEthV2BeaconBlockVoluntaryExit()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING"

		data := msg.GetEthV2BeaconBlockProposerSlashing()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT"

		data := msg.GetEthV2BeaconBlockDeposit()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING"

		data := msg.GetEthV2BeaconBlockAttesterSlashing()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)

	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION"

		data := msg.GetEthV2BeaconBlockExecutionTransaction()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)

	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE"

		data := msg.GetEthV2BeaconBlockBlsToExecutionChange()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL"

		data := msg.GetEthV2BeaconBlockWithdrawal()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)

	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK"

		data := msg.GetEthV2BeaconBlock()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)

	case xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION:
		l.Type = "BLOCKPRINT_BLOCK_CLASSIFICATION"

		data := msg.GetBlockprintBlockClassification()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)

	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR:
		l.Type = "BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR"

		data := msg.GetEthV1BeaconBlobSidecar()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY:
		l.Type = "BEACON_API_ETH_V1_PROPOSER_DUTY"

		data := msg.GetEthV1BeaconProposerDuty()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION"

		data := msg.GetEthV2BeaconBlockElaboratedAttestation()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS:
		l.Type = "BEACON_API_ETH_V1_BEACON_VALIDATORS"

		data := msg.GetEthV1BeaconValidators()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE:
		l.Type = "BEACON_API_ETH_V1_BEACON_COMMITTEE"

		data := msg.GetEthV1BeaconCommittee()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST"

		data := msg.GetEthV2BeaconBlockConsolidationRequest()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	default:
		return fmt.Errorf("unknown type: %s", msg.Type)
	}

	return nil
}

func (l *Location) Unmarshal() (*xatu.CannonLocation, error) {
	msg := &xatu.CannonLocation{
		NetworkId: l.NetworkID,
	}

	switch l.Type {
	case "BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_VOLUNTARY_EXIT

		data := &xatu.CannonLocationEthV2BeaconBlockVoluntaryExit{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockVoluntaryExit{
			EthV2BeaconBlockVoluntaryExit: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PROPOSER_SLASHING

		data := &xatu.CannonLocationEthV2BeaconBlockProposerSlashing{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockProposerSlashing{
			EthV2BeaconBlockProposerSlashing: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_DEPOSIT

		data := &xatu.CannonLocationEthV2BeaconBlockDeposit{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockDeposit{
			EthV2BeaconBlockDeposit: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ATTESTER_SLASHING

		data := &xatu.CannonLocationEthV2BeaconBlockAttesterSlashing{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockAttesterSlashing{
			EthV2BeaconBlockAttesterSlashing: data,
		}

	case "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_TRANSACTION

		data := &xatu.CannonLocationEthV2BeaconBlockExecutionTransaction{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionTransaction{
			EthV2BeaconBlockExecutionTransaction: data,
		}

	case "BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_BLS_TO_EXECUTION_CHANGE

		data := &xatu.CannonLocationEthV2BeaconBlockBlsToExecutionChange{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockBlsToExecutionChange{
			EthV2BeaconBlockBlsToExecutionChange: data,
		}

	case "BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_WITHDRAWAL

		data := &xatu.CannonLocationEthV2BeaconBlockWithdrawal{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockWithdrawal{
			EthV2BeaconBlockWithdrawal: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK

		data := &xatu.CannonLocationEthV2BeaconBlock{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlock{
			EthV2BeaconBlock: data,
		}
	case "BLOCKPRINT_BLOCK_CLASSIFICATION":
		msg.Type = xatu.CannonType_BLOCKPRINT_BLOCK_CLASSIFICATION

		data := &xatu.CannonLocationBlockprintBlockClassification{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_BlockprintBlockClassification{
			BlockprintBlockClassification: data,
		}
	case "BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOB_SIDECAR

		data := &xatu.CannonLocationEthV1BeaconBlobSidecar{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconBlobSidecar{
			EthV1BeaconBlobSidecar: data,
		}
	case "BEACON_API_ETH_V1_PROPOSER_DUTY":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_PROPOSER_DUTY

		data := &xatu.CannonLocationEthV1BeaconProposerDuty{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconProposerDuty{
			EthV1BeaconProposerDuty: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ELABORATED_ATTESTATION

		data := &xatu.CannonLocationEthV2BeaconBlockElaboratedAttestation{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockElaboratedAttestation{
			EthV2BeaconBlockElaboratedAttestation: data,
		}
	case "BEACON_API_ETH_V1_BEACON_VALIDATORS":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_VALIDATORS

		data := &xatu.CannonLocationEthV1BeaconValidators{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconValidators{
			EthV1BeaconValidators: data,
		}
	case "BEACON_API_ETH_V1_BEACON_COMMITTEE":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_COMMITTEE

		data := &xatu.CannonLocationEthV1BeaconCommittee{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconCommittee{
			EthV1BeaconCommittee: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_CONSOLIDATION_REQUEST

		data := &xatu.CannonLocationEthV2BeaconBlockConsolidationRequest{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockConsolidationRequest{
			EthV2BeaconBlockConsolidationRequest: data,
		}
	default:
		return nil, fmt.Errorf("unknown type: %s", l.Type)
	}

	return msg, nil
}

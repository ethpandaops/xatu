package cannon

import (
	"errors"
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

type Location struct {
	// LocationID is the location id.
	LocationID any `json:"locationId" db:"location_id"`
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

// marshalLocationData protojson-marshals data into l.Value and sets l.Type.
func (l *Location) marshalLocationData(typ string, data proto.Message) error {
	l.Type = typ

	b, err := protojson.Marshal(data)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
	}

	l.Value = string(b)

	return nil
}

// unmarshalLocationData protojson-unmarshals l.Value into data.
func unmarshalLocationData(value string, data proto.Message) error {
	if err := protojson.Unmarshal([]byte(value), data); err != nil {
		return fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
	}

	return nil
}

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
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE:
		l.Type = "BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE"

		data := msg.GetEthV1BeaconSyncCommittee()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE:
		l.Type = "BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE"

		data := msg.GetEthV2BeaconBlockSyncAggregate()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT", msg.GetEthV2BeaconBlockExecutionRequestDeposit())
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL", msg.GetEthV2BeaconBlockExecutionRequestWithdrawal())
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION", msg.GetEthV2BeaconBlockExecutionRequestConsolidation())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_BLOCK_REWARD", msg.GetEthV1BeaconBlockReward())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD", msg.GetEthV1BeaconAttestationReward())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD", msg.GetEthV1BeaconSyncCommitteeReward())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_RANDAO:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_STATE_RANDAO", msg.GetEthV1BeaconStateRandao())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT", msg.GetEthV1BeaconStateFinalityCheckpoint())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT", msg.GetEthV1BeaconStatePendingDeposit())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL", msg.GetEthV1BeaconStatePendingPartialWithdrawal())
	case xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION:
		return l.marshalLocationData("BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION", msg.GetEthV1BeaconStatePendingConsolidation())
	case xatu.CannonType_EXECUTION_CANONICAL_BLOCK:
		l.Type = "EXECUTION_CANONICAL_BLOCK"

		data := msg.GetExecutionCanonicalBlock()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION:
		l.Type = "EXECUTION_CANONICAL_TRANSACTION"

		data := msg.GetExecutionCanonicalTransaction()

		b, err := protojson.Marshal(data)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrFailedToMarshal, err)
		}

		l.Value = string(b)
	case xatu.CannonType_EXECUTION_CANONICAL_LOGS:
		return l.marshalLocationData("EXECUTION_CANONICAL_LOGS", msg.GetExecutionCanonicalLogs())
	case xatu.CannonType_EXECUTION_CANONICAL_TRACES:
		return l.marshalLocationData("EXECUTION_CANONICAL_TRACES", msg.GetExecutionCanonicalTraces())
	case xatu.CannonType_EXECUTION_CANONICAL_NATIVE_TRANSFERS:
		return l.marshalLocationData("EXECUTION_CANONICAL_NATIVE_TRANSFERS", msg.GetExecutionCanonicalNativeTransfers())
	case xatu.CannonType_EXECUTION_CANONICAL_ERC20_TRANSFERS:
		return l.marshalLocationData("EXECUTION_CANONICAL_ERC20_TRANSFERS", msg.GetExecutionCanonicalErc20Transfers())
	case xatu.CannonType_EXECUTION_CANONICAL_ERC721_TRANSFERS:
		return l.marshalLocationData("EXECUTION_CANONICAL_ERC721_TRANSFERS", msg.GetExecutionCanonicalErc721Transfers())
	case xatu.CannonType_EXECUTION_CANONICAL_CONTRACTS:
		return l.marshalLocationData("EXECUTION_CANONICAL_CONTRACTS", msg.GetExecutionCanonicalContracts())
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS:
		return l.marshalLocationData("EXECUTION_CANONICAL_BALANCE_DIFFS", msg.GetExecutionCanonicalBalanceDiffs())
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_DIFFS:
		return l.marshalLocationData("EXECUTION_CANONICAL_STORAGE_DIFFS", msg.GetExecutionCanonicalStorageDiffs())
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_DIFFS:
		return l.marshalLocationData("EXECUTION_CANONICAL_NONCE_DIFFS", msg.GetExecutionCanonicalNonceDiffs())
	case xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS:
		return l.marshalLocationData("EXECUTION_CANONICAL_BALANCE_READS", msg.GetExecutionCanonicalBalanceReads())
	case xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS:
		return l.marshalLocationData("EXECUTION_CANONICAL_STORAGE_READS", msg.GetExecutionCanonicalStorageReads())
	case xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS:
		return l.marshalLocationData("EXECUTION_CANONICAL_NONCE_READS", msg.GetExecutionCanonicalNonceReads())
	case xatu.CannonType_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS:
		return l.marshalLocationData("EXECUTION_CANONICAL_FOUR_BYTE_COUNTS", msg.GetExecutionCanonicalFourByteCounts())
	case xatu.CannonType_EXECUTION_CANONICAL_ADDRESS_APPEARANCES:
		return l.marshalLocationData("EXECUTION_CANONICAL_ADDRESS_APPEARANCES", msg.GetExecutionCanonicalAddressAppearances())
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST", msg.GetEthV2BeaconBlockAccessList())
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION", msg.GetEthV2BeaconBlockPayloadAttestation())
	case xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID:
		return l.marshalLocationData("BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID", msg.GetEthV2BeaconBlockExecutionPayloadBid())
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
	case "BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE

		data := &xatu.CannonLocationEthV1BeaconSyncCommittee{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconSyncCommittee{
			EthV1BeaconSyncCommittee: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_SYNC_AGGREGATE

		data := &xatu.CannonLocationEthV2BeaconBlockSyncAggregate{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockSyncAggregate{
			EthV2BeaconBlockSyncAggregate: data,
		}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_DEPOSIT

		data := &xatu.CannonLocationEthV2BeaconBlockExecutionRequestDeposit{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionRequestDeposit{EthV2BeaconBlockExecutionRequestDeposit: data}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_WITHDRAWAL

		data := &xatu.CannonLocationEthV2BeaconBlockExecutionRequestWithdrawal{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionRequestWithdrawal{EthV2BeaconBlockExecutionRequestWithdrawal: data}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_REQUEST_CONSOLIDATION

		data := &xatu.CannonLocationEthV2BeaconBlockExecutionRequestConsolidation{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionRequestConsolidation{EthV2BeaconBlockExecutionRequestConsolidation: data}
	case "BEACON_API_ETH_V1_BEACON_BLOCK_REWARD":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_BLOCK_REWARD

		data := &xatu.CannonLocationEthV1BeaconBlockReward{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconBlockReward{EthV1BeaconBlockReward: data}
	case "BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_ATTESTATION_REWARD

		data := &xatu.CannonLocationEthV1BeaconAttestationReward{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconAttestationReward{EthV1BeaconAttestationReward: data}
	case "BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_SYNC_COMMITTEE_REWARD

		data := &xatu.CannonLocationEthV1BeaconSyncCommitteeReward{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconSyncCommitteeReward{EthV1BeaconSyncCommitteeReward: data}
	case "BEACON_API_ETH_V1_BEACON_STATE_RANDAO":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_RANDAO

		data := &xatu.CannonLocationEthV1BeaconStateRandao{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconStateRandao{EthV1BeaconStateRandao: data}
	case "BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_FINALITY_CHECKPOINT

		data := &xatu.CannonLocationEthV1BeaconStateFinalityCheckpoint{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconStateFinalityCheckpoint{EthV1BeaconStateFinalityCheckpoint: data}
	case "BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_DEPOSIT

		data := &xatu.CannonLocationEthV1BeaconStatePendingDeposit{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconStatePendingDeposit{EthV1BeaconStatePendingDeposit: data}
	case "BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_PARTIAL_WITHDRAWAL

		data := &xatu.CannonLocationEthV1BeaconStatePendingPartialWithdrawal{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconStatePendingPartialWithdrawal{EthV1BeaconStatePendingPartialWithdrawal: data}
	case "BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V1_BEACON_STATE_PENDING_CONSOLIDATION

		data := &xatu.CannonLocationEthV1BeaconStatePendingConsolidation{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV1BeaconStatePendingConsolidation{EthV1BeaconStatePendingConsolidation: data}
	case "EXECUTION_CANONICAL_BLOCK":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_BLOCK

		data := &xatu.CannonLocationExecutionCanonicalBlock{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalBlock{
			ExecutionCanonicalBlock: data,
		}
	case "EXECUTION_CANONICAL_TRANSACTION":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_TRANSACTION

		data := &xatu.CannonLocationExecutionCanonicalTransaction{}

		err := protojson.Unmarshal([]byte(l.Value), data)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrFailedToUnmarshal, err)
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalTransaction{
			ExecutionCanonicalTransaction: data,
		}
	case "EXECUTION_CANONICAL_LOGS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_LOGS

		data := &xatu.CannonLocationExecutionCanonicalLogs{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalLogs{ExecutionCanonicalLogs: data}
	case "EXECUTION_CANONICAL_TRACES":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_TRACES

		data := &xatu.CannonLocationExecutionCanonicalTraces{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalTraces{ExecutionCanonicalTraces: data}
	case "EXECUTION_CANONICAL_NATIVE_TRANSFERS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_NATIVE_TRANSFERS

		data := &xatu.CannonLocationExecutionCanonicalNativeTransfers{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalNativeTransfers{ExecutionCanonicalNativeTransfers: data}
	case "EXECUTION_CANONICAL_ERC20_TRANSFERS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_ERC20_TRANSFERS

		data := &xatu.CannonLocationExecutionCanonicalErc20Transfers{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalErc20Transfers{ExecutionCanonicalErc20Transfers: data}
	case "EXECUTION_CANONICAL_ERC721_TRANSFERS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_ERC721_TRANSFERS

		data := &xatu.CannonLocationExecutionCanonicalErc721Transfers{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalErc721Transfers{ExecutionCanonicalErc721Transfers: data}
	case "EXECUTION_CANONICAL_CONTRACTS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_CONTRACTS

		data := &xatu.CannonLocationExecutionCanonicalContracts{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalContracts{ExecutionCanonicalContracts: data}
	case "EXECUTION_CANONICAL_BALANCE_DIFFS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_BALANCE_DIFFS

		data := &xatu.CannonLocationExecutionCanonicalBalanceDiffs{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalBalanceDiffs{ExecutionCanonicalBalanceDiffs: data}
	case "EXECUTION_CANONICAL_STORAGE_DIFFS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_STORAGE_DIFFS

		data := &xatu.CannonLocationExecutionCanonicalStorageDiffs{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalStorageDiffs{ExecutionCanonicalStorageDiffs: data}
	case "EXECUTION_CANONICAL_NONCE_DIFFS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_NONCE_DIFFS

		data := &xatu.CannonLocationExecutionCanonicalNonceDiffs{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalNonceDiffs{ExecutionCanonicalNonceDiffs: data}
	case "EXECUTION_CANONICAL_BALANCE_READS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_BALANCE_READS

		data := &xatu.CannonLocationExecutionCanonicalBalanceReads{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalBalanceReads{ExecutionCanonicalBalanceReads: data}
	case "EXECUTION_CANONICAL_STORAGE_READS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_STORAGE_READS

		data := &xatu.CannonLocationExecutionCanonicalStorageReads{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalStorageReads{ExecutionCanonicalStorageReads: data}
	case "EXECUTION_CANONICAL_NONCE_READS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_NONCE_READS

		data := &xatu.CannonLocationExecutionCanonicalNonceReads{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalNonceReads{ExecutionCanonicalNonceReads: data}
	case "EXECUTION_CANONICAL_FOUR_BYTE_COUNTS":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_FOUR_BYTE_COUNTS

		data := &xatu.CannonLocationExecutionCanonicalFourByteCounts{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalFourByteCounts{ExecutionCanonicalFourByteCounts: data}
	case "EXECUTION_CANONICAL_ADDRESS_APPEARANCES":
		msg.Type = xatu.CannonType_EXECUTION_CANONICAL_ADDRESS_APPEARANCES

		data := &xatu.CannonLocationExecutionCanonicalAddressAppearances{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_ExecutionCanonicalAddressAppearances{ExecutionCanonicalAddressAppearances: data}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_ACCESS_LIST

		data := &xatu.CannonLocationEthV2BeaconBlockAccessList{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockAccessList{EthV2BeaconBlockAccessList: data}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_PAYLOAD_ATTESTATION

		data := &xatu.CannonLocationEthV2BeaconBlockPayloadAttestation{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockPayloadAttestation{EthV2BeaconBlockPayloadAttestation: data}
	case "BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID":
		msg.Type = xatu.CannonType_BEACON_API_ETH_V2_BEACON_BLOCK_EXECUTION_PAYLOAD_BID

		data := &xatu.CannonLocationEthV2BeaconBlockExecutionPayloadBid{}

		if err := unmarshalLocationData(l.Value, data); err != nil {
			return nil, err
		}

		msg.Data = &xatu.CannonLocation_EthV2BeaconBlockExecutionPayloadBid{EthV2BeaconBlockExecutionPayloadBid: data}
	default:
		return nil, fmt.Errorf("unknown type: %s", l.Type)
	}

	return msg, nil
}

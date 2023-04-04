package xatu

import (
	"encoding/json"
	"strconv"

	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/encoding/protojson"
)

//nolint:gocyclo // Unfortunately required until we can refactor additional_data.
func (m *DecoratedEvent) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Event json.RawMessage `json:"event"`
		Data  json.RawMessage `json:"data,omitempty"`
		Meta  struct {
			Client json.RawMessage `json:"client,omitempty"`
			Server json.RawMessage `json:"server,omitempty"`
		} `json:"meta,omitempty"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	//nolint:goconst // No thanks
	validData := len(tmp.Data) != 0 && string(tmp.Data) != "null"

	// Also unmarshal the client meta field into a temporary struct to get the
	// additional_data field; which type depends on the event name.
	var clientMeta struct {
		//nolint:tagliatelle // Outside of our control
		AdditionalData json.RawMessage `json:"additional_data"`
	}

	validClientMeta := len(tmp.Meta.Client) != 0 && string(tmp.Meta.Client) != "null"

	event := new(Event)

	meta := new(ClientMeta)

	if validClientMeta {
		if err := json.Unmarshal(tmp.Meta.Client, &clientMeta); err != nil {
			return err
		}

		if err := json.Unmarshal(tmp.Meta.Client, meta); err != nil {
			return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta")
		}
	}

	if err := protojson.Unmarshal(tmp.Event, event); err != nil {
		return errors.Wrap(err, "failed to unmarshal DecoratedEvent.Event")
	}

	m.Event = event

	switch event.Name {
	case Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION:
		if validData {
			a := new(v1.Attestation)

			if err := protojson.Unmarshal(tmp.Data, a); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsAttestation")
			}

			m.Data = &DecoratedEvent_EthV1EventsAttestation{EthV1EventsAttestation: a}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsAttestationData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsAttestationData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsAttestation{EthV1EventsAttestation: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_BLOCK:
		if validData {
			b := new(v1.EventBlock)

			if err := protojson.Unmarshal(tmp.Data, b); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsBlock")
			}

			m.Data = &DecoratedEvent_EthV1EventsBlock{EthV1EventsBlock: b}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsBlockData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsBlockData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsBlock{EthV1EventsBlock: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG:
		if validData {
			cr := new(v1.EventChainReorg)

			if err := protojson.Unmarshal(tmp.Data, cr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsChainReorg")
			}

			m.Data = &DecoratedEvent_EthV1EventsChainReorg{EthV1EventsChainReorg: cr}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsChainReorgData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsChainReorgData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsChainReorg{EthV1EventsChainReorg: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT:
		if validData {
			fc := new(v1.EventFinalizedCheckpoint)

			if err := protojson.Unmarshal(tmp.Data, fc); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsFinalizedCheckpoint")
			}

			m.Data = &DecoratedEvent_EthV1EventsFinalizedCheckpoint{EthV1EventsFinalizedCheckpoint: fc}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsFinalizedCheckpointData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsFinalizedCheckpointData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsFinalizedCheckpoint{EthV1EventsFinalizedCheckpoint: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_HEAD:
		if validData {
			h := new(v1.EventHead)

			if err := protojson.Unmarshal(tmp.Data, h); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsHead")
			}

			m.Data = &DecoratedEvent_EthV1EventsHead{EthV1EventsHead: h}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsHeadData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsHeadData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsHead{EthV1EventsHead: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT:
		if validData {
			ve := new(v1.EventVoluntaryExit)

			if err := protojson.Unmarshal(tmp.Data, ve); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsVoluntaryExit")
			}

			m.Data = &DecoratedEvent_EthV1EventsVoluntaryExit{EthV1EventsVoluntaryExit: ve}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsVoluntaryExitData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsVoluntaryExitData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsVoluntaryExit{EthV1EventsVoluntaryExit: adr}
		}

	case Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF:
		if validData {
			cp := new(v1.EventContributionAndProof)

			if err := protojson.Unmarshal(tmp.Data, cp); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1EventsContributionAndProof")
			}

			m.Data = &DecoratedEvent_EthV1EventsContributionAndProof{EthV1EventsContributionAndProof: cp}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1EventsContributionAndProofData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1EventsContributionAndProofData")
			}

			meta.AdditionalData = &ClientMeta_EthV1EventsContributionAndProof{EthV1EventsContributionAndProof: adr}
		}

	case Event_MEMPOOL_TRANSACTION:
		if validData {
			tx := new(string)

			if err := json.Unmarshal(tmp.Data, &tx); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.MempoolTransaction")
			}

			m.Data = &DecoratedEvent_MempoolTransaction{MempoolTransaction: *tx}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalMempoolTransactionData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalMempoolTransactionData")
			}

			meta.AdditionalData = &ClientMeta_MempoolTransaction{MempoolTransaction: adr}
		}

	case Event_BEACON_API_ETH_V2_BEACON_BLOCK:
		if validData {
			bb := new(v2.EventBlock)

			if err := json.Unmarshal(tmp.Data, bb); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV2BeaconBlock")
			}

			m.Data = &DecoratedEvent_EthV2BeaconBlock{EthV2BeaconBlock: bb}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV2BeaconBlockData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV2BeaconBlockData")
			}

			meta.AdditionalData = &ClientMeta_EthV2BeaconBlock{EthV2BeaconBlock: adr}
		}

	case Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE:
		if validData {
			fc := new(v1.ForkChoice)

			if err := protojson.Unmarshal(tmp.Data, fc); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1DebugForkChoice")
			}

			m.Data = &DecoratedEvent_EthV1ForkChoice{EthV1ForkChoice: fc}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1DebugForkChoiceData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1DebugForkChoiceData")
			}

			meta.AdditionalData = &ClientMeta_EthV1DebugForkChoice{EthV1DebugForkChoice: adr}
		}

	case Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG:
		if validData {
			fcr := new(DebugForkChoiceReorg)

			if err := protojson.Unmarshal(tmp.Data, fcr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.EthV1DebugForkChoiceReorg")
			}

			m.Data = &DecoratedEvent_EthV1ForkChoiceReorg{EthV1ForkChoiceReorg: fcr}
		}

		if validClientMeta {
			adr := new(ClientMeta_AdditionalEthV1DebugForkChoiceReOrgData)

			if err := protojson.Unmarshal(clientMeta.AdditionalData, adr); err != nil {
				return errors.Wrap(err, "failed to unmarshal DecoratedEvent.ClientMeta_AdditionalEthV1DebugForkChoiceReOrgData")
			}

			meta.AdditionalData = &ClientMeta_EthV1DebugForkChoiceReorg{EthV1DebugForkChoiceReorg: adr}
		}

	default:
		return errors.New("no data field specified for DecoratedEvent")
	}

	serverMeta := new(ServerMeta)

	validServerMeta := len(tmp.Meta.Server) != 0 && string(tmp.Meta.Server) != "null"

	if validServerMeta {
		if err := protojson.Unmarshal(tmp.Meta.Server, serverMeta); err != nil {
			return errors.Wrap(err, "failed to unmarshal DecoratedEvent.Meta.Server")
		}
	}

	m.Meta = &Meta{
		Client: meta,
		Server: serverMeta,
	}

	return nil
}

func (c *ClientMeta) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name           string `json:"name,omitempty"`
		Version        string `json:"version,omitempty"`
		ID             string ` json:"id,omitempty"`
		Implementation string `json:"implementation,omitempty"`
		Os             string `json:"os,omitempty"`
		//nolint:tagliatelle // Outside of our control
		ClockDrift string               `json:"clock_drift,omitempty"`
		Ethereum   *ClientMeta_Ethereum `json:"ethereum,omitempty"`
		Labels     map[string]string    `json:"labels,omitempty"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return errors.Wrap(err, "failed to unmarshal ClientMeta")
	}

	c.Name = tmp.Name
	c.Version = tmp.Version
	c.Id = tmp.ID
	c.Implementation = tmp.Implementation
	c.Os = tmp.Os

	// Parse clock drift as milliseconds

	clockDrift, err := strconv.ParseUint(tmp.ClockDrift, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse ClientMeta.ClockDrift")
	}

	c.ClockDrift = clockDrift
	c.Ethereum = tmp.Ethereum

	return nil
}

func (c *ClientMeta_Ethereum_Network) UnmarshalJSON(data []byte) error {
	var tmp struct {
		Name string `json:"name,omitempty"`
		ID   string `json:"id,omitempty"`
	}

	if err := json.Unmarshal(data, &tmp); err != nil {
		return errors.Wrap(err, "failed to unmarshal ClientMeta_Ethereum_Network")
	}

	c.Name = tmp.Name

	id, err := strconv.ParseUint(tmp.ID, 10, 64)
	if err != nil {
		return errors.Wrap(err, "failed to parse ClientMeta_Ethereum_Network.Id")
	}

	c.Id = id

	return nil
}

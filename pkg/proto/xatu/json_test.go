package xatu

import (
	"encoding/json"
	reflect "reflect"
	"testing"
	"time"

	v1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	v2 "github.com/ethpandaops/xatu/pkg/proto/eth/v2"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/protojson"
	timestamppb "google.golang.org/protobuf/types/known/timestamppb"
)

func TestDecoratedEvent_UnmarshalJSON(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		d := []byte(`{"event":{"name":"BEACON_API_ETH_V1_EVENTS_BLOCK","date_time":"2023-03-27T02:32:02.245267282Z"},"meta":{"client":{"name":"example-instance","version":"dev-dev","id":"e8973fb1-d81e-4d5f-a924-db02cacab700","implementation":"Xatu","os":"darwin","clock_drift":"263","ethereum":{"network":{"name":"sepolia","id":"11155111"},"execution":{},"consensus":{"implementation":"teku","version":"teku/vUNKNOWN+g20fcf48/linux-x86_64/-eclipseadoptium-openjdk64bitservervm-java-17"}},"labels":{"ethpandaops":"rocks"},"additional_data":{"epoch":{"number":"62892","start_date_time":"2023-03-27T02:28:48Z"},"slot":{"number":"2012560","start_date_time":"2023-03-27T02:32:00Z"},"propagation":{"slot_start_diff":"2245"}}}},"data":{"slot":"2012560","block":"0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87"}}`)

		m := new(DecoratedEvent)

		err := m.UnmarshalJSON(d)

		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
	})

	t.Run("InvalidEventName", func(t *testing.T) {
		d := []byte(`{
					"event": {
							"name": "INVALID_EVENT_NAME"
					},
					"meta": {}
			}`)

		m := new(DecoratedEvent)

		err := m.UnmarshalJSON(d)

		if err == nil {
			t.Fatal("Expected an error")
		}

		assert.ErrorContains(t, err, "invalid value for enum type: \"INVALID_EVENT_NAME\"")
	})

	t.Run("UnknownKey", func(t *testing.T) {
		d := []byte(`{
					"event": {
							"name": "BEACON_API_ETH_V1_EVENTS_BLOCK"
					},
					"will_never_be_a_json_key": {},
					"meta": {}
			}`)

		m := new(DecoratedEvent)

		err := m.UnmarshalJSON(d)

		if err != nil {
			t.Fatal("Did not expect an error")
		}
	})

	// Note: If this test is failing it means that a new event has been added to the Event enum.
	// 		 In this case, the new event will need to be added to the switch statement in the
	// 		 DecoratedEvent.UnmarshalJSON() method.
	// May go
	t.Run("Marshal/Unmarshal", func(t *testing.T) {
		for _, id := range Event_Name_value {
			eventName := Event_Name(id)

			if eventName == Event_BEACON_API_ETH_V1_EVENTS_UNKNOWN {
				continue
			}

			decoratedEvent := new(DecoratedEvent)

			decoratedEvent.Event = &Event{
				Name:     eventName,
				DateTime: timestamppb.New(time.Now()),
			}

			// Marshal the decorated event to JSON.
			b, err := protojson.Marshal(decoratedEvent)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Unmarshal the decorated event from JSON.
			m := new(DecoratedEvent)

			err = json.Unmarshal(b, m)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			assert.Equal(t, decoratedEvent.Event.Name, m.Event.Name)
		}
	})

	t.Run("Marshal/Unmarshal with data", func(t *testing.T) {
		for _, id := range Event_Name_value {
			eventName := Event_Name(id)

			if eventName == Event_BEACON_API_ETH_V1_EVENTS_UNKNOWN {
				continue
			}

			decoratedEvent := new(DecoratedEvent)

			decoratedEvent.Event = &Event{
				Name:     eventName,
				DateTime: timestamppb.New(time.Now()),
			}

			decoratedEvent.Meta = &Meta{
				Client: &ClientMeta{
					Name:           "example-instance",
					Version:        "dev-dev",
					Id:             "e8973fb1-d81e-4d5f-a924-db02cacab700",
					Implementation: "xatu",
					Os:             "darwin",
					ClockDrift:     263,
					Ethereum: &ClientMeta_Ethereum{
						Network: &ClientMeta_Ethereum_Network{
							Name: "sepolia",
							Id:   11155111,
						},
						Execution: &ClientMeta_Ethereum_Execution{},
						Consensus: &ClientMeta_Ethereum_Consensus{
							Implementation: "teku",
							Version:        "teku/vUNKNOWN+g20fcf48/linux-x86_64/-eclipseadoptium-openjdk64bitservervm-java-17",
						},
					},
					Labels: map[string]string{
						"ethpandaops": "rocks",
					},
				},
			}

			switch eventName {
			case Event_BEACON_API_ETH_V1_EVENTS_BLOCK:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsBlock{
					EthV1EventsBlock: &v1.EventBlock{
						Slot:  2012560,
						Block: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsBlock{
					EthV1EventsBlock: &ClientMeta_AdditionalEthV1EventsBlockData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Slot: &Slot{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Propagation: &Propagation{
							SlotStartDiff: 400,
						},
					},
				}
			case Event_BEACON_API_ETH_V1_EVENTS_ATTESTATION:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsAttestation{
					EthV1EventsAttestation: &v1.Attestation{
						Signature: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
						Data: &v1.AttestationData{
							Slot: 2012560,
						},
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsAttestation{
					EthV1EventsAttestation: &ClientMeta_AdditionalEthV1EventsAttestationData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Slot: &Slot{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Propagation: &Propagation{
							SlotStartDiff: 400,
						},
					},
				}

			case Event_BEACON_API_ETH_V1_EVENTS_CHAIN_REORG:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsChainReorg{
					EthV1EventsChainReorg: &v1.EventChainReorg{
						Depth: 4,
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsChainReorg{
					EthV1EventsChainReorg: &ClientMeta_AdditionalEthV1EventsChainReorgData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Slot: &Slot{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Propagation: &Propagation{
							SlotStartDiff: 400,
						},
					},
				}

			case Event_BEACON_API_ETH_V1_EVENTS_FINALIZED_CHECKPOINT:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsFinalizedCheckpoint{
					EthV1EventsFinalizedCheckpoint: &v1.EventFinalizedCheckpoint{
						Epoch: 2012560,
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsFinalizedCheckpoint{
					EthV1EventsFinalizedCheckpoint: &ClientMeta_AdditionalEthV1EventsFinalizedCheckpointData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
					},
				}
			case Event_BEACON_API_ETH_V1_EVENTS_HEAD:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsHead{
					EthV1EventsHead: &v1.EventHead{
						Slot: 2012560,
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsHead{
					EthV1EventsHead: &ClientMeta_AdditionalEthV1EventsHeadData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Slot: &Slot{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Propagation: &Propagation{
							SlotStartDiff: 400,
						},
					},
				}
			case Event_BEACON_API_ETH_V1_EVENTS_CONTRIBUTION_AND_PROOF:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsContributionAndProof{
					EthV1EventsContributionAndProof: &v1.EventContributionAndProof{
						Signature: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsContributionAndProof{
					EthV1EventsContributionAndProof: &ClientMeta_AdditionalEthV1EventsContributionAndProofData{
						Contribution: &ClientMeta_AdditionalEthV1EventsContributionAndProofContributionData{
							Epoch: &Epoch{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
							Slot: &Slot{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
							Propagation: &Propagation{
								SlotStartDiff: 400,
							},
						},
					},
				}
			case Event_BEACON_API_ETH_V1_EVENTS_VOLUNTARY_EXIT:
				decoratedEvent.Data = &DecoratedEvent_EthV1EventsVoluntaryExit{
					EthV1EventsVoluntaryExit: &v1.EventVoluntaryExit{
						ValidatorIndex: 1,
						Epoch:          5,
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1EventsVoluntaryExit{
					EthV1EventsVoluntaryExit: &ClientMeta_AdditionalEthV1EventsVoluntaryExitData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
					},
				}
			case Event_MEMPOOL_TRANSACTION:
				decoratedEvent.Data = &DecoratedEvent_MempoolTransaction{
					MempoolTransaction: "{}",
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_MempoolTransaction{
					MempoolTransaction: &ClientMeta_AdditionalMempoolTransactionData{
						Hash: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
					},
				}

			case Event_BEACON_API_ETH_V2_BEACON_BLOCK:
				decoratedEvent.Data = &DecoratedEvent_EthV2BeaconBlock{
					EthV2BeaconBlock: &v2.EventBlock{
						Version:   v2.BlockVersion_BELLATRIX,
						Signature: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
						Message: &v2.EventBlock_BellatrixBlock{
							BellatrixBlock: &v2.BeaconBlockBellatrix{
								ParentRoot:    "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca87",
								ProposerIndex: 1,
								StateRoot:     "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca86",
								Slot:          2012560,
								Body: &v2.BeaconBlockBodyBellatrix{
									Eth1Data: &v1.Eth1Data{
										DepositRoot:  "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca85",
										DepositCount: 1,
										BlockHash:    "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca84",
									},
									RandaoReveal:      "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca83",
									Graffiti:          "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca82",
									ProposerSlashings: nil,
									AttesterSlashings: nil,
									Deposits:          nil,
									VoluntaryExits:    nil,
									SyncAggregate: &v1.SyncAggregate{
										SyncCommitteeBits:      "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca81",
										SyncCommitteeSignature: "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca80",
									},
									ExecutionPayload: &v1.ExecutionPayload{
										StateRoot:     "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7f",
										BlockNumber:   1,
										BlockHash:     "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7e",
										ParentHash:    "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7d",
										FeeRecipient:  "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7c",
										ReceiptsRoot:  "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7b",
										LogsBloom:     "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca7a",
										PrevRandao:    "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca79",
										GasLimit:      1,
										GasUsed:       1,
										Timestamp:     1,
										ExtraData:     "0x2506f42e292de118ace069902e27daa21f2b69ae003afc3ab937254a4f1aca78",
										BaseFeePerGas: "2",
										Transactions:  []string{"123", "321"},
									},
								},
							},
						},
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV2BeaconBlock{
					EthV2BeaconBlock: &ClientMeta_AdditionalEthV2BeaconBlockData{
						Epoch: &Epoch{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Slot: &Slot{
							Number: 2012560,
							StartDateTime: &timestamppb.Timestamp{
								Seconds: 1630000000,
							},
						},
						Version: "BELLATRIX",
					},
				}

			case Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE:
				decoratedEvent.Data = &DecoratedEvent_EthV1ForkChoice{
					EthV1ForkChoice: &v1.ForkChoice{
						JustifiedCheckpoint: &v1.Checkpoint{
							Epoch: 2012560,
						},
						FinalizedCheckpoint: &v1.Checkpoint{
							Epoch: 2012560,
						},
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1DebugForkChoice{
					EthV1DebugForkChoice: &ClientMeta_AdditionalEthV1DebugForkChoiceData{
						Snapshot: &ClientMeta_ForkChoiceSnapshot{
							RequestedAtSlotStartDiffMs: 1,
							RequestDurationMs:          500,
							RequestEpoch: &Epoch{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
							RequestSlot: &Slot{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
						},
					},
				}

			case Event_BEACON_API_ETH_V1_DEBUG_FORK_CHOICE_REORG:
				decoratedEvent.Data = &DecoratedEvent_EthV1ForkChoiceReorg{
					EthV1ForkChoiceReorg: &DebugForkChoiceReorg{
						Before: &v1.ForkChoice{
							JustifiedCheckpoint: &v1.Checkpoint{
								Epoch: 2012560,
							},
							FinalizedCheckpoint: &v1.Checkpoint{
								Epoch: 2012560,
							},
						},
						After: &v1.ForkChoice{
							JustifiedCheckpoint: &v1.Checkpoint{
								Epoch: 2012560,
							},
							FinalizedCheckpoint: &v1.Checkpoint{
								Epoch: 2012560,
							},
						},
					},
				}

				decoratedEvent.Meta.Client.AdditionalData = &ClientMeta_EthV1DebugForkChoiceReorg{
					EthV1DebugForkChoiceReorg: &ClientMeta_AdditionalEthV1DebugForkChoiceReOrgData{
						After: &ClientMeta_ForkChoiceSnapshot{
							RequestedAtSlotStartDiffMs: 1,
							RequestDurationMs:          500,
							RequestEpoch: &Epoch{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
							RequestSlot: &Slot{
								Number: 2012560,
								StartDateTime: &timestamppb.Timestamp{
									Seconds: 1630000000,
								},
							},
						},
					},
				}
			}

			// Marshal the decorated event to JSON.
			b, err := protojson.Marshal(decoratedEvent)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Unmarshal the decorated event from JSON.
			m := new(DecoratedEvent)

			err = json.Unmarshal(b, m)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			assert.EqualValues(t, decoratedEvent.GetMeta().Client.Name, m.GetMeta().Client.Name)
			assert.EqualValues(t, decoratedEvent.GetMeta().Client.Version, m.GetMeta().Client.Version)

			if eventName != Event_BEACON_API_ETH_V2_BEACON_BLOCK {
				if !reflect.DeepEqual(decoratedEvent.GetData(), m.GetData()) {
					t.Fatalf("Unexpected result, expected %v, got %v", decoratedEvent.GetData(), m.GetData())
				}
			} else {
				// Special case for beacon block, as the data is a lot more complex.
				// We'll just manually check the fields we care about.
				assert.EqualValues(t, decoratedEvent.GetEthV2BeaconBlock().Message, m.GetEthV2BeaconBlock().Message)
				assert.EqualValues(t, decoratedEvent.GetEthV2BeaconBlock().Signature, m.GetEthV2BeaconBlock().Signature)
				assert.EqualValues(t, decoratedEvent.GetEthV2BeaconBlock().Version, m.GetEthV2BeaconBlock().Version)
			}
		}
	})
}

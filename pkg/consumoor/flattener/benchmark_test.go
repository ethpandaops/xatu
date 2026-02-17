package flattener_test

import (
	"testing"
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/flattener"
	tabledefs "github.com/ethpandaops/xatu/pkg/consumoor/flattener/table"
	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	ethv1 "github.com/ethpandaops/xatu/pkg/proto/eth/v1"
	libp2p "github.com/ethpandaops/xatu/pkg/proto/libp2p"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func BenchmarkFlattenerHead(b *testing.B) {
	route := benchmarkFindRouteByTable(b, "beacon_api_eth_v1_events_head")
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_EVENTS_HEAD_V2,
			DateTime: timestamppb.Now(),
			Id:       "bench-head",
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "bench-client",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{
						Name: "mainnet",
						Id:   1,
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_EthV1EventsHead{
			EthV1EventsHead: &ethv1.EventHead{
				Slot:                      10_000_000,
				Block:                     "0xabc",
				State:                     "0xdef",
				EpochTransition:           false,
				PreviousDutyDependentRoot: "0x111",
				CurrentDutyDependentRoot:  "0x222",
			},
		},
	}
	meta := metadata.Extract(event)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := route.Flatten(event, meta); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlattenerLibP2PConnected(b *testing.B) {
	route := benchmarkFindRouteByTable(b, "libp2p_connected")
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_LIBP2P_TRACE_CONNECTED,
			DateTime: timestamppb.Now(),
			Id:       "bench-libp2p",
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "bench-client",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{
						Name: "mainnet",
						Id:   1,
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_Libp2PTraceConnected{
			Libp2PTraceConnected: &libp2p.Connected{
				RemotePeer:   wrapperspb.String("16Uiu2HAmMZ5Hf5fY5sFf2tN4w5v9nSzNnW2bX"),
				RemoteMaddrs: wrapperspb.String("/ip4/1.2.3.4/tcp/9000"),
			},
		},
	}
	meta := metadata.Extract(event)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := route.Flatten(event, meta); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFlattenerValidatorsFanoutPubkeys(b *testing.B) {
	route := benchmarkFindRouteByTable(b, "canonical_beacon_validators_pubkeys")
	event := &xatu.DecoratedEvent{
		Event: &xatu.Event{
			Name:     xatu.Event_BEACON_API_ETH_V1_BEACON_VALIDATORS,
			DateTime: timestamppb.Now(),
			Id:       "bench-validators",
		},
		Meta: &xatu.Meta{
			Client: &xatu.ClientMeta{
				Name: "bench-client",
				Ethereum: &xatu.ClientMeta_Ethereum{
					Network: &xatu.ClientMeta_Ethereum_Network{
						Name: "mainnet",
						Id:   1,
					},
				},
				AdditionalData: &xatu.ClientMeta_EthV1Validators{
					EthV1Validators: &xatu.ClientMeta_AdditionalEthV1ValidatorsData{
						Epoch: &xatu.EpochV2{
							Number:        wrapperspb.UInt64(123),
							StartDateTime: timestamppb.New(time.Unix(1_700_000_000, 0)),
						},
					},
				},
			},
		},
		Data: &xatu.DecoratedEvent_EthV1Validators{
			EthV1Validators: &xatu.Validators{
				Validators: []*ethv1.Validator{
					{
						Index: wrapperspb.UInt64(1),
						Data: &ethv1.ValidatorData{
							Pubkey: wrapperspb.String("0x01"),
						},
					},
					{
						Index: wrapperspb.UInt64(2),
						Data: &ethv1.ValidatorData{
							Pubkey: wrapperspb.String("0x02"),
						},
					},
					{
						Index: wrapperspb.UInt64(3),
						Data: &ethv1.ValidatorData{
							Pubkey: wrapperspb.String("0x03"),
						},
					},
				},
			},
		},
	}
	meta := metadata.Extract(event)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := route.Flatten(event, meta); err != nil {
			b.Fatal(err)
		}
	}
}

func benchmarkFindRouteByTable(b *testing.B, table string) flattener.Route {
	b.Helper()

	for _, route := range tabledefs.All() {
		if route.TableName() == table {
			return route
		}
	}

	b.Fatalf("route for table %s not found", table)

	return nil
}

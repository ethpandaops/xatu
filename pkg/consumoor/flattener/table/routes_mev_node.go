package table

import "github.com/ethpandaops/xatu/pkg/consumoor/flattener"

func mevAndNodeRoutes() []flattener.Route {
	return []flattener.Route{
		MevRelayBidTraceRoute{}.Build(),
		MevRelayProposerPayloadDeliveredRoute{}.Build(),
		MevRelayValidatorRegistrationRoute{}.Build(),
		NodeRecordConsensusRoute{}.Build(),
		NodeRecordExecutionRoute{}.Build(),
	}
}

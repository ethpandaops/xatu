package clmimicry

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
)

// HandleMetadataEvent represents a metadata exchange event in the request/response protocol.
type HandleMetadataEvent struct {
	TraceEventBase
	ProtocolID        protocol.ID
	LatencyS          float64
	SeqNumber         uint64
	Attnets           string
	Syncnets          string
	CustodyGroupCount uint64
	Error             string
	Direction         string
}

// StatusData contains the status information exchanged in a status request/response.
type StatusData struct {
	ForkDigest            string
	FinalizedRoot         string
	FinalizedEpoch        uint64
	HeadRoot              string
	HeadSlot              uint64
	EarliestAvailableSlot uint64
}

// HandleStatusEvent represents a status exchange event in the request/response protocol.
type HandleStatusEvent struct {
	TraceEventBase
	ProtocolID protocol.ID
	LatencyS   float64
	Direction  string
	Request    *StatusData
	Response   *StatusData
	Error      string
}

// CustodyProbeEvent represents a data column custody probe event.
type CustodyProbeEvent struct {
	TraceEventBase
	JobStartTimestamp time.Time
	PeerIDStr         string // PeerID as string (different from TraceEventBase.PeerID)
	Slot              uint64
	Epoch             uint64
	ColumnIndex       uint64
	Result            string
	DurationMs        int64
	Error             string
	BlockHash         string
	ColumnSize        int
}

// Compile-time interface compliance checks.
var (
	_ TraceEvent = (*HandleMetadataEvent)(nil)
	_ TraceEvent = (*HandleStatusEvent)(nil)
	_ TraceEvent = (*CustodyProbeEvent)(nil)
)

// NewHandleMetadataEvent creates a new HandleMetadataEvent with the provided fields.
func NewHandleMetadataEvent(
	timestamp time.Time,
	peerID peer.ID,
	protocolID protocol.ID,
	latencyS float64,
	seqNumber uint64,
	attnets string,
	syncnets string,
	custodyGroupCount uint64,
	errorStr string,
	direction string,
) *HandleMetadataEvent {
	return &HandleMetadataEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		ProtocolID:        protocolID,
		LatencyS:          latencyS,
		SeqNumber:         seqNumber,
		Attnets:           attnets,
		Syncnets:          syncnets,
		CustodyGroupCount: custodyGroupCount,
		Error:             errorStr,
		Direction:         direction,
	}
}

// NewHandleStatusEvent creates a new HandleStatusEvent with the provided fields.
func NewHandleStatusEvent(
	timestamp time.Time,
	peerID peer.ID,
	protocolID protocol.ID,
	latencyS float64,
	direction string,
	request *StatusData,
	response *StatusData,
	errorStr string,
) *HandleStatusEvent {
	return &HandleStatusEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		ProtocolID: protocolID,
		LatencyS:   latencyS,
		Direction:  direction,
		Request:    request,
		Response:   response,
		Error:      errorStr,
	}
}

// NewCustodyProbeEvent creates a new CustodyProbeEvent with the provided fields.
func NewCustodyProbeEvent(
	timestamp time.Time,
	peerID peer.ID,
	jobStartTimestamp time.Time,
	peerIDStr string,
	slot uint64,
	epoch uint64,
	columnIndex uint64,
	result string,
	durationMs int64,
	errorStr string,
	blockHash string,
	columnSize int,
) *CustodyProbeEvent {
	return &CustodyProbeEvent{
		TraceEventBase: TraceEventBase{
			Timestamp: timestamp,
			PeerID:    peerID,
		},
		JobStartTimestamp: jobStartTimestamp,
		PeerIDStr:         peerIDStr,
		Slot:              slot,
		Epoch:             epoch,
		ColumnIndex:       columnIndex,
		Result:            result,
		DurationMs:        durationMs,
		Error:             errorStr,
		BlockHash:         blockHash,
		ColumnSize:        columnSize,
	}
}

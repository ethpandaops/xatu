package libp2p

import (
	"time"

	"github.com/ethpandaops/xatu/pkg/consumoor/metadata"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
)

// UpsertPeerRow keeps only rows that identify a peer and normalizes the
// canonical keys used by libp2p_peer table upserts.
func UpsertPeerRow(
	_ *xatu.DecoratedEvent,
	_ *metadata.CommonMetadata,
	row map[string]any,
) ([]map[string]any, error) {
	peerID := firstNonEmpty(row, "remote_peer", "peer_id")
	if peerID == "" {
		return nil, nil
	}

	row["peer_id"] = peerID
	row["unique_key"] = hashKey(peerID + asString(row["meta_network_name"]))
	row["updated_date_time"] = time.Now().Unix()

	return []map[string]any{row}, nil
}

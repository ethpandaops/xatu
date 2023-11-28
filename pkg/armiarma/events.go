package armiarma

// func (a *Armiarma) subscribeToEvents(ctx context.Context) error {
// 	a.log.Debug("Subscribing to events")

// 	// Subscribe to events
// 	a.crawler.OnAttestation(a.handleAttestationEvent)

// 	return nil
// }

// func (a *Armiarma) handleAttestationEvent(event *crawler.AttestationReceievedEvent) {
// 	a.log.WithFields(logrus.Fields{
// 		"peer_ip":               event.HostInfo.IP,
// 		"slot":                  event.Attestation.Data.Slot,
// 		"index":                 event.Attestation.Data.Index,
// 		"beacon_block_root":     event.Attestation.Data.BeaconBlockRoot,
// 		"arrival_time_in_slot":  event.TrackedAttestation.TimeInSlot.String(),
// 		"peer_client":           event.HostInfo.PeerInfo.UserAgent,
// 		"peer_latency":          event.HostInfo.PeerInfo.Latency.String(),
// 		"peer_protocol_version": event.HostInfo.PeerInfo.ProtocolVersion,
// 		"subnet":                event.TrackedAttestation.Subnet,
// 	}).Info("Got attestation for xatu")
// }

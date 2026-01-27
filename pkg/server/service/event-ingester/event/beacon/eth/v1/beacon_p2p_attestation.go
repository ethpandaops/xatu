package v1

import (
	"context"
	"errors"
	"net"

	"github.com/ethpandaops/xatu/pkg/geoip"
	"github.com/ethpandaops/xatu/pkg/geoip/lookup"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/sirupsen/logrus"
)

const (
	BeaconP2PAttestationType = "BEACON_P2P_ATTESTATION"
)

type BeaconP2PAttestation struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewBeaconP2PAttestation(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *BeaconP2PAttestation {
	return &BeaconP2PAttestation{
		log:           log.WithField("event", BeaconP2PAttestationType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (b *BeaconP2PAttestation) Type() string {
	return BeaconP2PAttestationType
}

func (b *BeaconP2PAttestation) Validate(_ context.Context) error {
	_, ok := b.event.GetData().(*xatu.DecoratedEvent_BeaconP2PAttestation)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *BeaconP2PAttestation) Filter(_ context.Context) bool {
	return false
}

func (b *BeaconP2PAttestation) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	additionalData, ok := b.event.Meta.Client.AdditionalData.(*xatu.ClientMeta_BeaconP2PAttestation)
	if !ok {
		b.log.Error("failed to cast client additional data")

		return meta
	}

	ipAddress := additionalData.BeaconP2PAttestation.GetPeer().GetIp()

	if ipAddress != "" {
		ip := net.ParseIP(ipAddress)
		if ip != nil && b.geoipProvider != nil {
			geoipLookupResult, err := b.geoipProvider.LookupIP(ctx, ip, lookup.PrecisionFull)
			if err != nil {
				b.log.WithField("ip", ipAddress).WithError(err).Warn("failed to lookup geoip data")
			}

			if geoipLookupResult != nil {
				meta.AdditionalData = &xatu.ServerMeta_BEACON_P2P_ATTESTATION{
					BEACON_P2P_ATTESTATION: &xatu.ServerMeta_AdditionalBeaconP2PAttestationData{
						Peer: &xatu.ServerMeta_Peer{
							Geo: &xatu.ServerMeta_Geo{
								Country:                      geoipLookupResult.CountryName,
								CountryCode:                  geoipLookupResult.CountryCode,
								City:                         geoipLookupResult.CityName,
								Latitude:                     geoipLookupResult.Latitude,
								Longitude:                    geoipLookupResult.Longitude,
								ContinentCode:                geoipLookupResult.ContinentCode,
								AutonomousSystemNumber:       geoipLookupResult.AutonomousSystemNumber,
								AutonomousSystemOrganization: geoipLookupResult.AutonomousSystemOrganization,
							},
						},
					},
				}
			}
		}
	}

	return meta
}

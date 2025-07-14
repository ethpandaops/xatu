package noderecord

import (
	"context"
	"errors"
	"net"

	coreenr "github.com/ethpandaops/ethcore/pkg/ethereum/node/enr"
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/ethpandaops/xatu/pkg/server/geoip"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var (
	ConsensusType = xatu.Event_NODE_RECORD_CONSENSUS.String()
)

type Consensus struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewConsensus(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *Consensus {
	return &Consensus{
		log:           log.WithField("event", ConsensusType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (b *Consensus) Type() string {
	return ConsensusType
}

func (b *Consensus) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_NodeRecordConsensus)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Consensus) Filter(ctx context.Context) bool {
	return false
}

func (b *Consensus) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	// Extract ENR from the consensus event data.
	consensusData := b.event.GetNodeRecordConsensus()
	if consensusData == nil {
		b.log.Error("failed to get consensus data from event")

		return meta
	}

	// Get ENR string from the proto data.
	enrString := ""
	if consensusData.GetEnr() != nil {
		enrString = consensusData.GetEnr().GetValue()
	}

	if enrString == "" {
		b.log.Debug("no ENR data available for IP extraction")

		return meta
	}

	// Parse ENR to extract IP address.
	parsedENR, err := coreenr.Parse(enrString)
	if err != nil {
		b.log.WithError(err).WithField("enr", enrString).Error("failed to parse ENR")

		return meta
	}

	// Try to get IP + port(s) from ENR (IPv4 first, then IPv6).
	ipString, ports := extractPorts(parsedENR)
	tcpPort := ports.tcp
	udpPort := ports.udp
	quicPort := ports.quic

	if ipString == "" {
		b.log.Debug("no IP address found in ENR")

		return meta
	}

	// Validate and parse IP address.
	ip := net.ParseIP(ipString)
	if ip == nil {
		b.log.WithField("ip", ipString).Error("failed to parse IP address")

		return meta
	}

	// Populate IP + port fields in consensus event data.
	consensusData.Ip = wrapperspb.String(ipString)
	consensusData.Tcp = wrapperspb.UInt32(tcpPort)
	consensusData.Udp = wrapperspb.UInt32(udpPort)
	consensusData.Quic = wrapperspb.UInt32(quicPort)

	// Perform GeoIP lookup if provider is available.
	if b.geoipProvider != nil {
		geoipResult, err := b.geoipProvider.LookupIP(ctx, ip)
		if err != nil {
			b.log.WithField("ip", ipString).WithError(err).Warn("failed to lookup geoip data")

			return meta
		}

		if geoipResult != nil {
			meta.AdditionalData = &xatu.ServerMeta_NODE_RECORD_CONSENSUS{
				NODE_RECORD_CONSENSUS: &xatu.ServerMeta_AdditionalNodeRecordConsensusData{
					Geo: &xatu.ServerMeta_Geo{
						Country:                      geoipResult.CountryName,
						CountryCode:                  geoipResult.CountryCode,
						City:                         geoipResult.CityName,
						Latitude:                     geoipResult.Latitude,
						Longitude:                    geoipResult.Longitude,
						ContinentCode:                geoipResult.ContinentCode,
						AutonomousSystemNumber:       geoipResult.AutonomousSystemNumber,
						AutonomousSystemOrganization: geoipResult.AutonomousSystemOrganization,
					},
				},
			}

			b.log.WithFields(logrus.Fields{
				"ip":      ipString,
				"country": geoipResult.CountryName,
				"city":    geoipResult.CityName,
			}).Debug("successfully updated server meta with consensus node geo information")
		}
	}

	return meta
}

type portConfig struct {
	tcp, udp, quic uint32
}

func extractPorts(parsedENR *coreenr.ENR) (ipString string, ports portConfig) {
	if parsedENR.IP4 != nil {
		return *parsedENR.IP4, portConfig{
			tcp:  safeDeref(parsedENR.TCP4),
			udp:  safeDeref(parsedENR.UDP4),
			quic: safeDeref(parsedENR.QUIC4),
		}
	}

	if parsedENR.IP6 != nil {
		return *parsedENR.IP6, portConfig{
			tcp:  safeDeref(parsedENR.TCP6),
			udp:  safeDeref(parsedENR.UDP6),
			quic: safeDeref(parsedENR.QUIC6),
		}
	}

	return "", portConfig{}
}

func safeDeref[T any](ptr *T) T {
	if ptr != nil {
		return *ptr
	}

	var zero T

	return zero
}

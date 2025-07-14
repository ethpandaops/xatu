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
	ExecutionType = xatu.Event_NODE_RECORD_EXECUTION.String()
)

type Execution struct {
	log           logrus.FieldLogger
	event         *xatu.DecoratedEvent
	geoipProvider geoip.Provider
}

func NewExecution(log logrus.FieldLogger, event *xatu.DecoratedEvent, geoipProvider geoip.Provider) *Execution {
	return &Execution{
		log:           log.WithField("event", ExecutionType),
		event:         event,
		geoipProvider: geoipProvider,
	}
}

func (b *Execution) Type() string {
	return ExecutionType
}

func (b *Execution) Validate(ctx context.Context) error {
	_, ok := b.event.Data.(*xatu.DecoratedEvent_NodeRecordExecution)
	if !ok {
		return errors.New("failed to cast event data")
	}

	return nil
}

func (b *Execution) Filter(ctx context.Context) bool {
	return false
}

func (b *Execution) AppendServerMeta(ctx context.Context, meta *xatu.ServerMeta) *xatu.ServerMeta {
	// Extract ENR from the execution event data.
	executionData := b.event.GetNodeRecordExecution()
	if executionData == nil {
		b.log.Error("failed to get execution data from event")

		return meta
	}

	// Get ENR string from the proto data.
	enrString := ""
	if executionData.GetEnr() != nil {
		enrString = executionData.GetEnr().GetValue()
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

	// Try to get IP + ports from ENR (IPv4 first, then IPv6).
	var ipString string

	var tcpPort uint32

	var udpPort uint32

	if parsedENR.IP4 != nil {
		ipString = *parsedENR.IP4
		tcpPort = *parsedENR.TCP4
		udpPort = *parsedENR.UDP4
	} else if parsedENR.IP6 != nil {
		ipString = *parsedENR.IP6
		tcpPort = *parsedENR.TCP6
		udpPort = *parsedENR.UDP6
	}

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

	// Populate IP + ports field in execution event data.
	executionData.Ip = wrapperspb.String(ipString)
	executionData.Tcp = wrapperspb.UInt32(tcpPort)
	executionData.Udp = wrapperspb.UInt32(udpPort)

	// Populate node ID field in execution event data.
	executionData.NodeId = wrapperspb.String(*parsedENR.NodeID)

	// Perform GeoIP lookup if provider is available.
	if b.geoipProvider != nil {
		geoipResult, err := b.geoipProvider.LookupIP(ctx, ip)
		if err != nil {
			b.log.WithField("ip", ipString).WithError(err).Warn("failed to lookup geoip data")

			return meta
		}

		if geoipResult != nil {
			meta.AdditionalData = &xatu.ServerMeta_NODE_RECORD_EXECUTION{
				NODE_RECORD_EXECUTION: &xatu.ServerMeta_AdditionalNodeRecordExecutionData{
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
			}).Debug("successfully updated server meta with execution node geo information")
		}
	}

	return meta
}

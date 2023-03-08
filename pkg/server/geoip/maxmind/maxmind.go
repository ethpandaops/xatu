package maxmind

import (
	"context"
	"net"

	"github.com/ethpandaops/xatu/pkg/server/geoip/lookup"
	"github.com/ethpandaops/xatu/pkg/server/geoip/maxmind/database"
	"github.com/savid/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

const Type = "maxmind"

type Maxmind struct {
	config *Config

	log logrus.FieldLogger

	client *ttlcache.Cache[string, string]

	city *database.City
	asn  *database.ASN

	metrics *Metrics
}

func New(config *Config, log logrus.FieldLogger) (*Maxmind, error) {
	nLog := log.WithField("geoip/provider", Type)

	return &Maxmind{
		config:  config,
		log:     nLog,
		client:  ttlcache.New[string, string](),
		city:    database.NewCity(config.Database, nLog),
		asn:     database.NewASN(config.Database, nLog),
		metrics: NewMetrics("xatu_server_geoip_provider"),
	}, nil
}

func (m *Maxmind) Type() string {
	return Type
}

func (m *Maxmind) Start(ctx context.Context) error {
	if err := m.city.Start(); err != nil {
		return err
	}

	if err := m.asn.Start(); err != nil {
		return err
	}

	return nil
}

func (m *Maxmind) Stop(ctx context.Context) error {
	if m.city != nil {
		if err := m.city.Stop(); err != nil {
			return err
		}
	}

	if m.asn != nil {
		if err := m.asn.Stop(); err != nil {
			return err
		}
	}

	return nil
}

func (m *Maxmind) LookupIP(ctx context.Context, ip net.IP) (*lookup.Result, error) {
	result := &lookup.Result{}

	if m.city != nil {
		city, err := m.city.Lookup(ip)
		if err != nil {
			m.metrics.AddLookupIP(1, m.Type(), "error")

			return nil, err
		}

		if city != nil {
			result.CityName = city.City.Names.EN
			result.CountryName = city.Country.Names.EN
			result.CountryCode = city.Country.ISOCode
			result.ContinentCode = city.Continent.Code
			result.Latitude = city.Location.Latitude
			result.Longitude = city.Location.Longitude
		}
	}

	if m.asn != nil {
		asn, err := m.asn.Lookup(ip)
		if err != nil {
			m.metrics.AddLookupIP(1, m.Type(), "error")

			return nil, err
		}

		if asn != nil {
			result.AutonomousSystemNumber = asn.AutonomousSystemNumber
			result.AutonomousSystemOrganization = asn.AutonomousSystemOrganization
		}
	}

	m.metrics.AddLookupIP(1, m.Type(), "ok")

	return result, nil
}

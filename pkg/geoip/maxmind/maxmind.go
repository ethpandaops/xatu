package maxmind

import (
	"context"
	"fmt"
	"net"

	"github.com/ethpandaops/xatu/pkg/geoip/lookup"
	"github.com/ethpandaops/xatu/pkg/geoip/maxmind/database"
	"github.com/ethpandaops/xatu/pkg/geoip/maxmind/geonames"
	"github.com/jellydator/ttlcache/v3"
	"github.com/sirupsen/logrus"
)

const Type = "maxmind"

type Maxmind struct {
	config *Config

	log logrus.FieldLogger

	client *ttlcache.Cache[string, string]

	city *database.City
	asn  *database.ASN

	geonameCoords   map[uint]geonames.Coordinates   // City centroids by GeoName ID
	countryCoords   map[uint]geonames.Coordinates   // Country centroids by GeoName ID
	continentCoords map[string]geonames.Coordinates // Continent centroids by code

	metrics *Metrics
}

func New(config *Config, log logrus.FieldLogger) (*Maxmind, error) {
	nLog := log.WithField("geoip/provider", Type)

	// Load GeoNames city data if configured
	var geonameCoords map[uint]geonames.Coordinates
	if config.GeoNames != nil && config.GeoNames.Cities != "" {
		coords, err := geonames.LoadCitiesFromFile(config.GeoNames.Cities)
		if err != nil {
			return nil, fmt.Errorf("failed to load GeoNames cities file: %w", err)
		}

		nLog.WithFields(logrus.Fields{
			"file":   config.GeoNames.Cities,
			"cities": len(coords),
		}).Info("Loaded GeoNames city centroids")
		geonameCoords = coords
	}

	// Load GeoNames country data if configured
	var countryCoords map[uint]geonames.Coordinates
	if config.GeoNames != nil && config.GeoNames.Countries != "" {
		coords, err := geonames.LoadCountriesFromFile(config.GeoNames.Countries)
		if err != nil {
			return nil, fmt.Errorf("failed to load GeoNames countries file: %w", err)
		}

		nLog.WithFields(logrus.Fields{
			"file":      config.GeoNames.Countries,
			"countries": len(coords),
		}).Info("Loaded GeoNames country centroids")
		countryCoords = coords
	}

	// Load hardcoded continent centroids
	continentCoords := geonames.GetContinentCentroids()
	nLog.WithField("continents", len(continentCoords)).Info("Loaded continent centroids")

	return &Maxmind{
		config:          config,
		log:             nLog,
		client:          ttlcache.New[string, string](),
		city:            database.NewCity(config.Database, nLog),
		asn:             database.NewASN(config.Database, nLog),
		geonameCoords:   geonameCoords,
		countryCoords:   countryCoords,
		continentCoords: continentCoords,
		metrics:         NewMetrics("xatu_geoip_provider"),
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

func (m *Maxmind) LookupIP(ctx context.Context, ip net.IP, precision lookup.Precision) (*lookup.Result, error) {
	// Early return for PrecisionNone - skip all geo lookups
	if precision == lookup.PrecisionNone {
		m.metrics.AddLookupIP(1, m.Type(), "none")

		return &lookup.Result{}, nil
	}

	result := &lookup.Result{}

	if m.city != nil {
		city, err := m.city.Lookup(ip)
		if err != nil {
			m.metrics.AddLookupIP(1, m.Type(), "error")

			return nil, err
		}

		if city != nil {
			// Always populate non-coordinate fields initially
			result.CityName = city.City.Names.EN
			result.CountryName = city.Country.Names.EN
			result.CountryCode = city.Country.ISOCode
			result.ContinentCode = city.Continent.Code

			// Adjust fields and coordinates based on precision level
			switch precision {
			case lookup.PrecisionFull:
				// Full precision - always use original MaxMind coordinates
				result.Latitude = city.Location.Latitude
				result.Longitude = city.Location.Longitude

			case lookup.PrecisionCity:
				// City precision - use city centroid only if GeoNames available and found
				if m.geonameCoords != nil && city.City.GeoNameID > 0 {
					if coords, ok := m.geonameCoords[city.City.GeoNameID]; ok {
						result.Latitude = coords.Latitude
						result.Longitude = coords.Longitude
					}
					// No else - leave coordinates as zero if not found
				}
				// No coordinates set if GeoNames not configured or city not found

			case lookup.PrecisionCountry:
				// Country precision - clear city, use country centroid only if available
				result.CityName = ""

				if m.countryCoords != nil && city.Country.GeoNameID > 0 {
					if coords, ok := m.countryCoords[city.Country.GeoNameID]; ok {
						result.Latitude = coords.Latitude
						result.Longitude = coords.Longitude
					}
					// No else - leave coordinates as zero if not found
				}
				// No coordinates set if country centroids not configured or country not found

			case lookup.PrecisionContinent:
				// Continent precision - clear city/country, use continent centroid only if available
				result.CityName = ""
				result.CountryName = ""
				result.CountryCode = ""

				if m.continentCoords != nil && city.Continent.Code != "" {
					if coords, ok := m.continentCoords[city.Continent.Code]; ok {
						result.Latitude = coords.Latitude
						result.Longitude = coords.Longitude
					}
					// No else - leave coordinates as zero if not found
				}
				// No coordinates set if continent not found
			}
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

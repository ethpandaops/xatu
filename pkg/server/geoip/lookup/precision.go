package lookup

import (
	"fmt"
	"strings"
)

// Precision defines the level of geographic precision for lookups
type Precision int

const (
	// PrecisionFull includes city, country, continent and exact coordinates
	PrecisionFull Precision = iota
	// PrecisionCity includes city, country, continent with city centroid coordinates
	PrecisionCity
	// PrecisionCountry includes country, continent with country centroid coordinates
	PrecisionCountry
	// PrecisionContinent includes only continent with continent centroid coordinates
	PrecisionContinent
	// PrecisionNone skips geo lookup entirely, returns no geo data
	PrecisionNone
)

// String returns the string representation of the precision level
func (p Precision) String() string {
	switch p {
	case PrecisionFull:
		return "full"
	case PrecisionCity:
		return "city"
	case PrecisionCountry:
		return "country"
	case PrecisionContinent:
		return "continent"
	case PrecisionNone:
		return "none"
	default:
		return "unknown"
	}
}

// UnmarshalYAML implements yaml.Unmarshaler interface for YAML parsing
func (p *Precision) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	switch strings.ToLower(s) {
	case "full", "":
		*p = PrecisionFull
	case "city":
		*p = PrecisionCity
	case "country":
		*p = PrecisionCountry
	case "continent":
		*p = PrecisionContinent
	case "none":
		*p = PrecisionNone
	default:
		return fmt.Errorf("invalid precision value: %s", s)
	}

	return nil
}

// MarshalYAML implements yaml.Marshaler interface for YAML output
func (p Precision) MarshalYAML() (any, error) {
	return p.String(), nil
}

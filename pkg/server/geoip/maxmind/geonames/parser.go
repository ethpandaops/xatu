package geonames

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Coordinates represents a geographic coordinate pair
type Coordinates struct {
	Latitude  float64
	Longitude float64
}

// LoadCitiesFromFile loads GeoNames geoname_id to coordinates mapping from a local file
// The file should be in GeoNames tab-delimited format (e.g., cities1000.txt)
func LoadCitiesFromFile(filepath string) (map[uint]Coordinates, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoNames file: %w", err)
	}
	defer file.Close()

	coords := make(map[uint]Coordinates)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		fields := strings.Split(scanner.Text(), "\t")

		// Minimum required fields: geoname_id (0), latitude (4), longitude (5)
		if len(fields) < 6 {
			continue
		}

		// Parse geoname_id (field 0)
		geonameID, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil {
			continue
		}

		// Parse latitude (field 4)
		lat, err := strconv.ParseFloat(fields[4], 64)
		if err != nil {
			continue
		}

		// Parse longitude (field 5)
		lon, err := strconv.ParseFloat(fields[5], 64)
		if err != nil {
			continue
		}

		coords[uint(geonameID)] = Coordinates{
			Latitude:  lat,
			Longitude: lon,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading GeoNames file: %w", err)
	}

	return coords, nil
}

// LoadCountriesFromFile loads GeoNames country geoname_id to coordinates mapping from countries.txt
// The file should be tab-delimited with format: geoname_id<tab>latitude<tab>longitude
func LoadCountriesFromFile(filepath string) (map[uint]Coordinates, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GeoNames countries file: %w", err)
	}
	defer file.Close()

	coords := make(map[uint]Coordinates)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and empty lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Split(line, "\t")

		// Expected format: geoname_id<tab>latitude<tab>longitude
		if len(fields) < 3 {
			continue
		}

		// Parse geoname_id (field 0)
		geonameID, err := strconv.ParseUint(fields[0], 10, 32)
		if err != nil {
			continue
		}

		// Parse latitude (field 1)
		lat, err := strconv.ParseFloat(fields[1], 64)
		if err != nil {
			continue
		}

		// Parse longitude (field 2)
		lon, err := strconv.ParseFloat(fields[2], 64)
		if err != nil {
			continue
		}

		coords[uint(geonameID)] = Coordinates{
			Latitude:  lat,
			Longitude: lon,
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading GeoNames countries file: %w", err)
	}

	return coords, nil
}

// GetContinentCentroids returns hardcoded continent centroids
// These are approximate geographic centers of continents
func GetContinentCentroids() map[string]Coordinates {
	return map[string]Coordinates{
		"AF": {Latitude: -8.7832, Longitude: 34.5085},   // Africa
		"AN": {Latitude: -82.8628, Longitude: 135.0000}, // Antarctica
		"AS": {Latitude: 34.0479, Longitude: 100.6197},  // Asia
		"EU": {Latitude: 54.5260, Longitude: 15.2551},   // Europe
		"NA": {Latitude: 54.5260, Longitude: -105.2551}, // North America
		"OC": {Latitude: -22.7359, Longitude: 140.0188}, // Oceania
		"SA": {Latitude: -8.7832, Longitude: -55.4915},  // South America
	}
}

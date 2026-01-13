package geonames

// Config defines the configuration for GeoNames data files
type Config struct {
	Cities    string `yaml:"cities"`    // Optional: GeoNames cities file for city centroids
	Countries string `yaml:"countries"` // Optional: GeoNames countries file for country centroids
}

// IsConfigured returns true if any GeoNames data is configured
func (c *Config) IsConfigured() bool {
	return c != nil && (c.Cities != "" || c.Countries != "")
}

package database

import "errors"

type Config struct {
	City string `yaml:"city"`
	ASN  string `yaml:"asn"`
}

func (c *Config) Validate() error {
	if c.City == "" {
		return errors.New("city is required")
	}

	if c.ASN == "" {
		return errors.New("asn is required")
	}

	return nil
}

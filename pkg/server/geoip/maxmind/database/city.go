package database

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
	"github.com/sirupsen/logrus"
)

type City struct {
	config *Config

	log logrus.FieldLogger

	db *maxminddb.Reader
}

type LookupCity struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
		Names   struct {
			EN string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"country"`
	Continent struct {
		Code string `maxminddb:"code"`
	} `maxminddb:"continent"`
	City struct {
		Names struct {
			EN string `maxminddb:"en"`
		} `maxminddb:"names"`
	} `maxminddb:"city"`
	Location struct {
		Latitude  float64 `maxminddb:"latitude"`
		Longitude float64 `maxminddb:"longitude"`
	} `maxminddb:"location"`
}

func NewCity(config *Config, log logrus.FieldLogger) *City {
	return &City{
		config: config,
		log:    log.WithField("database", "city"),
	}
}

func (c *City) Start() error {
	db, err := maxminddb.Open(c.config.City)
	if err != nil {
		return err
	}

	c.db = db

	return nil
}

func (c *City) Stop() error {
	if c.db != nil {
		c.db.Close()
	}

	return nil
}

func (c *City) Lookup(ip net.IP) (*LookupCity, error) {
	var lookup LookupCity

	err := c.db.Lookup(ip, &lookup)
	if err != nil {
		return nil, err
	}

	return &lookup, nil
}

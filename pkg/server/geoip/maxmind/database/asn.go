package database

import (
	"net"

	"github.com/oschwald/maxminddb-golang"
	"github.com/sirupsen/logrus"
)

type ASN struct {
	config *Config

	log logrus.FieldLogger

	db *maxminddb.Reader
}

type LookupASN struct {
	AutonomousSystemNumber       uint32 `maxminddb:"autonomous_system_number"`
	AutonomousSystemOrganization string `maxminddb:"autonomous_system_organization"`
}

func NewASN(config *Config, log logrus.FieldLogger) *ASN {
	return &ASN{
		config: config,
		log:    log.WithField("database", "city"),
	}
}

func (c *ASN) Start() error {
	db, err := maxminddb.Open(c.config.ASN)
	if err != nil {
		return err
	}

	c.db = db

	return nil
}

func (c *ASN) Stop() error {
	if c.db != nil {
		c.db.Close()
	}

	return nil
}

func (c *ASN) Lookup(ip net.IP) (*LookupASN, error) {
	var lookup LookupASN

	err := c.db.Lookup(ip, &lookup)
	if err != nil {
		return nil, err
	}

	return &lookup, nil
}

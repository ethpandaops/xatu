package persistence

import (
	"context"
	"database/sql"

	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Client struct {
	log    logrus.FieldLogger
	config *Config

	db *sql.DB
}

func NewClient(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Client, error) {
	c := &Client{
		log:    log,
		config: conf,
	}

	return c, nil
}

func (c *Client) Start(ctx context.Context) error {
	db, err := sql.Open(string(c.config.DriverName), c.config.ConnectionString)
	if err != nil {
		return err
	}

	db.SetMaxIdleConns(c.config.MaxIdleConns)
	db.SetMaxOpenConns(c.config.MaxOpenConns)

	c.db = db

	return nil
}

func (c *Client) Stop(ctx context.Context) error {
	if c.db == nil {
		return nil
	}

	return c.db.Close()
}

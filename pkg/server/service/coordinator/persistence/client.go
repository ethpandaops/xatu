package persistence

import (
	"context"
	"database/sql"

	//nolint:blank-imports // Required for postgres driver
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

type Client struct {
	log    logrus.FieldLogger
	config *Config

	db *sql.DB
}

func New(ctx context.Context, log logrus.FieldLogger, conf *Config) (*Client, error) {
	c := &Client{
		log:    log,
		config: conf,
	}

	return c, nil
}

func (e *Client) Start(ctx context.Context) error {
	db, err := sql.Open(string(e.config.DriverName), e.config.ConnectionString)
	if err != nil {
		return err
	}

	e.db = db

	return nil
}

func (e *Client) Stop(ctx context.Context) error {
	if e.db == nil {
		return nil
	}

	return e.db.Close()
}

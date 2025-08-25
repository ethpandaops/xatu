package persistence

import (
	"context"
	"errors"
	"time"

	perrors "github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/server/persistence/relaymonitor"
	"github.com/huandu/go-sqlbuilder"
)

var relayMonitorLocationStruct = sqlbuilder.NewStruct(new(relaymonitor.Location)).For(sqlbuilder.PostgreSQL)

var ErrRelayMonitorLocationNotFound = errors.New("relay monitor location not found")

func (c *Client) UpsertRelayMonitorLocation(ctx context.Context, location *relaymonitor.Location) error {
	if location.LocationID == nil {
		location.LocationID = sqlbuilder.Raw("DEFAULT")
	}

	location.CreateTime = time.Now()
	location.UpdateTime = time.Now()

	ub := relayMonitorLocationStruct.InsertInto("relay_monitor_location", location)

	sqlQuery, args := ub.Build()
	sqlQuery += " ON CONFLICT ON CONSTRAINT relay_monitor_location_unique DO UPDATE SET update_time = EXCLUDED.update_time, value = EXCLUDED.value"

	c.log.WithField("sql", sqlQuery).WithField("args", args).Debug("UpsertRelayMonitorLocation")

	_, err := c.db.ExecContext(ctx, sqlQuery, args...)

	return err
}

func (c *Client) GetRelayMonitorLocationByID(ctx context.Context, id int64) (*relaymonitor.Location, error) {
	sb := relayMonitorLocationStruct.SelectFrom("relay_monitor_location")
	sb.Where(sb.E("location_id", id))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*relaymonitor.Location

	for rows.Next() {
		var location relaymonitor.Location

		err = rows.Scan(relayMonitorLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrRelayMonitorLocationNotFound
	}

	return locations[0], nil
}

// GetRelayMonitorLocationByNetworkIDTypeAndRelay gets location by network name, client name, type, and relay name
func (c *Client) GetRelayMonitorLocationByNetworkIDTypeAndRelay(ctx context.Context, networkName, clientName, typ, relayName string) (*relaymonitor.Location, error) {
	sb := relayMonitorLocationStruct.SelectFrom("relay_monitor_location")
	sb.Where(sb.E("meta_network_name", networkName))
	sb.Where(sb.E("meta_client_name", clientName))
	sb.Where(sb.E("type", typ))
	sb.Where(sb.E("relay_name", relayName))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*relaymonitor.Location

	for rows.Next() {
		var location relaymonitor.Location

		err = rows.Scan(relayMonitorLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrRelayMonitorLocationNotFound
	}

	return locations[0], nil
}

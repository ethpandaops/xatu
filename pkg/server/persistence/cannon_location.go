package persistence

import (
	"context"
	"errors"
	"time"

	perrors "github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/server/persistence/cannon"
	"github.com/huandu/go-sqlbuilder"
)

var cannonLocationStruct = sqlbuilder.NewStruct(new(cannon.Location)).For(sqlbuilder.PostgreSQL)

var ErrCannonLocationNotFound = errors.New("cannon location not found")

func (c *Client) UpsertCannonLocation(ctx context.Context, location *cannon.Location) error {
	if location.LocationID == nil {
		location.LocationID = sqlbuilder.Raw("DEFAULT")
	}

	location.CreateTime = time.Now()
	location.UpdateTime = time.Now()

	ub := cannonLocationStruct.InsertInto("cannon_location", location)

	sqlQuery, args := ub.Build()
	sqlQuery += " ON CONFLICT ON CONSTRAINT cannon_location_unique DO UPDATE SET update_time = EXCLUDED.update_time, value = EXCLUDED.value"

	c.log.WithField("sql", sqlQuery).WithField("args", args).Debug("UpsertCannonLocation")

	_, err := c.db.ExecContext(ctx, sqlQuery, args...)

	return err
}

func (c *Client) GetCannonLocationByID(ctx context.Context, id int64) (*cannon.Location, error) {
	sb := cannonLocationStruct.SelectFrom("cannon_location")
	sb.Where(sb.E("location_id", id))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*cannon.Location

	for rows.Next() {
		var location cannon.Location

		err = rows.Scan(cannonLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrCannonLocationNotFound
	}

	return locations[0], nil
}

// get by network id and type
func (c *Client) GetCannonLocationByNetworkIDAndType(ctx context.Context, networkID, typ string) (*cannon.Location, error) {
	sb := cannonLocationStruct.SelectFrom("cannon_location")
	sb.Where(sb.E("network_id", networkID))
	sb.Where(sb.E("type", typ))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*cannon.Location

	for rows.Next() {
		var location cannon.Location

		err = rows.Scan(cannonLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrCannonLocationNotFound
	}

	return locations[0], nil
}

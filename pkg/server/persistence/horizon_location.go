package persistence

import (
	"context"
	"errors"
	"time"

	perrors "github.com/pkg/errors"

	"github.com/ethpandaops/xatu/pkg/server/persistence/horizon"
	"github.com/huandu/go-sqlbuilder"
)

var horizonLocationStruct = sqlbuilder.NewStruct(new(horizon.Location)).For(sqlbuilder.PostgreSQL)

var ErrHorizonLocationNotFound = errors.New("horizon location not found")

func (c *Client) UpsertHorizonLocation(ctx context.Context, location *horizon.Location) error {
	if location.LocationID == nil {
		location.LocationID = sqlbuilder.Raw("DEFAULT")
	}

	location.CreateTime = time.Now()
	location.UpdateTime = time.Now()

	ub := horizonLocationStruct.InsertInto("horizon_location", location)

	sqlQuery, args := ub.Build()
	sqlQuery += " ON CONFLICT ON CONSTRAINT horizon_location_unique DO UPDATE SET update_time = EXCLUDED.update_time, head_slot = EXCLUDED.head_slot, fill_slot = EXCLUDED.fill_slot"

	c.log.WithField("sql", sqlQuery).WithField("args", args).Debug("UpsertHorizonLocation")

	_, err := c.db.ExecContext(ctx, sqlQuery, args...)

	return err
}

func (c *Client) GetHorizonLocationByID(ctx context.Context, id int64) (*horizon.Location, error) {
	sb := horizonLocationStruct.SelectFrom("horizon_location")
	sb.Where(sb.E("location_id", id))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*horizon.Location

	for rows.Next() {
		var location horizon.Location

		err = rows.Scan(horizonLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrHorizonLocationNotFound
	}

	return locations[0], nil
}

// GetHorizonLocationByNetworkIDAndType gets location by network id and type.
func (c *Client) GetHorizonLocationByNetworkIDAndType(ctx context.Context, networkID, typ string) (*horizon.Location, error) {
	sb := horizonLocationStruct.SelectFrom("horizon_location")
	sb.Where(sb.E("network_id", networkID))
	sb.Where(sb.E("type", typ))

	sql, args := sb.Build()

	rows, err := c.db.QueryContext(ctx, sql, args...)
	if err != nil {
		return nil, perrors.Wrap(err, "db query failed")
	}

	defer rows.Close()

	var locations []*horizon.Location

	for rows.Next() {
		var location horizon.Location

		err = rows.Scan(horizonLocationStruct.Addr(&location)...)
		if err != nil {
			return nil, perrors.Wrap(err, "db scan failed")
		}

		locations = append(locations, &location)
	}

	if len(locations) != 1 {
		return nil, ErrHorizonLocationNotFound
	}

	return locations[0], nil
}

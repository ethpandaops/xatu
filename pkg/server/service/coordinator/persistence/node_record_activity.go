package persistence

import (
	"context"
	"time"

	"github.com/ethpandaops/xatu/pkg/server/service/coordinator/node"
	"github.com/huandu/go-sqlbuilder"
)

var nodeRecordActivityStruct = sqlbuilder.NewStruct(new(node.Activity)).For(sqlbuilder.PostgreSQL)

type AvailableExecutionNodeRecord struct {
	Enr             string    `db:"enr"`
	ExecutionCount  int64     `db:"execution_count"`
	LastConnectTime time.Time `db:"last_connect_time"`
}

var availableExecutionNodeRecordStruct = sqlbuilder.NewStruct(new(AvailableExecutionNodeRecord)).For(sqlbuilder.PostgreSQL)

func (e *Client) UpsertNodeRecordActivities(ctx context.Context, activities []*node.Activity) error {
	values := make([]interface{}, len(activities))

	for i, activity := range activities {
		values[i] = &node.Activity{
			ActivityID: sqlbuilder.Raw("DEFAULT"),
			Enr:        activity.Enr,
			ClientID:   activity.ClientID,
			CreateTime: time.Now(),
			UpdateTime: time.Now(),
			Connected:  activity.Connected,
		}
	}

	ub := nodeRecordActivityStruct.InsertInto("node_record_activity", values...)

	sqlQuery, args := ub.Build()
	sqlQuery += " ON CONFLICT ON CONSTRAINT c_unique DO UPDATE SET update_time = EXCLUDED.update_time, connected = EXCLUDED.connected"

	_, err := e.db.Exec(sqlQuery, args...)

	return err
}

func (e *Client) ListAvailableExecutionNodeRecords(ctx context.Context, clientID string, ignoredNodeRecords []string, networkIDs []uint64, forkIDHashes [][]byte, limit int) ([]*string, error) {
	inr := make([]interface{}, 0, len(ignoredNodeRecords))
	for _, enr := range ignoredNodeRecords {
		inr = append(inr, enr)
	}

	nids := make([]interface{}, 0, len(networkIDs))
	for _, nid := range networkIDs {
		nids = append(nids, nid)
	}

	fidhs := make([]interface{}, 0, len(forkIDHashes))
	for _, fidh := range forkIDHashes {
		fidhs = append(fidhs, fidh)
	}

	sbsub := sqlbuilder.PostgreSQL.NewSelectBuilder()
	sbsub.Select(
		"enr",
		"COUNT(*) as active_clients",
		"SUM(connected::int) as connected_clients",
	)
	sbsub.From("node_record_activity")
	sbsub.Where(
		sbsub.GreaterThan("update_time", sqlbuilder.Raw("now() - interval '1 hour'")),
		sbsub.NotEqual("client_id", clientID),
	)
	sbsub.GroupBy("enr")

	subQuery, subArgs := sbsub.Build()

	sb := sqlbuilder.PostgreSQL.NewSelectBuilder()
	sb.Select(
		"nre.enr as enr",
		"count(*) as execution_count",
		"max(nre.create_time) as last_connect_time",
	)
	sb.From("node_record_execution as nre")
	sb.JoinWithOption(
		"LEFT",
		"("+subQuery+") as nra",
		"nra.enr = nre.enr",
	)

	where := []string{
		sb.GreaterThan("nre.create_time", sqlbuilder.Raw("now() - interval '1 week'")),
		sb.Or(
			sb.LessThan("nra.active_clients", 2),
			sb.IsNull("nra.active_clients"),
		),
	}

	if len(inr) > 0 {
		where = append(where, sb.NotIn("nre.enr", inr...))
	}

	if len(nids) > 0 {
		where = append(where, sb.In("nre.network_id", nids...))
	}

	if len(fidhs) > 0 {
		where = append(where, sb.In("nre.fork_id_hash", fidhs...))
	}

	sb.Where(where...)
	sb.GroupBy("nre.enr")
	sb.OrderBy("last_connect_time ASC")
	sb.Limit(limit)

	sqlQuery, args := sb.Build()

	args[0] = subArgs[0]

	rows, err := e.db.Query(sqlQuery, args...)
	if err != nil {
		return nil, err
	}

	nodeRecords := make([]*string, 0, limit)

	for rows.Next() {
		var record AvailableExecutionNodeRecord

		err = rows.Scan(availableExecutionNodeRecordStruct.Addr(&record)...)
		if err != nil {
			return nil, err
		}

		nodeRecords = append(nodeRecords, &record.Enr)
	}

	return nodeRecords, nil
}

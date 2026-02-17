package flattener

// TableName is a typed ClickHouse target table identifier for flatteners.
type TableName string

func (t TableName) String() string {
	return string(t)
}

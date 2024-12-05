package processor

const (
	DefaultMaxQueueSize       = 51200
	DefaultScheduleDelay      = 5000
	DefaultExportTimeout      = 30000
	DefaultMaxExportBatchSize = 512
	DefaultShippingMethod     = ShippingMethodAsync
	DefaultNumWorkers         = 5
)

// ShippingMethod is the method of shipping items for export.
type ShippingMethod string

const (
	ShippingMethodUnknown ShippingMethod = "unknown"
	ShippingMethodAsync   ShippingMethod = "async"
	ShippingMethodSync    ShippingMethod = "sync"
)

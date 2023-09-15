package options

type ShippingMethod string

const (
	ShippingMethodUnknown ShippingMethod = "unknown"
	ShippingMethodAsync   ShippingMethod = "async"
	ShippingMethodSync    ShippingMethod = "sync"
)

type Options struct {
	ShippingMethod ShippingMethod
}

func DefaultOptions() *Options {
	return &Options{
		ShippingMethod: ShippingMethodAsync,
	}
}

func (o *Options) SetShippingMethod(method ShippingMethod) *Options {
	o.ShippingMethod = method

	return o
}

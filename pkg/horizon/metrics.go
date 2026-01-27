package horizon

import (
	"github.com/ethpandaops/xatu/pkg/proto/xatu"
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	decoratedEventTotal *prometheus.CounterVec

	// Slot tracking gauges
	headSlot *prometheus.GaugeVec
	fillSlot *prometheus.GaugeVec
	lagSlots *prometheus.GaugeVec

	// Block processing counter
	blocksDerivedTotal *prometheus.CounterVec
}

func NewMetrics(namespace string) *Metrics {
	m := &Metrics{
		decoratedEventTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "decorated_event_total",
			Help:      "Total number of decorated events created by horizon",
		}, []string{"type", "network"}),

		headSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "head_slot",
			Help:      "Current HEAD slot position being processed by horizon",
		}, []string{"deriver", "network"}),

		fillSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "fill_slot",
			Help:      "Current FILL slot position for catch-up processing",
		}, []string{"deriver", "network"}),

		lagSlots: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag_slots",
			Help:      "Number of slots FILL is behind HEAD",
		}, []string{"deriver", "network"}),

		blocksDerivedTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "blocks_derived_total",
			Help:      "Total number of blocks derived by horizon",
		}, []string{"deriver", "network", "iterator"}),
	}

	prometheus.MustRegister(
		m.decoratedEventTotal,
		m.headSlot,
		m.fillSlot,
		m.lagSlots,
		m.blocksDerivedTotal,
	)

	return m
}

func (m *Metrics) AddDecoratedEvent(count int, eventType *xatu.DecoratedEvent, network string) {
	m.decoratedEventTotal.WithLabelValues(eventType.Event.Name.String(), network).Add(float64(count))
}

// SetHeadSlot sets the current HEAD slot position for a deriver.
func (m *Metrics) SetHeadSlot(slot uint64, deriver, network string) {
	m.headSlot.WithLabelValues(deriver, network).Set(float64(slot))
}

// SetFillSlot sets the current FILL slot position for a deriver.
func (m *Metrics) SetFillSlot(slot uint64, deriver, network string) {
	m.fillSlot.WithLabelValues(deriver, network).Set(float64(slot))
}

// SetLagSlots sets the number of slots FILL is behind HEAD for a deriver.
func (m *Metrics) SetLagSlots(lag uint64, deriver, network string) {
	m.lagSlots.WithLabelValues(deriver, network).Set(float64(lag))
}

// AddBlocksDerived increments the count of blocks derived.
func (m *Metrics) AddBlocksDerived(count int, deriver, network, iterator string) {
	m.blocksDerivedTotal.WithLabelValues(deriver, network, iterator).Add(float64(count))
}

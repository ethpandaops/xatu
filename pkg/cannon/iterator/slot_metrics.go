package iterator

import "github.com/prometheus/client_golang/prometheus"

type SlotMetrics struct {
	TrailingSlots *prometheus.GaugeVec
}

func NewSlotMetrics(namespace string) (SlotMetrics, error) {
	s := SlotMetrics{
		TrailingSlots: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "slot_iterator_trailing_slots",
			Help:      "The number of slots that the iterator is trailing behind the head of the chain",
		}, []string{"cannon_type", "network"}),
	}

	if err := prometheus.Register(s.TrailingSlots); err != nil && err.Error() != "duplicate metrics collector registration attempted" {
		return s, err
	}

	return s, nil
}

func (s *SlotMetrics) SetTrailingSlots(cannonType, network string, trailingSlots float64) {
	s.TrailingSlots.WithLabelValues(cannonType, network).Set(trailingSlots)
}

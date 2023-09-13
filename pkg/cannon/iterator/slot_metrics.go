package iterator

import "github.com/prometheus/client_golang/prometheus"

type SlotMetrics struct {
	TrailingSlots *prometheus.GaugeVec
	CurrentSlot   *prometheus.GaugeVec
}

func NewSlotMetrics(namespace string) SlotMetrics {
	namespace += "_slot_iterator"

	s := SlotMetrics{
		TrailingSlots: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "trailing_slots",
			Help:      "The number of slots that the iterator is trailing behind the head of the chain",
		}, []string{"cannon_type", "network"}),
		CurrentSlot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_slot",
			Help:      "The current slot",
		}, []string{"cannon_type", "network"}),
	}

	prometheus.MustRegister(s.TrailingSlots)
	prometheus.MustRegister(s.CurrentSlot)

	return s
}

func (s *SlotMetrics) SetTrailingSlots(cannonType, network string, trailingSlots float64) {
	s.TrailingSlots.WithLabelValues(cannonType, network).Set(trailingSlots)
}

func (s *SlotMetrics) SetCurrentSlot(cannonType, network string, currentSlot float64) {
	s.CurrentSlot.WithLabelValues(cannonType, network).Set(currentSlot)
}

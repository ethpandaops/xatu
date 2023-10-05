package iterator

import "github.com/prometheus/client_golang/prometheus"

type BlockprintMetrics struct {
	Targetslot  *prometheus.GaugeVec
	Currentslot *prometheus.GaugeVec
}

func NewBlockprintMetrics(namespace string) BlockprintMetrics {
	namespace += "_slot_iterator"

	s := BlockprintMetrics{
		Targetslot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "target_slot",
			Help:      "The target slot",
		}, []string{"cannon_type", "network"}),
		Currentslot: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_slot",
			Help:      "The current slot",
		}, []string{"cannon_type", "network"}),
	}

	prometheus.MustRegister(s.Targetslot)
	prometheus.MustRegister(s.Currentslot)

	return s
}

func (s *BlockprintMetrics) SetTargetSlot(cannonType, network string, target float64) {
	s.Targetslot.WithLabelValues(cannonType, network).Set(target)
}

func (s *BlockprintMetrics) SetCurrentSlot(cannonType, network string, current float64) {
	s.Currentslot.WithLabelValues(cannonType, network).Set(current)
}

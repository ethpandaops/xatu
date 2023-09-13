package iterator

import "github.com/prometheus/client_golang/prometheus"

type CheckpointMetrics struct {
	Trailingepochs *prometheus.GaugeVec
	Currentepoch   *prometheus.GaugeVec
}

func NewCheckpointMetrics(namespace string) CheckpointMetrics {
	namespace += "_epoch_iterator"

	s := CheckpointMetrics{
		Trailingepochs: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "trailing_epochs",
			Help:      "The number of epochs that the iterator is trailing behind the latest checkpoint epoch",
		}, []string{"cannon_type", "network", "checkpoint"}),
		Currentepoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "current_epoch",
			Help:      "The current epoch",
		}, []string{"cannon_type", "network", "checkpoint"}),
	}

	prometheus.MustRegister(s.Trailingepochs)
	prometheus.MustRegister(s.Currentepoch)

	return s
}

func (s *CheckpointMetrics) SetTrailingEpochs(cannonType, network, checkpoint string, trailing float64) {
	s.Trailingepochs.WithLabelValues(cannonType, network, checkpoint).Set(trailing)
}

func (s *CheckpointMetrics) SetCurrentEpoch(cannonType, network, checkpoint string, current float64) {
	s.Currentepoch.WithLabelValues(cannonType, network, checkpoint).Set(current)
}

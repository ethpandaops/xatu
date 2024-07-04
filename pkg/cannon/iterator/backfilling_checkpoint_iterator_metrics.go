package iterator

import "github.com/prometheus/client_golang/prometheus"

type BackfillingCheckpointMetrics struct {
	BackfillEpoch            *prometheus.GaugeVec
	FinalizedEpoch           *prometheus.GaugeVec
	FinalizedCheckpointEpoch *prometheus.GaugeVec
}

func NewBackfillingCheckpointMetrics(namespace string) BackfillingCheckpointMetrics {
	namespace += "_epoch_iterator"

	s := BackfillingCheckpointMetrics{
		BackfillEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "backfill_epoch",
			Help:      "The current position of the backfill epoch",
		}, []string{"cannon_type", "network", "checkpoint"}),
		FinalizedEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "finalized_epoch",
			Help:      "The current position of the finalized epoch",
		}, []string{"cannon_type", "network", "checkpoint"}),
		FinalizedCheckpointEpoch: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "finalized_checkpoint_epoch",
			Help:      "The finalized checkpoint epoch of the network",
		}, []string{"network"}),
	}

	prometheus.MustRegister(s.BackfillEpoch)
	prometheus.MustRegister(s.FinalizedEpoch)
	prometheus.MustRegister(s.FinalizedCheckpointEpoch)

	return s
}

func (s *BackfillingCheckpointMetrics) SetBackfillEpoch(cannonType, network, checkpoint string, epoch float64) {
	s.BackfillEpoch.WithLabelValues(cannonType, network, checkpoint).Set(epoch)
}

func (s *BackfillingCheckpointMetrics) SetFinalizedEpoch(cannonType, network, checkpoint string, epoch float64) {
	s.FinalizedEpoch.WithLabelValues(cannonType, network, checkpoint).Set(epoch)
}

func (s *BackfillingCheckpointMetrics) SetFinalizedCheckpointEpoch(network string, epoch float64) {
	s.FinalizedCheckpointEpoch.WithLabelValues(network).Set(epoch)
}

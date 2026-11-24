package iterator

import "github.com/prometheus/client_golang/prometheus"

type BackfillingCheckpointMetrics struct {
	BackfillEpoch            *prometheus.GaugeVec
	FinalizedEpoch           *prometheus.GaugeVec
	FinalizedCheckpointEpoch *prometheus.GaugeVec
	Lag                      *prometheus.GaugeVec
	BackfillBatchFallbacks   *prometheus.CounterVec
}

func NewBackfillingCheckpointMetrics(namespace string) BackfillingCheckpointMetrics {
	s := newBackfillingCheckpointMetrics(namespace)

	prometheus.MustRegister(s.BackfillEpoch)
	prometheus.MustRegister(s.FinalizedEpoch)
	prometheus.MustRegister(s.FinalizedCheckpointEpoch)
	prometheus.MustRegister(s.Lag)
	prometheus.MustRegister(s.BackfillBatchFallbacks)

	return s
}

// newBackfillingCheckpointMetrics builds the collectors without registering them.
func newBackfillingCheckpointMetrics(namespace string) BackfillingCheckpointMetrics {
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
		Lag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag_epochs",
			Help:      "The lag of the iterator in epochs",
		}, []string{"cannon_type", "network", "direction"}),
		BackfillBatchFallbacks: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "backfill_batch_fallbacks_total",
			Help:      "Backfill batches that did not commit and were retried one epoch at a time",
		}, []string{"cannon_type", "network"}),
	}

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

func (s *BackfillingCheckpointMetrics) SetLag(cannonType, network string, direction BackfillingCheckpointDirection, lag float64) {
	s.Lag.WithLabelValues(cannonType, network, string(direction)).Set(lag)
}

func (s *BackfillingCheckpointMetrics) IncBackfillBatchFallbacks(cannonType, network string) {
	s.BackfillBatchFallbacks.WithLabelValues(cannonType, network).Inc()
}

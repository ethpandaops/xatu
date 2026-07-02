package iterator

import "github.com/prometheus/client_golang/prometheus"

const (
	labelCannonType = "cannon_type"
	labelNetwork    = "network"
	labelDirection  = "direction"
)

// BackfillingBlockMetrics exposes the position of an EL cannon block iterator.
type BackfillingBlockMetrics struct {
	BackfillBlock  *prometheus.GaugeVec
	FinalizedBlock *prometheus.GaugeVec
	CeilingBlock   *prometheus.GaugeVec
	Lag            *prometheus.GaugeVec
}

// NewBackfillingBlockMetrics registers and returns the block iterator metrics.
func NewBackfillingBlockMetrics(namespace string) BackfillingBlockMetrics {
	namespace += "_block_iterator"

	s := BackfillingBlockMetrics{
		BackfillBlock: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "backfill_block",
			Help:      "The current position of the backfill block",
		}, []string{labelCannonType, labelNetwork}),
		FinalizedBlock: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "finalized_block",
			Help:      "The current position of the finalized (head) block",
		}, []string{labelCannonType, labelNetwork}),
		CeilingBlock: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "ceiling_block",
			Help:      "The CL-finalized execution block number ceiling",
		}, []string{labelNetwork}),
		Lag: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Namespace: namespace,
			Name:      "lag_blocks",
			Help:      "The lag of the iterator in blocks",
		}, []string{labelCannonType, labelNetwork, labelDirection}),
	}

	prometheus.MustRegister(s.BackfillBlock)
	prometheus.MustRegister(s.FinalizedBlock)
	prometheus.MustRegister(s.CeilingBlock)
	prometheus.MustRegister(s.Lag)

	return s
}

// SetBackfillBlock records the backfill position.
func (s *BackfillingBlockMetrics) SetBackfillBlock(cannonType, network string, block float64) {
	s.BackfillBlock.WithLabelValues(cannonType, network).Set(block)
}

// SetFinalizedBlock records the head position.
func (s *BackfillingBlockMetrics) SetFinalizedBlock(cannonType, network string, block float64) {
	s.FinalizedBlock.WithLabelValues(cannonType, network).Set(block)
}

// SetCeilingBlock records the CL-finalized execution block ceiling.
func (s *BackfillingBlockMetrics) SetCeilingBlock(network string, block float64) {
	s.CeilingBlock.WithLabelValues(network).Set(block)
}

// SetLag records the iterator lag in blocks for a direction.
func (s *BackfillingBlockMetrics) SetLag(cannonType, network string, direction BackfillingBlockDirection, lag float64) {
	s.Lag.WithLabelValues(cannonType, network, string(direction)).Set(lag)
}

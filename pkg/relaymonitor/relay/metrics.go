package relay

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	apiRequestsTotal *prometheus.CounterVec
	bidsReceived     *prometheus.CounterVec
	apiFailuresTotal *prometheus.CounterVec
}

var (
	GetBidsEndpoint = "get_bids"
	instance        *Metrics
	once            sync.Once
)

func GetMetrics(namespace string) *Metrics {
	once.Do(func() {
		instance = newMetrics(namespace)
	})

	return instance
}

func newMetrics(namespace string) *Metrics {
	m := &Metrics{
		apiRequestsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_requests_total",
			Help:      "Total number of API requests made to the relay",
		}, []string{"relay", "endpoint", "network"}),
		bidsReceived: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "bids_received_total",
			Help:      "Total number of bids received from the relay",
		}, []string{"relay", "network"}),
		apiFailuresTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "api_failures_total",
			Help:      "Total number of API failures",
		}, []string{"relay", "endpoint", "network"}),
	}

	prometheus.MustRegister(
		m.apiRequestsTotal,
		m.bidsReceived,
		m.apiFailuresTotal,
	)

	return m
}

func (m *Metrics) IncAPIRequests(relay, endpoint, network string) {
	m.apiRequestsTotal.WithLabelValues(relay, endpoint, network).Inc()
}

func (m *Metrics) IncBidsReceived(relay, network string, count int) {
	m.bidsReceived.WithLabelValues(relay, network).Add(float64(count))
}

func (m *Metrics) IncAPIFailures(relay, endpoint, network string) {
	m.apiFailuresTotal.WithLabelValues(relay, endpoint, network).Inc()
}
package auth

import "github.com/prometheus/client_golang/prometheus"

var (
	DefaultGroupMetrics = NewGroupMetrics("xatu_event_ingester")
)

type GroupMetrics struct {
	fieldsRedacted *prometheus.CounterVec
}

func NewGroupMetrics(namespace string) *GroupMetrics {
	if namespace != "" {
		namespace += "_"
	}

	namespace += "group"

	m := &GroupMetrics{
		fieldsRedacted: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name:      "fields_redacted_total",
			Namespace: namespace,
			Help:      "Number of fields redacted",
		}, []string{"group", "path"}),
	}

	return m
}

func (m *GroupMetrics) IncFieldsRedacted(group, path string) {
	m.fieldsRedacted.WithLabelValues(group, path).Inc()
}

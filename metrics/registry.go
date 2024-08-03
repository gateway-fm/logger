package metrics

import (
	"strings"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

type registry struct {
	Subsystem    string
	Namespace    string
	PromRegistry *prometheus.Registry

	metricsMu  sync.Mutex
	counters   map[string]*prometheus.CounterVec
	histograms map[string]*prometheus.HistogramVec
}

// NewRegistry creates a new metrics registry with the specified subsystem and namespace.
func NewRegistry(subsystem, namespace string) Registry {
	r := &registry{
		Subsystem:    subsystem,
		Namespace:    namespace,
		PromRegistry: prometheus.NewRegistry(),
		counters:     make(map[string]*prometheus.CounterVec),
		histograms:   make(map[string]*prometheus.HistogramVec),
	}

	registerMetrics(r)

	return r
}

func (r *registry) sanitizeMetricName(name string) string {
	return strings.ReplaceAll(name, "-", "_")
}

// Inc increments a counter for the given Series.
func (r *registry) Inc(name string, labels prometheus.Labels) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()

	sanitized := r.sanitizeMetricName(name)
	counter, exists := r.counters[sanitized]
	if !exists {
		counter = prometheus.NewCounterVec(prometheus.CounterOpts{
			Subsystem: r.Subsystem,
			Namespace: r.Namespace,
			Name:      sanitized,
		}, []string{"series_type", "sub_type", "operation", "status", "error_code"})
		r.PromRegistry.MustRegister(counter)
		r.counters[sanitized] = counter
	}
	counter.With(labels).Inc()
}

// RecordDuration records a duration for the given Series.
func (r *registry) RecordDuration(name string, labels prometheus.Labels, duration float64) {
	r.metricsMu.Lock()
	defer r.metricsMu.Unlock()

	sanitized := r.sanitizeMetricName(name)
	histogram, exists := r.histograms[sanitized]
	if !exists {
		histogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Subsystem: r.Subsystem,
			Namespace: r.Namespace,
			Name:      sanitized,
			Buckets:   prometheus.DefBuckets,
		}, []string{"series_type", "sub_type", "operation"})
		r.PromRegistry.MustRegister(histogram)
		r.histograms[sanitized] = histogram
	}
	histogram.With(labels).Observe(duration)
}

// PrometheusRegistry returns the underlying Prometheus registry.
func (r *registry) PrometheusRegistry() *prometheus.Registry {
	return r.PromRegistry
}

func registerMetrics(registry *registry) {
	registry.PromRegistry.MustRegister(
		collectors.NewGoCollector(
			collectors.WithGoCollectorMemStatsMetricsDisabled(),
			collectors.WithGoCollectorRuntimeMetrics(collectors.MetricsScheduler),
		))
}

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all the prometheus metrics for the system.
type Metrics struct {
	CacheHits            *prometheus.CounterVec
	CacheMisses          *prometheus.CounterVec
	BackendCalls         *prometheus.CounterVec
	ConfidenceScore      prometheus.Histogram
	PolicyRejections     *prometheus.CounterVec
	CacheLatency         *prometheus.HistogramVec
	EmbeddingDuration    prometheus.Histogram
	BackendCostTotal     prometheus.Counter
	CacheCostSavedTotal  prometheus.Counter
}

var defaultMetrics *Metrics

// InitMetrics initializes the global prometheus metrics.
func InitMetrics() *Metrics {
	if defaultMetrics != nil {
		return defaultMetrics
	}

	m := &Metrics{
		CacheHits: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "cache_hits_total",
			Help: "The total number of cache hits by tier",
		}, []string{"tier", "tenant"}),
		
		CacheMisses: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "cache_misses_total",
			Help: "The total number of cache misses",
		}, []string{"tenant"}),

		BackendCalls: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "backend_calls_total",
			Help: "The total number of backend calls by domain",
		}, []string{"domain", "tenant"}),

		ConfidenceScore: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "l2b_confidence_score",
			Help:    "Distribution of confidence scores from semantic matches",
			Buckets: prometheus.LinearBuckets(0.5, 0.05, 10),
		}),

		PolicyRejections: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "policy_rejection_total",
			Help: "The total number of cache entries rejected by policy",
		}, []string{"reason", "domain"}),

		CacheLatency: promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "cache_latency_seconds",
			Help:    "Latency of cache operations in seconds by tier",
			Buckets: prometheus.DefBuckets,
		}, []string{"tier"}),

		EmbeddingDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "embedding_duration_seconds",
			Help:    "Duration of embedding generation in seconds",
			Buckets: prometheus.DefBuckets,
		}),

		BackendCostTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "backend_cost_usd_total",
			Help: "Theoretical cost of backend requests (simulated)",
		}),

		CacheCostSavedTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "cache_cost_saved_usd_total",
			Help: "Theoretical cost saved by using cache (simulated)",
		}),
	}

	defaultMetrics = m
	return m
}

func GetMetrics() *Metrics {
	if defaultMetrics == nil {
		return InitMetrics()
	}
	return defaultMetrics
}

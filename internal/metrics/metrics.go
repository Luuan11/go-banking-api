package metrics

import (
	"database/sql"
	"net/http"
	"runtime"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const ResultOK = "ok"

type Metrics struct {
	registry   *prometheus.Registry
	requests   *prometheus.CounterVec
	duration   *prometheus.HistogramVec
	inFlight   prometheus.Gauge
	operations *prometheus.CounterVec
	volume     *prometheus.CounterVec
}

func New(version string) *Metrics {
	m := &Metrics{
		registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "HTTP requests by method, route and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latency by method and route.",
			Buckets: []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5},
		}, []string{"method", "route"}),
		inFlight: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "HTTP requests being served.",
		}),
		operations: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "bank_operations_total",
			Help: "Money movements by operation and result (ok or error code).",
		}, []string{"operation", "result"}),
		volume: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "bank_operation_amount_cents_total",
			Help: "Amount moved by successful operations, in cents.",
		}, []string{"operation"}),
	}

	buildInfo := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "bank_build_info",
		Help: "Build information; always 1.",
	}, []string{"version", "go_version"})
	buildInfo.WithLabelValues(version, runtime.Version()).Set(1)

	m.registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
		buildInfo, m.requests, m.duration, m.inFlight, m.operations, m.volume,
	)
	return m
}

// RegisterDB exports connection pool stats (go_sql_*).
func (m *Metrics) RegisterDB(db *sql.DB) {
	m.registry.MustRegister(collectors.NewDBStatsCollector(db, "bankdb"))
}

func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{Registry: m.registry})
}

func (m *Metrics) RequestStarted() {
	m.inFlight.Inc()
}

func (m *Metrics) RequestFinished(method, route string, status int, d time.Duration) {
	m.inFlight.Dec()
	m.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	m.duration.WithLabelValues(method, route).Observe(d.Seconds())
}

func (m *Metrics) Operation(operation string, amount int64, result string) {
	m.operations.WithLabelValues(operation, result).Inc()
	if result == ResultOK {
		m.volume.WithLabelValues(operation).Add(float64(amount))
	}
}

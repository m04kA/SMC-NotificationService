package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics содержит все метрики приложения
type Metrics struct {
	// HTTP метрики
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPErrorsTotal     *prometheus.CounterVec

	// Database метрики
	DBQueriesTotal    *prometheus.CounterVec
	DBQueryDuration   *prometheus.HistogramVec
	DBErrorsTotal     *prometheus.CounterVec
	DBConnectionsActive prometheus.Gauge
	DBConnectionsIdle   prometheus.Gauge
	DBConnectionsMax    prometheus.Gauge
}

// New создаёт новый экземпляр метрик с автоматической регистрацией в Prometheus
func New(serviceName string) *Metrics {
	m := &Metrics{
		// HTTP метрики
		HTTPRequestsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_requests_total",
				Help: "Total number of HTTP requests",
			},
			[]string{"service", "method", "endpoint", "status_code"},
		),

		HTTPRequestDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "http_request_duration_seconds",
				Help:    "HTTP request duration in seconds",
				Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
			},
			[]string{"service", "method", "endpoint", "status_code"},
		),

		HTTPErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "http_errors_total",
				Help: "Total number of HTTP errors",
			},
			[]string{"service", "method", "endpoint", "status_code", "error_type"},
		),

		// Database метрики
		DBQueriesTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_queries_total",
				Help: "Total number of database queries",
			},
			[]string{"service", "operation", "table", "status"},
		),

		DBQueryDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "db_query_duration_seconds",
				Help:    "Database query duration in seconds",
				Buckets: []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1},
			},
			[]string{"service", "operation", "table"},
		),

		DBErrorsTotal: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "db_errors_total",
				Help: "Total number of database errors",
			},
			[]string{"service", "operation", "table", "error_type"},
		),

		DBConnectionsActive: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_active",
				Help: "Number of active database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),

		DBConnectionsIdle: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_idle",
				Help: "Number of idle database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),

		DBConnectionsMax: promauto.NewGauge(
			prometheus.GaugeOpts{
				Name: "db_connections_max",
				Help: "Maximum number of database connections",
				ConstLabels: prometheus.Labels{
					"service": serviceName,
				},
			},
		),
	}

	return m
}

// RecordHTTPRequest записывает метрики HTTP запроса
func (m *Metrics) RecordHTTPRequest(service, method, endpoint, statusCode string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(service, method, endpoint, statusCode).Inc()
	m.HTTPRequestDuration.WithLabelValues(service, method, endpoint, statusCode).Observe(duration)
}

// RecordHTTPError записывает метрику HTTP ошибки
func (m *Metrics) RecordHTTPError(service, method, endpoint, statusCode, errorType string) {
	m.HTTPErrorsTotal.WithLabelValues(service, method, endpoint, statusCode, errorType).Inc()
}

// RecordDBQuery записывает метрики database запроса
func (m *Metrics) RecordDBQuery(service, operation, table, status string, duration float64) {
	m.DBQueriesTotal.WithLabelValues(service, operation, table, status).Inc()
	m.DBQueryDuration.WithLabelValues(service, operation, table).Observe(duration)
}

// RecordDBError записывает метрику database ошибки
func (m *Metrics) RecordDBError(service, operation, table, errorType string) {
	m.DBErrorsTotal.WithLabelValues(service, operation, table, errorType).Inc()
}

// UpdateDBConnectionStats обновляет метрики connection pool
func (m *Metrics) UpdateDBConnectionStats(active, idle, max int) {
	m.DBConnectionsActive.Set(float64(active))
	m.DBConnectionsIdle.Set(float64(idle))
	m.DBConnectionsMax.Set(float64(max))
}

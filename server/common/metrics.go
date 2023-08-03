package common

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
)

// PlikMetrics handles Prometheus metrics
type PlikMetrics struct {
	reg *prometheus.Registry

	httpCounter *prometheus.CounterVec

	uploads          prometheus.Gauge
	anonymousUploads prometheus.Gauge
	users            prometheus.Gauge
	files            prometheus.Gauge
	size             prometheus.Gauge
	anonymousSize    prometheus.Gauge

	serverStatsRefreshDuration prometheus.Histogram
	cleaningDuration           prometheus.Histogram

	cleaningRemovedUploads prometheus.Counter
	cleaningDeletedFiles   prometheus.Counter
	cleaningDeletedUploads prometheus.Counter
	cleaningOrphanFiles    prometheus.Counter
	cleaningOrphanTokens   prometheus.Counter

	lastStatsRefresh prometheus.Gauge
	lastCleaning     prometheus.Gauge
}

// NewPlikMetrics initialize Plik metrics
func NewPlikMetrics() (m *PlikMetrics) {
	m = &PlikMetrics{reg: prometheus.NewRegistry()}

	// Runtime metrics
	m.reg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	m.reg.MustRegister(collectors.NewGoCollector())

	m.httpCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "plik_http_request_total",
		Help: "Count of HTTP requests",
	}, []string{"method", "path", "code"})
	m.reg.MustRegister(m.httpCounter)

	m.uploads = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_uploads_count",
		Help: "Total number of uploads in the database",
	})
	m.reg.MustRegister(m.uploads)

	m.anonymousUploads = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_anonymous_uploads_count",
		Help: "Total number of anonymous uploads in the database",
	})
	m.reg.MustRegister(m.anonymousUploads)

	m.users = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_users_count",
		Help: "Total number of users in the database",
	})
	m.reg.MustRegister(m.users)

	m.files = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_files_count",
		Help: "Total number of files in the database",
	})
	m.reg.MustRegister(m.files)

	m.anonymousSize = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_anonymous_size_bytes",
		Help: "Total anonymous upload size in the database",
	})
	m.reg.MustRegister(m.anonymousSize)

	m.size = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_size_bytes",
		Help: "Total upload size in the database",
	})
	m.reg.MustRegister(m.size)

	m.serverStatsRefreshDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "plik_server_stats_refresh_duration_second",
		Help:    "Duration of server stats refresh requests",
		Buckets: prometheus.ExponentialBucketsRange((10 * time.Millisecond).Seconds(), (10 * time.Second).Seconds(), 20),
	})
	m.reg.MustRegister(m.serverStatsRefreshDuration)

	m.cleaningDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "plik_cleaning_duration_second",
		Help:    "Duration of cleaning runs",
		Buckets: prometheus.ExponentialBucketsRange((10 * time.Millisecond).Seconds(), (600 * time.Second).Seconds(), 20),
	})
	m.reg.MustRegister(m.cleaningDuration)

	m.cleaningRemovedUploads = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "plik_cleaning_removed_uploads",
		Help: "Cleaning routine removed uploads",
	})
	m.reg.MustRegister(m.cleaningRemovedUploads)

	m.cleaningDeletedFiles = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "plik_cleaning_deleted_files",
		Help: "Cleaning routine deleted files",
	})
	m.reg.MustRegister(m.cleaningDeletedFiles)

	m.cleaningDeletedUploads = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "plik_cleaning_deleted_uploads",
		Help: "Cleaning routine deleted uploads",
	})
	m.reg.MustRegister(m.cleaningDeletedUploads)

	m.cleaningOrphanFiles = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "plik_cleaning_removed_orphan_files",
		Help: "Cleaning routine removed orphan files",
	})
	m.reg.MustRegister(m.cleaningOrphanFiles)

	m.cleaningOrphanTokens = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "plik_cleaning_removed_orphan_tokens",
		Help: "Cleaning routine removed orphan tokens",
	})
	m.reg.MustRegister(m.cleaningOrphanTokens)

	m.lastStatsRefresh = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_last_stats_refresh_timestamp",
		Help: "Timestamp of the last server stats refresh",
	})
	m.reg.MustRegister(m.lastStatsRefresh)

	m.lastCleaning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "plik_last_cleaning_timestamp",
		Help: "Timestamp of the last server cleaning",
	})
	m.reg.MustRegister(m.lastCleaning)

	return m
}

// GetRegistry returns the dedicated Prometheus Registry
func (m *PlikMetrics) GetRegistry() *prometheus.Registry {
	return m.reg
}

// UpdateHTTPMetrics update metrics about HTTP requests/responses
func (m *PlikMetrics) UpdateHTTPMetrics(method string, path string, statusCode int, elapsed time.Duration) {
	m.httpCounter.WithLabelValues(method, path, strconv.Itoa(statusCode)).Add(1)
}

// UpdateServerStatistics update metrics about plik metadata
func (m *PlikMetrics) UpdateServerStatistics(stats *ServerStats, elapsed time.Duration) {
	m.uploads.Set(float64(stats.Uploads))
	m.anonymousUploads.Set(float64(stats.AnonymousUploads))
	m.files.Set(float64(stats.Files))
	m.size.Set(float64(stats.TotalSize))
	m.anonymousSize.Set(float64(stats.AnonymousSize))
	m.users.Set(float64(stats.Users))
	m.lastStatsRefresh.Set(float64(time.Now().Second()))
	m.serverStatsRefreshDuration.Observe(elapsed.Seconds())
}

// UpdateCleaningStatistics update metrics about plik cleaning
func (m *PlikMetrics) UpdateCleaningStatistics(stats *CleaningStats, elapsed time.Duration) {
	m.cleaningRemovedUploads.Add(float64(stats.RemovedUploads))
	m.cleaningDeletedFiles.Add(float64(stats.DeletedFiles))
	m.cleaningDeletedUploads.Add(float64(stats.DeletedUploads))
	m.cleaningOrphanFiles.Add(float64(stats.OrphanFilesCleaned))
	m.cleaningOrphanTokens.Add(float64(stats.OrphanTokensCleaned))
	m.lastCleaning.Set(float64(time.Now().Second()))
	m.cleaningDuration.Observe(elapsed.Seconds())
}

// Register a set of collectors to the dedicated Prometheus registry
// This can be used by modules to register dedicated metrics
func (m *PlikMetrics) Register(collectors ...prometheus.Collector) {
	m.reg.MustRegister(collectors...)
}

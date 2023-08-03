package prometheus

import (
	"database/sql"
	"github.com/prometheus/client_golang/prometheus"
	"reflect"
)

type DBStats struct {
	MaxOpenConnections prometheus.Gauge // Maximum number of open connections to the database.

	// Pool status
	OpenConnections prometheus.Gauge // The number of established connections both in use and idle.
	InUse           prometheus.Gauge // The number of connections currently in use.
	Idle            prometheus.Gauge // The number of idle connections.

	// Counters
	WaitCount         prometheus.Gauge // The total number of connections waited for.
	WaitDuration      prometheus.Gauge // The total time blocked waiting for a new connection.
	MaxIdleClosed     prometheus.Gauge // The total number of connections closed due to SetMaxIdleConns.
	MaxLifetimeClosed prometheus.Gauge // The total number of connections closed due to SetConnMaxLifetime.
	MaxIdleTimeClosed prometheus.Gauge // The total number of connections closed due to SetConnMaxIdleTime.
}

func newStats(labels map[string]string) *DBStats {
	stats := &DBStats{
		MaxOpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_max_open_connections",
			Help:        "Maximum number of open connections to the database.",
			ConstLabels: labels,
		}),
		OpenConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_open_connections",
			Help:        "The number of established connections both in use and idle.",
			ConstLabels: labels,
		}),
		InUse: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_in_use",
			Help:        "The number of connections currently in use.",
			ConstLabels: labels,
		}),
		Idle: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_idle",
			Help:        "The number of idle connections.",
			ConstLabels: labels,
		}),
		WaitCount: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_wait_count",
			Help:        "The total number of connections waited for.",
			ConstLabels: labels,
		}),
		WaitDuration: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_wait_duration",
			Help:        "The total time blocked waiting for a new connection.",
			ConstLabels: labels,
		}),
		MaxIdleClosed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_max_idle_closed",
			Help:        "The total number of connections closed due to SetMaxIdleConns.",
			ConstLabels: labels,
		}),
		MaxLifetimeClosed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_max_lifetime_closed",
			Help:        "The total number of connections closed due to SetConnMaxLifetime.",
			ConstLabels: labels,
		}),
		MaxIdleTimeClosed: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        "gorm_dbstats_max_idletime_closed",
			Help:        "The total number of connections closed due to SetConnMaxIdleTime.",
			ConstLabels: labels,
		}),
	}

	for _, collector := range stats.Collectors() {
		_ = prometheus.Register(collector)
	}

	return stats
}

func (stats *DBStats) Set(dbStats sql.DBStats) {
	stats.MaxOpenConnections.Set(float64(dbStats.MaxOpenConnections))
	stats.OpenConnections.Set(float64(dbStats.OpenConnections))
	stats.InUse.Set(float64(dbStats.InUse))
	stats.Idle.Set(float64(dbStats.Idle))
	stats.WaitCount.Set(float64(dbStats.WaitCount))
	stats.WaitDuration.Set(float64(dbStats.WaitDuration))
	stats.MaxIdleClosed.Set(float64(dbStats.MaxIdleClosed))
	stats.MaxLifetimeClosed.Set(float64(dbStats.MaxLifetimeClosed))
	stats.MaxIdleTimeClosed.Set(float64(dbStats.MaxIdleTimeClosed))
}

//get collector in stats
func (stats *DBStats) Collectors() (collector []prometheus.Collector) {
	dbStatsValue := reflect.ValueOf(*stats)
	for i := 0; i < dbStatsValue.NumField(); i++ {
		collector = append(collector, dbStatsValue.Field(i).Interface().(prometheus.Gauge))
	}
	return
}

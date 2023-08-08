package common

import (
	"strconv"
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"
)

func TestNewPlikMetrics(t *testing.T) {
	m := NewPlikMetrics()
	require.NotNil(t, m)

	require.NotNil(t, m.reg)

	require.NotNil(t, m.httpCounter)

	require.NotNil(t, m.uploads)
	require.NotNil(t, m.anonymousUploads)
	require.NotNil(t, m.users)
	require.NotNil(t, m.files)
	require.NotNil(t, m.size)
	require.NotNil(t, m.anonymousSize)

	require.NotNil(t, m.serverStatsRefreshDuration)
	require.NotNil(t, m.cleaningDuration)

	require.NotNil(t, m.cleaningRemovedUploads)
	require.NotNil(t, m.cleaningDeletedFiles)
	require.NotNil(t, m.cleaningDeletedUploads)
	require.NotNil(t, m.cleaningOrphanFiles)
	require.NotNil(t, m.cleaningOrphanTokens)

	require.NotNil(t, m.lastStatsRefresh)
	require.NotNil(t, m.lastCleaning)
}

func TestGetRegistry(t *testing.T) {
	m := NewPlikMetrics()
	require.Equal(t, m.reg, m.GetRegistry())
}

func TestUpdateHttpMetrics(t *testing.T) {
	m := NewPlikMetrics()
	m.UpdateHTTPMetrics("GET", "/upload", 200, time.Second)
	counter, err := m.httpCounter.GetMetricWithLabelValues("GET", "/upload", strconv.Itoa(200))
	require.NoError(t, err)

	metric := &dto.Metric{}
	err = counter.Write(metric)
	require.NoError(t, err)

	require.Equal(t, float64(1), *metric.GetCounter().Value)
}

func TestUpdateServerStatistics(t *testing.T) {
	m := NewPlikMetrics()
	stats := &ServerStats{
		Users:            1,
		Uploads:          2,
		AnonymousUploads: 3,
		TotalSize:        4,
		AnonymousSize:    5,
	}
	m.UpdateServerStatistics(stats, 1*time.Second)

	metric := &dto.Metric{}

	err := m.uploads.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.Uploads), *metric.GetGauge().Value)

	err = m.anonymousUploads.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.AnonymousUploads), *metric.GetGauge().Value)

	err = m.files.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.Files), *metric.GetGauge().Value)

	err = m.size.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.TotalSize), *metric.GetGauge().Value)

	err = m.anonymousSize.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.AnonymousSize), *metric.GetGauge().Value)

	err = m.users.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.Users), *metric.GetGauge().Value)

	err = m.lastStatsRefresh.Write(metric)
	require.NoError(t, err)
	require.NotZero(t, *metric.GetGauge().Value)

	err = m.serverStatsRefreshDuration.Write(metric)
	require.NoError(t, err)
	require.Equal(t, uint64(1), *metric.GetHistogram().SampleCount)
}

func TestUpdateCleaningStatistics(t *testing.T) {
	m := NewPlikMetrics()
	stats := &CleaningStats{
		RemovedUploads:      1,
		DeletedFiles:        2,
		DeletedUploads:      3,
		OrphanFilesCleaned:  4,
		OrphanTokensCleaned: 5,
	}
	m.UpdateCleaningStatistics(stats, 1*time.Second)

	metric := &dto.Metric{}

	err := m.cleaningRemovedUploads.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.RemovedUploads), *metric.GetCounter().Value)

	err = m.cleaningDeletedFiles.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.DeletedFiles), *metric.GetCounter().Value)

	err = m.cleaningDeletedUploads.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.DeletedUploads), *metric.GetCounter().Value)

	err = m.cleaningOrphanFiles.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.OrphanFilesCleaned), *metric.GetCounter().Value)

	err = m.cleaningOrphanTokens.Write(metric)
	require.NoError(t, err)
	require.Equal(t, float64(stats.OrphanTokensCleaned), *metric.GetCounter().Value)

	err = m.lastCleaning.Write(metric)
	require.NoError(t, err)
	require.NotZero(t, *metric.GetGauge().Value)

	err = m.cleaningDuration.Write(metric)
	require.NoError(t, err)
	require.Equal(t, uint64(1), *metric.GetHistogram().SampleCount)
}

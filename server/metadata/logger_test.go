package metadata

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/root-gg/logger"
	"github.com/stretchr/testify/require"
	gorm_logger "gorm.io/gorm/logger"
)

func TestNewGormLoggerAdapter(t *testing.T) {
	gormLoggerAdapter := NewGormLoggerAdapter(logger.NewLogger())
	require.NotNil(t, gormLoggerAdapter)
	require.NotNil(t, gormLoggerAdapter.LogMode(gorm_logger.Info))
	gormLoggerAdapter.Info(context.Background(), "info %s", "message")
	gormLoggerAdapter.Warn(context.Background(), "warn %s", "message")
	gormLoggerAdapter.Error(context.Background(), "error %s", "message")
}

func TestGormLoggerAdapter_Trace(t *testing.T) {
	gormLoggerAdapter := NewGormLoggerAdapter(logger.NewLogger())

	f := func(query string, rows int) func() (string, int64) {
		return func() (string, int64) {
			return query, int64(rows)
		}
	}

	// SQL ERROR
	gormLoggerAdapter.Trace(context.Background(), time.Now(), f("SQL QUERY", -1), errors.New("SQL ERROR"))
	gormLoggerAdapter.Trace(context.Background(), time.Now(), f("SQL QUERY", 1), errors.New("SQL ERROR"))

	// SLOW QUERY
	gormLoggerAdapter.Trace(context.Background(), time.Now().Add(-10*time.Second), f("SQL QUERY", -1), nil)
	gormLoggerAdapter.Trace(context.Background(), time.Now().Add(-10*time.Second), f("SQL QUERY", 1), nil)

	// TRACE QUERY
	gormLoggerAdapter.Trace(context.Background(), time.Now(), f("SQL QUERY", -1), nil)
	gormLoggerAdapter.Trace(context.Background(), time.Now(), f("SQL QUERY", 1), nil)
}

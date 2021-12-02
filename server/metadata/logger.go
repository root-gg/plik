package metadata

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/root-gg/logger"
	gorm_logger "gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

// GormLoggerAdapter forward Gorm logs to a root-gg logger
type GormLoggerAdapter struct {
	logger             *logger.Logger
	SlowQueryThreshold time.Duration
}

// NewGormLoggerAdapter create a new GormLoggerAdapter for a root-gg logger
func NewGormLoggerAdapter(log *logger.Logger) *GormLoggerAdapter {
	return &GormLoggerAdapter{
		logger:             log,
		SlowQueryThreshold: time.Second,
	}
}

// LogMode is not relevant as log level is managed by the root-gg logger level
func (l *GormLoggerAdapter) LogMode(level gorm_logger.LogLevel) gorm_logger.Interface {
	return l
}

// Info print info
func (l *GormLoggerAdapter) Info(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Infof(msg, data...)
}

// Warn print warn messages
func (l *GormLoggerAdapter) Warn(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Warningf(msg, data...)
}

// Error print error messages
func (l *GormLoggerAdapter) Error(ctx context.Context, msg string, data ...interface{}) {
	l.logger.Criticalf(msg, data...)
}

// Trace print sql message
func (l *GormLoggerAdapter) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	elapsed := time.Since(begin)
	switch {
	case err != nil && !errors.Is(err, gorm_logger.ErrRecordNotFound): // Do not log RecordNotFound errors
		sql, rows := fc()
		if rows == -1 {
			l.logger.Warningf("%s %s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Warningf("%s %s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), err, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	case l.SlowQueryThreshold > 0 && elapsed > l.SlowQueryThreshold:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowQueryThreshold)
		if rows == -1 {
			l.logger.Warningf("%s %s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Warningf("%s %s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), slowLog, float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	default:
		sql, rows := fc()
		if rows == -1 {
			l.logger.Debugf("%s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, "-", sql)
		} else {
			l.logger.Debugf("%s\n[%.3fms] [rows:%v] %s", utils.FileWithLineNum(), float64(elapsed.Nanoseconds())/1e6, rows, sql)
		}
	}
}

package metadata

import (
	"github.com/root-gg/plik/server/common"
	"gorm.io/gorm"
)

// GetUserStatistics returns counter-backed current/lifetime user statistics.
// When tokenStr is nil it reads the user row and populates the bounded download
// windows from the user's daily series; when tokenStr is set it reads
// token-scoped usage for that user (empty token matches no-token usage) and
// omits the windows, because token scope has no daily rollup series.
func (b *Backend) GetUserStatistics(userID string, tokenStr *string) (stats *common.UserStats, err error) {
	token := ""
	if tokenStr != nil {
		token = *tokenStr
	}

	usage := &common.UsageStats{}
	err = b.db.Where("user_id = ? AND token = ?", userID, token).Take(usage).Error
	if err == nil {
		stats = userStatsFromUsage(usage)
	} else if err == gorm.ErrRecordNotFound {
		stats = &common.UserStats{Usage: &common.UsageStatsResponse{}}
	} else {
		return nil, err
	}

	// Only the user scope (no token filter) carries the bounded download/upload
	// windows, derived from the merged activity series so they match the chart.
	if tokenStr == nil {
		points, err := b.GetUserActivityStatsDaily(userID, activityWindowDays)
		if err != nil {
			return nil, err
		}
		applyActivityWindows(stats.Usage, points)
	}

	return stats, nil
}

// GetServerStatistics returns the counter-backed server statistics.
//
// There is no server usage_stats row: server totals are the sum of every
// token=” row (users + anonymous + the deleted-user tombstone). Token rows
// (token!=”) are excluded so token'd uploads — which fan out to both the owner's
// user row and the token row — are not double-counted. The summed columns and the
// select list are registry-generated (serverUsageSumSelect), and startedAt is the
// MIN(started_at) across those rows, anchored by the migration/init-created
// tombstone so it is never NULL. AnonymousUploads/Size still read the anonymous
// row directly.
func (b *Backend) GetServerStatistics() (stats *common.ServerStats, err error) {
	usage := &common.UsageStats{}
	err = b.db.Model(&common.UsageStats{}).
		Where("token = ?", "").
		Select(serverUsageSumSelect()).
		Scan(usage).Error
	if err != nil {
		return nil, err
	}

	// startedAt is the "stats since" anchor: the earliest started_at among the
	// summed rows. It is read via a model-aware ordered Take rather than a raw
	// MIN(started_at) aggregate, because MIN over the column loses its datetime
	// affinity on SQLite and comes back as an unscannable string. The tombstone
	// (created at migration/init) guarantees at least one token='' row exists, so
	// this normally finds one; an empty result leaves the zero anchor.
	anchor := &common.UsageStats{}
	err = b.db.Model(&common.UsageStats{}).
		Where("token = ?", "").
		Order("started_at ASC").
		Limit(1).
		Take(anchor).Error
	if err == nil {
		usage.StartedAt = anchor.StartedAt
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	anonymous := &common.UsageStats{}
	err = b.db.Where("user_id = ? AND token = ?", common.AnonymousUserUsageStatsID, "").Take(anonymous).Error
	if err == gorm.ErrRecordNotFound {
		anonymous = &common.UsageStats{}
	} else if err != nil {
		return nil, err
	}

	users, err := b.CountUsers("", nil)
	if err != nil {
		return nil, err
	}
	stats = serverStatsFromUsage(usage, anonymous, users)

	err = b.populateServerActivityWindows(stats)
	if err != nil {
		return nil, err
	}

	return stats, nil
}

package metadata

import (
	"fmt"
	"time"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/root-gg/plik/server/common"
)

// CreateToken create a new token in DB
func (b *Backend) CreateToken(token *common.Token) (err error) {
	return b.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(token).Error
		if err != nil {
			return err
		}

		now := time.Now()
		return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
			UserID:    token.UserID,
			Token:     token.Token,
			StartedAt: tokenUsageStartedAt(token, now),
		}).Error
	})
}

// tokenUsageStartedAt uses the token creation date as the "Stats since" date
// when the token row is already materialized. The fallback covers imported or
// hand-built token values that do not carry GORM timestamps yet.
func tokenUsageStartedAt(token *common.Token, fallback time.Time) time.Time {
	if token != nil && !token.CreatedAt.IsZero() {
		return token.CreatedAt
	}
	return fallback
}

// GetToken return a token from the DB ( return nil and non error if not found )
func (b *Backend) GetToken(tokenStr string) (token *common.Token, err error) {
	token = &common.Token{}
	err = b.db.Where(&common.Token{Token: tokenStr}).Take(token).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return token, err
}

// GetTokens return all tokens for a user.
func (b *Backend) GetTokens(userID string, sort string, pagingQuery *common.PagingQuery) (tokens []*common.Token, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}
	if sort == "" {
		sort = StatsSortDate
	}
	if sort == StatsSortSize || sort == StatsSortLifetimeSize {
		return b.getTokensSortedByUsage(userID, sort, pagingQuery)
	}
	if sort != StatsSortDate {
		return nil, nil, fmt.Errorf("invalid token sort")
	}

	stmt := b.db.Model(&common.Token{}).Where(&common.Token{UserID: userID})

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "Token")

	result, c, err := p.Paginate(stmt, &tokens)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	err = b.attachTokenUsageStats(tokens)
	if err != nil {
		return nil, nil, err
	}

	return tokens, &c, err
}

func (b *Backend) getTokensSortedByUsage(userID string, sort string, pagingQuery *common.PagingQuery) (tokens []*common.Token, cursor *paginator.Cursor, err error) {
	// Sort/paginate on the stats table, then fetch full token rows and restore the
	// cursor order. This keeps pagination stable while avoiding raw token values in
	// URL parameters beyond the existing cursor token.
	type tokenRef struct {
		Token        string
		Size         int64
		LifetimeSize int64
	}
	var refs []*tokenRef

	stmt := b.db.
		Model(&common.Token{}).
		Select("tokens.token, COALESCE(usage_stats.current_size, 0) as size, COALESCE(usage_stats.lifetime_size, 0) as lifetime_size").
		Joins("LEFT JOIN usage_stats ON usage_stats.user_id = tokens.user_id AND usage_stats.token = tokens.token").
		Where(&common.Token{UserID: userID})

	p := pagingQuery.Paginator()
	if sort == StatsSortLifetimeSize {
		p.SetRules(
			paginator.Rule{Key: "LifetimeSize", SQLRepr: "COALESCE(usage_stats.lifetime_size, 0)"},
			paginator.Rule{Key: "Token", SQLRepr: "tokens.token"},
		)
	} else {
		p.SetRules(
			paginator.Rule{Key: "Size", SQLRepr: "COALESCE(usage_stats.current_size, 0)"},
			paginator.Rule{Key: "Token", SQLRepr: "tokens.token"},
		)
	}

	result, c, err := p.Paginate(stmt, &refs)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}
	if len(refs) == 0 {
		return tokens, &c, nil
	}

	tokenStrings := make([]string, len(refs))
	for i, ref := range refs {
		tokenStrings[i] = ref.Token
	}
	err = b.db.Where("token IN ?", tokenStrings).Find(&tokens).Error
	if err != nil {
		return nil, nil, err
	}
	err = b.attachTokenUsageStats(tokens)
	if err != nil {
		return nil, nil, err
	}

	// Reorder to match the pagination sort order
	tokens = reorderByRefs(tokenStrings, tokens, func(t *common.Token) string { return t.Token })

	return tokens, &c, nil
}

func (b *Backend) attachTokenUsageStats(tokens []*common.Token) error {
	if len(tokens) == 0 {
		return nil
	}

	// Token stats are stored in usage_stats so upload/delete paths can update
	// counters atomically. Attach them in bulk for the home token list response.
	tokenStrings := make([]string, len(tokens))
	for i, token := range tokens {
		tokenStrings[i] = token.Token
	}

	var usages []*common.UsageStats
	err := b.db.Where("token IN ?", tokenStrings).Find(&usages).Error
	if err != nil {
		return err
	}

	byToken := make(map[string]*common.UsageStats, len(usages))
	for _, usage := range usages {
		byToken[usage.Token] = usage
	}
	for _, token := range tokens {
		if usage := byToken[token.Token]; usage != nil {
			// token.Stats wraps the canonical nested usage payload in the usage{}
			// envelope; lastUploadAt lives inside it (token.Stats.Usage.LastUploadAt)
			// and nowhere else. Token scope has no daily rollup series, so the
			// download windows stay omitted.
			token.Stats = &common.TokenStats{Usage: usageStatsResponseFromUsage(usage)}
		}
	}

	return nil
}

// DeleteToken remove a token from the DB
func (b *Backend) DeleteToken(tokenStr string) (deleted bool, err error) {

	// Delete token
	err = b.db.Transaction(func(tx *gorm.DB) error {
		deleted = false

		result := tx.Delete(&common.Token{Token: tokenStr})
		if result.Error != nil {
			return fmt.Errorf("unable to delete token metadata : %s", result.Error)
		}
		deleted = result.RowsAffected > 0
		if deleted {
			err = tx.Where("token = ?", tokenStr).Delete(&common.UsageStats{}).Error
			if err != nil {
				return fmt.Errorf("unable to delete token usage stats : %s", err)
			}
		}
		return nil
	})

	return deleted, err
}

// CountUserTokens count how many token a user has
func (b *Backend) CountUserTokens(userID string) (count int, err error) {
	var c int64 // Gorm V2 needs int64 for counts
	err = b.db.Model(&common.Token{}).Where(&common.Token{UserID: userID}).Count(&c).Error
	if err != nil {
		return -1, err
	}

	return int(c), nil
}

// ForEachToken execute f for every token in the database
func (b *Backend) ForEachToken(f func(token *common.Token) error) (err error) {
	stmt := b.db.Model(&common.Token{})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		token := &common.Token{}
		err = b.db.ScanRows(rows, token)
		if err != nil {
			return err
		}
		err = f(token)
		if err != nil {
			return err
		}
	}

	return nil
}

package metadata

import (
	"fmt"
	"time"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/root-gg/plik/server/common"
)

// CreateUser create a new user in DB
func (b *Backend) CreateUser(user *common.User) (err error) {
	return b.db.Transaction(func(tx *gorm.DB) error {
		err := tx.Create(user).Error
		if err != nil {
			return err
		}

		// Eagerly seed the user's own usage row with lifetime_users = 1 (mirroring
		// CreateToken's eager token-row seeding). lifetime_users is a uniform
		// per-user counter: server lifetime_users is Σ over token='' rows, so a new
		// user contributes exactly its own +1 and DeleteUser folds it into the
		// tombstone rather than decrementing anything. started_at is the user's
		// creation time so the "stats since" anchor matches the account.
		now := time.Now()
		startedAt := user.CreatedAt
		if startedAt.IsZero() {
			startedAt = now
		}
		err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
			UserID:        user.ID,
			LifetimeUsers: 1,
			StartedAt:     startedAt,
		}).Error
		if err != nil {
			return err
		}
		for _, token := range user.Tokens {
			err = tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&common.UsageStats{
				UserID:    user.ID,
				Token:     token.Token,
				StartedAt: tokenUsageStartedAt(token, now),
			}).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateUser update user info in DB
func (b *Backend) UpdateUser(user *common.User) (err error) {
	result := b.db.Save(user)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(1) {
		return fmt.Errorf("no user updated")
	}

	return nil
}

// GetUser return a user from DB ( return nil and no error if not found )
func (b *Backend) GetUser(ID string) (user *common.User, err error) {
	user = &common.User{}
	err = b.db.Where(&common.User{ID: ID}).Take(user).Error
	if err == gorm.ErrRecordNotFound {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return user, err
}

const (
	StatsSortDate         = "date"
	StatsSortSize         = "size"
	StatsSortLifetimeSize = "lifetimeSize"
	// StatsSortDownloadedBytes sorts by the scope's lifetime downloaded bytes
	// (usage_stats.downloaded_bytes). Currently wired only for GET /users
	// (isUserSort in server/handlers/admin.go) — GET /me/token intentionally
	// keeps the narrower isUsageStatsSort list, so this value is invalid there.
	StatsSortDownloadedBytes = "downloadedBytes"
)

// GetUsers return all users.
// provider is an optional filter.
// admin is an optional filter ( nil = no filter, true = admins only, false = non-admins only ).
func (b *Backend) GetUsers(provider string, admin *bool, withTokens bool, sort string, pagingQuery *common.PagingQuery) (users []*common.User, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
	}
	if sort == "" {
		sort = StatsSortDate
	}
	if sort == StatsSortSize || sort == StatsSortLifetimeSize || sort == StatsSortDownloadedBytes {
		return b.getUsersSortedByUsage(provider, admin, withTokens, sort, pagingQuery)
	}
	if sort != StatsSortDate {
		return nil, nil, fmt.Errorf("invalid user sort")
	}

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	stmt := b.db.Model(&common.User{})

	if withTokens {
		stmt = stmt.Preload("Tokens")
	}

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		// Use raw SQL instead of struct-based Where because GORM ignores zero-value
		// fields in structs, and false is the zero value for bool. Using the struct
		// pattern would silently skip the filter when querying for non-admin users.
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	result, c, err := p.Paginate(stmt, &users)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	err = b.attachUserUsageStats(users)
	if err != nil {
		return nil, nil, err
	}

	return users, &c, err
}

func (b *Backend) getUsersSortedByUsage(provider string, admin *bool, withTokens bool, sort string, pagingQuery *common.PagingQuery) (users []*common.User, cursor *paginator.Cursor, err error) {
	// The cursor paginator can sort on joined stats columns, but we still want to
	// return full User models (and optionally preloaded tokens). First page stable
	// user IDs with the usage value, then hydrate the models and restore the page
	// order from the ref slice.
	type userRef struct {
		ID              string
		Size            int64
		LifetimeSize    int64
		DownloadedBytes int64
	}
	var refs []*userRef

	stmt := b.db.
		Model(&common.User{}).
		Select("users.id, COALESCE(usage_stats.current_size, 0) as size, COALESCE(usage_stats.lifetime_size, 0) as lifetime_size, COALESCE(usage_stats.downloaded_bytes, 0) as downloaded_bytes").
		Joins("LEFT JOIN usage_stats ON usage_stats.user_id = users.id AND usage_stats.token = ?", "")

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}
	if admin != nil {
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	p := pagingQuery.Paginator()
	switch sort {
	case StatsSortLifetimeSize:
		p.SetRules(
			paginator.Rule{Key: "LifetimeSize", SQLRepr: "COALESCE(usage_stats.lifetime_size, 0)"},
			paginator.Rule{Key: "ID", SQLRepr: "users.id"},
		)
	case StatsSortDownloadedBytes:
		p.SetRules(
			paginator.Rule{Key: "DownloadedBytes", SQLRepr: "COALESCE(usage_stats.downloaded_bytes, 0)"},
			paginator.Rule{Key: "ID", SQLRepr: "users.id"},
		)
	default:
		p.SetRules(
			paginator.Rule{Key: "Size", SQLRepr: "COALESCE(usage_stats.current_size, 0)"},
			paginator.Rule{Key: "ID", SQLRepr: "users.id"},
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
		return users, &c, nil
	}

	ids := make([]string, len(refs))
	for i, ref := range refs {
		ids[i] = ref.ID
	}

	query := b.db.Where("id IN ?", ids)
	if withTokens {
		query = query.Preload("Tokens")
	}
	err = query.Find(&users).Error
	if err != nil {
		return nil, nil, err
	}
	err = b.attachUserUsageStats(users)
	if err != nil {
		return nil, nil, err
	}

	// Reorder to match the pagination sort order
	users = reorderByRefs(ids, users, func(u *common.User) string { return u.ID })

	return users, &c, nil
}

func (b *Backend) attachUserUsageStats(users []*common.User) error {
	if len(users) == 0 {
		return nil
	}

	// Usage stats are not a GORM association on User. Attach them in one batched
	// query so listing/searching users does not degrade into N+1 stats lookups.
	ids := make([]string, len(users))
	for i, user := range users {
		ids[i] = user.ID
	}

	var usages []*common.UsageStats
	err := b.db.Where("user_id IN ? AND token = ?", ids, "").Find(&usages).Error
	if err != nil {
		return err
	}

	byID := make(map[string]*common.UsageStats, len(usages))
	for _, usage := range usages {
		byID[usage.UserID] = usage
	}
	for _, user := range users {
		user.Stats = userStatsFromUsage(byID[user.ID])
	}

	return nil
}

// SearchUsers returns users matching a LIKE query on id, login, name, and email.
// Results are always sorted by login and hard-capped at limit (max 20).
// provider and admin are optional filters, same as GetUsers.
func (b *Backend) SearchUsers(query string, provider string, admin *bool, limit int) (users []*common.User, err error) {
	if query == "" {
		return nil, fmt.Errorf("missing search query")
	}
	if limit <= 0 || limit > 20 {
		limit = 20
	}

	pattern := "%" + query + "%"
	stmt := b.db.Model(&common.User{}).
		Where("id LIKE ? OR login LIKE ? OR name LIKE ? OR email LIKE ?", pattern, pattern, pattern, pattern)

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	stmt = stmt.Order("login ASC").Limit(limit)

	err = stmt.Find(&users).Error
	if err != nil {
		return nil, err
	}

	if users == nil {
		users = []*common.User{}
	}

	err = b.attachUserUsageStats(users)
	if err != nil {
		return nil, err
	}

	return users, nil
}

// ForEachUserUploads execute f for all upload matching the user and token filters
func (b *Backend) ForEachUserUploads(userID string, tokenStr string, f func(upload *common.Upload) error) (err error) {
	stmt := b.db.Model(&common.Upload{}).Where(&common.Upload{User: userID, Token: tokenStr})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		upload := &common.Upload{}
		err = b.db.ScanRows(rows, upload)
		if err != nil {
			return err
		}
		err = f(upload)
		if err != nil {
			return err
		}
	}

	return nil
}

// RemoveUserUploads soft-deletes all uploads matching the user and token
// filters: it marks their files for cleanup and soft-deletes the uploads, and
// the cleanup job later removes file data from the data backend.
//
// It lists the target uploads first (ascending id so per-upload locks are taken
// in a stable order across the batch), then removes each one through the
// race-free single-upload path. Each RemoveUpload transaction takes the parent
// upload row's lock first and derives its counter deltas from file rows that
// cannot change under it, so bulk removal decrements user/server/token counters
// exactly once per file — the same guarantee as removing the uploads one by one.
// Routing every scope (user/server/token) through RemoveUpload's shared logic
// also drops the previous per-token aggregate N+1 and reuses userUsageStatsID()
// consistently instead of a raw user id.
func (b *Backend) RemoveUserUploads(userID string, tokenStr string) (removed int, err error) {
	stmt := b.db.Model(&common.Upload{}).Where(&common.Upload{User: userID})
	if tokenStr != "" {
		stmt = stmt.Where(&common.Upload{Token: tokenStr})
	}

	var uploadIDs []string
	err = stmt.Order("id ASC").Pluck("id", &uploadIDs).Error
	if err != nil {
		return 0, fmt.Errorf("unable to list user uploads : %s", err)
	}

	for _, uploadID := range uploadIDs {
		err = b.RemoveUpload(uploadID)
		if err != nil {
			return removed, fmt.Errorf("unable to remove upload %s : %s", uploadID, err)
		}
		removed++
	}

	return removed, nil
}

// DeleteUser delete a user from the DB
func (b *Backend) DeleteUser(userID string) (deleted bool, err error) {
	_, err = b.RemoveUserUploads(userID, "")
	if err != nil {
		return false, err
	}

	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		deleted = false

		// Delete user tokens
		err = tx.Where(&common.Token{UserID: userID}).Delete(&common.Token{}).Error
		if err != nil {
			return fmt.Errorf("unable to delete tokens metadata : %s", err)
		}
		// Token usage rows are dropped with the user, not folded (their history
		// goes with the revoked token — the revoked-token tests pin this).
		err = tx.Where("user_id = ? AND token != ?", userID, "").Delete(&common.UsageStats{}).Error
		if err != nil {
			return fmt.Errorf("unable to delete token usage stats : %s", err)
		}

		// Delete user
		result := tx.Where(&common.User{ID: userID}).Delete(common.User{})
		if result.Error != nil {
			return fmt.Errorf("unable to delete user metadata : %s", result.Error)
		}

		if result.RowsAffected > 0 {
			deleted = true
		}

		if deleted {
			// Fold the user's own usage row into the deleted-user tombstone before
			// dropping it, so server lifetime totals (Σ over token='' rows, including
			// lifetime_users) are preserved across the deletion. RemoveUserUploads
			// above already zeroed the user's current counters, so only lifetime
			// counters and lifetime_users move into the tombstone; server current
			// totals are unaffected. Reading the row first also makes a repeat delete
			// idempotent: the second call finds no row and folds nothing.
			//
			// The read takes the row's write lock (FOR UPDATE, dialect-guarded by
			// applyUpdateLock) so the snapshot folded into the tombstone and the row
			// deleted just below are consistent: a concurrent download's usage UPDATE
			// committing in the read→delete window would otherwise be swallowed by the
			// delete, under-counting server downloads/bytes.
			usage := &common.UsageStats{}
			err = b.applyUpdateLock(tx).Where("user_id = ? AND token = ?", userID, "").Take(usage).Error
			if err == nil {
				err = b.foldUsageIntoDeletedTombstone(tx, usage)
				if err != nil {
					return fmt.Errorf("unable to fold user usage stats into tombstone : %s", err)
				}
				err = tx.Where("user_id = ? AND token = ?", userID, "").Delete(&common.UsageStats{}).Error
				if err != nil {
					return fmt.Errorf("unable to delete user usage stats : %s", err)
				}
			} else if err != gorm.ErrRecordNotFound {
				return fmt.Errorf("unable to read user usage stats : %s", err)
			}
		}

		return nil
	})

	return deleted, err
}

// CountUsers count the number of users matching the optional filters
func (b *Backend) CountUsers(provider string, admin *bool) (count int64, err error) {
	stmt := b.db.Model(&common.User{})

	if provider != "" {
		stmt = stmt.Where(&common.User{Provider: provider})
	}

	if admin != nil {
		stmt = stmt.Where("is_admin = ?", *admin)
	}

	err = stmt.Count(&count).Error
	return count, err
}

// ForEachUsers execute f for every user in the database
func (b *Backend) ForEachUsers(f func(user *common.User) error) (err error) {
	rows, err := b.db.Model(&common.User{}).Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		user := &common.User{}
		err = b.db.ScanRows(rows, user)
		if err != nil {
			return err
		}
		err = f(user)
		if err != nil {
			return err
		}
	}

	return nil
}

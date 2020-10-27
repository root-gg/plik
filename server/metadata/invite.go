package metadata

import (
	"fmt"
	"time"

	paginator "github.com/pilagod/gorm-cursor-paginator"

	"github.com/jinzhu/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateInvite create a new invite in DB
func (b *Backend) CreateInvite(invite *common.Invite) (err error) {
	return b.db.Create(invite).Error
}

// GetInvite return an invite from the DB ( return nil and non error if not found )
func (b *Backend) GetInvite(inviteID string) (invite *common.Invite, err error) {
	invite = &common.Invite{}
	err = b.db.Where(&common.Invite{ID: inviteID}).Take(invite).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return invite, err
}

// DeleteInvite remove an invite from the DB
func (b *Backend) DeleteInvite(inviteID string) (deleted bool, err error) {

	result := b.db.Delete(&common.Invite{ID: inviteID})
	if result.Error != nil {
		return false, fmt.Errorf("unable to delete invite metadata")
	}

	return result.RowsAffected > 0, err
}

// GetUserInvites return all invites for a user
func (b *Backend) GetUserInvites(userID string, pagingQuery *common.PagingQuery) (invites []*common.Invite, cursor *paginator.Cursor, err error) {
	stmt := b.db.Model(&common.Invite{})

	if userID == "*" {
		// Select all invites
	} else if userID == "" {
		stmt = stmt.Where("issuer IS NULL")
	} else {
		stmt = stmt.Where(&common.Invite{Issuer: &userID})
	}

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "ID")

	err = p.Paginate(stmt, &invites).Error
	if err != nil {
		return nil, nil, err
	}

	c := p.GetNextCursor()
	return invites, &c, err
}

// CountUserInvites count how many invite a user has created
func (b *Backend) CountUserInvites(userID string) (count int, err error) {
	stmt := b.db.Model(&common.Invite{})

	if userID == "*" {
		// Select all invites
	} else if userID == "" {
		stmt = stmt.Where("issuer IS NULL")
	} else {
		stmt = stmt.Where(&common.Invite{Issuer: &userID})
	}

	err = stmt.Count(&count).Error
	if err != nil {
		return -1, err
	}

	return count, nil
}

// ForEachInvites execute f for every invite in the database
func (b *Backend) ForEachInvites(f func(invite *common.Invite) error) (err error) {
	stmt := b.db.Model(&common.Invite{})

	rows, err := stmt.Rows()
	if err != nil {
		return err
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		invite := &common.Invite{}
		err = b.db.ScanRows(rows, invite)
		if err != nil {
			return err
		}
		err = f(invite)
		if err != nil {
			return err
		}
	}

	return nil
}

// DeleteExpiredInvites deletes all expired invites
func (b *Backend) DeleteExpiredInvites() (removed int, err error) {
	rows, err := b.db.Model(&common.Invite{}).Where("expire_at < ?", time.Now()).Rows()
	if err != nil {
		return 0, fmt.Errorf("unable to fetch expired invites : %s", err)
	}
	defer func() { _ = rows.Close() }()

	var errors []error
	for rows.Next() {
		invite := &common.Invite{}
		err = b.db.ScanRows(rows, invite)
		if err != nil {
			return 0, fmt.Errorf("unable to fetch next expired invite : %s", err)
		}

		_, err := b.DeleteInvite(invite.ID)
		if err != nil {
			errors = append(errors, err)
			continue
		}

		removed++
	}

	if len(errors) > 0 {
		return removed, fmt.Errorf("unable to remove %d expired invites", len(errors))
	}

	return removed, nil
}

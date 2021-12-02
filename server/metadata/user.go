package metadata

import (
	"fmt"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateUser create a new user in DB
func (b *Backend) CreateUser(user *common.User) (err error) {
	return b.db.Create(user).Error
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

// GetUsers return all users
// provider is an optional filter
func (b *Backend) GetUsers(provider string, withTokens bool, pagingQuery *common.PagingQuery) (users []*common.User, cursor *paginator.Cursor, err error) {
	if pagingQuery == nil {
		return nil, nil, fmt.Errorf("missing paging query")
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

	result, c, err := p.Paginate(stmt, &users)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return users, &c, err
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

// RemoveUserUploads deletes all uploads matching the user and token filters
func (b *Backend) RemoveUserUploads(userID string, tokenStr string) (removed int, err error) {
	deleted := 0
	var errors []error
	f := func(upload *common.Upload) (err error) {
		err = b.RemoveUpload(upload.ID)
		if err != nil {
			// TODO LOG
			errors = append(errors, err)
			return nil
		}
		deleted++
		return nil
	}

	err = b.ForEachUserUploads(userID, tokenStr, f)
	if err != nil {
		return deleted, err
	}
	if len(errors) > 0 {
		return deleted, fmt.Errorf("unable to delete all user uploads")
	}

	return deleted, nil
}

// DeleteUser delete a user from the DB
func (b *Backend) DeleteUser(userID string) (deleted bool, err error) {
	_, err = b.RemoveUserUploads(userID, "")
	if err != nil {
		return false, err
	}

	err = b.db.Transaction(func(tx *gorm.DB) (err error) {
		// Delete user tokens
		err = tx.Where(&common.Token{UserID: userID}).Delete(&common.Token{}).Error
		if err != nil {
			return fmt.Errorf("unable to delete tokens metadata : %s", err)
		}

		// Delete user
		result := tx.Where(&common.User{ID: userID}).Delete(common.User{})
		if result.Error != nil {
			return fmt.Errorf("unable to delete user metadata : %s", result.Error)
		}

		if result.RowsAffected > 0 {
			deleted = true
		}

		return nil
	})

	return deleted, err
}

// CountUsers count the number of user in the DB
func (b *Backend) CountUsers() (count int, err error) {
	var c int64 // Gorm V2 needs int64 for counts
	err = b.db.Model(&common.User{}).Count(&c).Error
	if err != nil {
		return -1, err
	}

	return int(c), nil
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

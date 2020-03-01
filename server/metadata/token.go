package metadata

import (
	"fmt"

	paginator "github.com/pilagod/gorm-cursor-paginator"

	"github.com/jinzhu/gorm"

	"github.com/root-gg/plik/server/common"
)

// CreateToken create a new token in DB
func (b *Backend) CreateToken(token *common.Token) (err error) {
	return b.db.Create(token).Error
}

// GetToken return a token from the DB ( return nil and non error if not found )
func (b *Backend) GetToken(tokenStr string) (token *common.Token, err error) {
	token = &common.Token{}
	err = b.db.Where(&common.Token{Token: tokenStr}).Take(token).Error
	if gorm.IsRecordNotFoundError(err) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}

	return token, err
}

// GetTokens return all tokens for a user
func (b *Backend) GetTokens(userID string, pagingQuery *common.PagingQuery) (tokens []*common.Token, cursor *paginator.Cursor, err error) {
	stmt := b.db.Model(&common.Token{}).Where(&common.Token{UserID: userID})

	p := pagingQuery.Paginator()
	p.SetKeys("CreatedAt", "Token")

	err = p.Paginate(stmt, &tokens).Error
	if err != nil {
		return nil, nil, err
	}

	c := p.GetNextCursor()
	return tokens, &c, err
}

// DeleteToken remove a token from the DB
func (b *Backend) DeleteToken(tokenStr string) (deleted bool, err error) {

	// Delete token
	result := b.db.Delete(&common.Token{Token: tokenStr})
	if result.Error != nil {
		return false, fmt.Errorf("unable to delete token metadata")
	}

	return result.RowsAffected > 0, err
}

// CountUserTokens count how many token a user has
func (b *Backend) CountUserTokens(userID string) (count int, err error) {
	err = b.db.Model(&common.Token{}).Where(&common.Token{UserID: userID}).Count(&count).Error
	if err != nil {
		return -1, err
	}

	return count, nil
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

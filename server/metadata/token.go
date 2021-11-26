package metadata

import (
	"fmt"

	"github.com/pilagod/gorm-cursor-paginator/v2/paginator"
	"gorm.io/gorm"

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
	if err == gorm.ErrRecordNotFound {
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

	result, c, err := p.Paginate(stmt, &tokens)
	if err != nil {
		return nil, nil, err
	}
	if result.Error != nil {
		return nil, nil, result.Error
	}

	return tokens, &c, err
}

// DeleteToken remove a token from the DB
func (b *Backend) DeleteToken(tokenStr string) (deleted bool, err error) {

	// Delete token
	result := b.db.Delete(&common.Token{Token: tokenStr})
	if result.Error != nil {
		return false, fmt.Errorf("unable to delete token metadata : %s", result.Error)
	}

	return result.RowsAffected > 0, err
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

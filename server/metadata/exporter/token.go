package exporter

import (
	"github.com/root-gg/plik/server/common"
	"time"
)

// Token metadata in 1.3 format
type Token struct {
	Token   string
	Comment string

	UserID string

	CreatedAt time.Time
}

// AdaptToken for 1.3 metadata format
func AdaptToken(u *User, token *common.Token) (t *Token, err error) {
	t = &Token{}
	t.Token = token.Token
	t.Comment = token.Comment
	t.UserID = u.ID
	t.CreatedAt = u.CreatedAt
	return t, nil
}

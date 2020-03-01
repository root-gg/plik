package exporter

import (
	"github.com/root-gg/plik/server/common"
	"strings"
	"time"
)

// User metadata in 1.3 format
type User struct {
	ID       string
	Provider string
	Login    string
	Password string
	Name     string
	Email    string
	IsAdmin  bool

	Tokens []*Token

	CreatedAt time.Time
}

// AdaptUser for 1.3 metadata format
func AdaptUser(user *common.User) (u *User, err error) {
	u = &User{}
	u.ID = user.ID
	u.Provider = strings.Split(u.ID, ":")[0]
	u.Login = user.Login
	u.Name = user.Name
	u.Email = user.Email
	u.CreatedAt = time.Now()

	// Adapt tokens
	for _, token := range user.Tokens {
		t, err := AdaptToken(u, token)
		if err != nil {
			return nil, err
		}
		u.Tokens = append(u.Tokens, t)
	}

	return u, nil
}

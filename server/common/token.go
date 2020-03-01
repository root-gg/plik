package common

import (
	"fmt"
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

// Token provide a very basic authentication mechanism
type Token struct {
	Token   string `json:"token" gorm:"primary_key"`
	Comment string `json:"comment,omitempty"`

	UserID string `json:"-" gorm:"type:varchar(255) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE CASCADE"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewToken create a new Token instance
func NewToken() (t *Token) {
	t = &Token{}
	t.Initialize()
	return t
}

// Initialize generate the token uuid and sets the creation date
func (t *Token) Initialize() {
	token, err := uuid.NewV4()
	if err != nil {
		panic(fmt.Errorf("unable to generate token uuid %s", err))
	}
	t.Token = token.String()
}

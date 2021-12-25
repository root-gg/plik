package common

import (
	"fmt"
	"time"
)

// ProviderGoogle for authentication
const ProviderGoogle = "google"

// ProviderOVH for authentication
const ProviderOVH = "ovh"

// ProviderLocal for authentication
const ProviderLocal = "local"

// User is a Plik user
type User struct {
	ID       string `json:"id,omitempty"`
	Provider string `json:"provider"`
	Login    string `json:"login,omitempty"`
	Password string `json:"-"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
	IsAdmin  bool   `json:"admin"`

	MaxFileSize int64 `json:"maxFileSize"`
	MaxTTL      int   `json:"maxTTL"`

	Tokens []*Token `json:"tokens,omitempty"`

	CreatedAt time.Time `json:"createdAt"`
}

// NewUser create a new user object
func NewUser(provider string, providerID string) (user *User) {
	user = &User{}
	user.ID = GetUserID(provider, providerID)
	user.Provider = provider
	return user
}

// GetUserID return user ID from provider and login
func GetUserID(provider string, providerID string) string {
	return fmt.Sprintf("%s:%s", provider, providerID)
}

// IsValidProvider return true if the provider string is valid
func IsValidProvider(provider string) bool {
	switch provider {
	case ProviderLocal, ProviderGoogle, ProviderOVH:
		return true
	default:
		return false
	}
}

// NewToken add a new token to a user
func (user *User) NewToken() (token *Token) {
	token = NewToken()
	token.UserID = user.ID
	user.Tokens = append(user.Tokens, token)
	return token
}

// NewToken add a new token to a user
func (user *User) String() string {
	str := user.Provider + ":" + user.Login
	if user.Name != "" {
		str += " " + user.Name
	}
	if user.Email != "" {
		str += " " + user.Email
	}
	return str
}

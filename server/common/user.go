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
	MaxUserSize int64 `json:"maxUserSize"`
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

// CreateUserFromParams return a user object ready to be inserted in the metadata backend
func CreateUserFromParams(userParams *User) (user *User, err error) {
	if !IsValidProvider(userParams.Provider) {
		return nil, fmt.Errorf("invalid provider")
	}

	if len(userParams.Login) < 4 {
		return nil, fmt.Errorf("login is too short (min 4 chars)")
	}

	user = NewUser(userParams.Provider, userParams.Login)
	user.Login = userParams.Login
	user.Name = userParams.Name
	user.Email = userParams.Email
	user.IsAdmin = userParams.IsAdmin
	user.MaxFileSize = userParams.MaxFileSize
	user.MaxUserSize = userParams.MaxUserSize
	user.MaxTTL = userParams.MaxTTL

	if user.Provider == ProviderLocal {
		if len(userParams.Password) < 8 {
			return nil, fmt.Errorf("password is too short (min 8 chars)")
		}

		hash, err := HashPassword(userParams.Password)
		if err != nil {
			return nil, fmt.Errorf("unable to hash password : %s", err)
		}
		user.Password = hash
	}

	return user, nil
}

// UpdateUser update a user object with the params
//   - prevent to update provider, user ID or login
//   - only update password if a new one is provided
func UpdateUser(user *User, userParams *User) (err error) {
	if user.Provider == ProviderLocal && len(userParams.Password) > 0 {
		if len(userParams.Password) < 8 {
			return fmt.Errorf("password is too short (min 8 chars)")
		}
		hash, err := HashPassword(userParams.Password)
		if err != nil {
			return fmt.Errorf("unable to hash password : %s", err)
		}
		user.Password = hash
	}

	user.Name = userParams.Name
	user.Email = userParams.Email
	user.IsAdmin = userParams.IsAdmin
	user.MaxFileSize = userParams.MaxFileSize
	user.MaxUserSize = userParams.MaxUserSize
	user.MaxTTL = userParams.MaxTTL
	return nil
}

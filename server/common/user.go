package common

import (
	"fmt"
	"regexp"
	"time"
)

// ProviderGoogle for authentication
const ProviderGoogle = "google"

// ProviderOVH for authentication
const ProviderOVH = "ovh"

// ProviderLocal for authentication
const ProviderLocal = "local"

// User is a plik user
type User struct {
	ID               string `json:"id,omitempty"`
	Provider         string `json:"provider"`
	Login            string `json:"login,omitempty"`
	Password         string `json:"-"`
	Name             string `json:"name,omitempty"`
	Email            string `json:"email,omitempty"`
	IsAdmin          bool   `json:"admin"`
	VerificationCode string `json:"-"`
	Verified         bool   `json:"verified"`

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

// NewInvite create a new invite from a user
func (user *User) NewInvite(validity time.Duration) (invite *Invite, err error) {
	return NewInvite(user, validity)
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

var loginRegexp = regexp.MustCompile("^[\\w\\d._@-]{3,50}$")
var nameRegexp = regexp.MustCompile("^[\\p{L}\\d._@ -]{3,50}$")

// PrepareInsert user for database insert ( check configuration and default values, ...)
func (user *User) PrepareInsert(config *Configuration) (err error) {
	if !IsValidProvider(user.Provider) {
		return fmt.Errorf("invalid provider")
	}

	if !loginRegexp.MatchString(user.Login) {
		return fmt.Errorf("invalid login : 3 to 50 alphanumerical and the following characters ._-@")
	}

	if !nameRegexp.MatchString(user.Name) {
		return fmt.Errorf("invalid name : 3 to 50 utf8 characters, space and the following characters ._-@")
	}

	err = config.CheckEmail(user.Email)
	if err != nil {
		return err
	}

	if !config.EmailVerification {
		user.Verified = true
	}

	if user.Provider == ProviderLocal {
		if len(user.Password) < 4 || len(user.Password) > 50 {
			return fmt.Errorf("password should be 4 to 50 characters long")
		}

		// Hash password
		user.Password, err = HashPassword(user.Password)
		if err != nil {
			return fmt.Errorf("unable to hash password : %s", err)
		}
	}

	return nil
}

// GenVerificationCode generate a random verification code
func (user *User) GenVerificationCode() {
	user.VerificationCode = GenerateRandomID(32)
}

// GetVerifyURL return the url to follow to verify the user
func (user *User) GetVerifyURL(config *Configuration) string {
	return fmt.Sprintf("%s/auth/local/verify/%s/%s", config.GetServerURL().String(), user.Login, user.VerificationCode)
}

package common

import (
	"fmt"
	"time"

	uuid "github.com/nu7hatch/gouuid"
)

// Invite a user to create an account
type Invite struct {
	ID     string  `json:"id"`
	Issuer *string `json:"-" gorm:"type:varchar(255) REFERENCES users(id) ON UPDATE RESTRICT ON DELETE CASCADE;index:idx_invite_issuer"`

	Email string `json:"email"`

	Admin    bool `json:"admin"`
	Verified bool `json:"verified"`

	ExpireAt  *time.Time `json:"expireAt" gorm:"index:idx_invite_expire_at"`
	CreatedAt time.Time  `json:"createdAt"`
}

// NewInvite create a new invite object
func NewInvite(issuer *User, validity time.Duration) (invite *Invite, err error) {
	invite = &Invite{}
	uid, err := uuid.NewV4()
	if err != nil {
		return nil, fmt.Errorf("Unable to create uuid : %s", err)
	}
	invite.ID = uid.String()

	if issuer != nil {
		invite.Issuer = &issuer.ID
	}

	if validity > 0 {
		deadline := time.Now().Add(validity)
		invite.ExpireAt = &deadline
	}

	return invite, nil
}

// PrepareInsert user for database insert ( check configuration and default values, ...)
func (invite *Invite) PrepareInsert(config *Configuration) (err error) {
	if invite.Email != "" && !IsValidEmail(invite.Email) {
		return fmt.Errorf("invalid email")
	}
	return nil
}

// HasExpired Check if invite has expired
func (invite *Invite) HasExpired() bool {
	if invite.ExpireAt == nil {
		return false
	}
	return time.Now().After(*invite.ExpireAt)
}

// String return a string representation of the invite
func (invite *Invite) String() string {
	str := invite.ID
	if invite.Admin {
		str += " (admin)"
	}

	if invite.Issuer != nil {
		str += " from " + *invite.Issuer
	}

	if invite.HasExpired() {
		str += " is expired"
	} else if invite.ExpireAt != nil {
		str += fmt.Sprintf(" expire in %s", invite.ExpireAt.Sub(time.Now()))
	}
	return str
}

// GetURL return the link to follow to use the invite
func (invite *Invite) GetURL(config *Configuration) string {
	return fmt.Sprintf("%s/#/register?invite=%s?email=%s", config.GetServerURL(), invite.ID, invite.Email)
}

package common

import (
	"fmt"
	"regexp"
	"strings"
)

var emailRegexp = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

// IsValidEmail check if the email looks valid
func IsValidEmail(email string) bool {
	return emailRegexp.MatchString(email)
}

// CheckEmail check if the email looks valid and is allowed by the configuration
func (config *Configuration) CheckEmail(email string) (err error) {
	if !IsValidEmail(email) {
		return fmt.Errorf("invalid email")
	}

	if len(config.EmailValidDomains) > 0 || len(config.GoogleValidDomains) > 0 {
		// Check email domain
		var validDomains []string
		validDomains = append(validDomains, config.EmailValidDomains...)
		validDomains = append(validDomains, config.GoogleValidDomains...)

		components := strings.Split(email, "@")
		ok := false
		for _, validDomain := range validDomains {
			if strings.Compare(components[1], validDomain) == 0 {
				ok = true
				break
			}
		}
		if !ok {
			return fmt.Errorf("invalid email domain")
		}
	}

	return nil
}

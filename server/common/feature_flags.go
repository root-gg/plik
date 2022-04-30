package common

import (
	"fmt"
	"github.com/root-gg/utils"
)

// FeatureDisabled feature is always off
const FeatureDisabled = "disabled"

// FeatureEnabled feature is opt-in
const FeatureEnabled = "enabled"

// FeatureDefault feature is opt-out
const FeatureDefault = "default"

// FeatureForced feature is always on
const FeatureForced = "forced"

// ValidateFeatureFlag validates a feature flag string value
func ValidateFeatureFlag(value string) (err error) {
	possibleValues := []string{FeatureDisabled, FeatureEnabled, FeatureDefault, FeatureForced}
	for _, possibleValue := range possibleValues {
		if value == possibleValue {
			return nil
		}
	}
	return fmt.Errorf("Invalid feature flag value. Expecting : %s|%s|%s|%s", utils.ToInterfaceArray(possibleValues)...)
}

// IsFeatureAvailable return true is the feature is available
func IsFeatureAvailable(value string) bool {
	return value == FeatureEnabled || value == FeatureDefault || value == FeatureForced
}

// IsFeatureDefault return true is the feature is enabled by default
func IsFeatureDefault(value string) bool {
	return value == FeatureDefault || value == FeatureForced
}

func (config *Configuration) initializeFeatureFlags() error {
	initializations := []func() error{
		config.initializeFeatureAuthentication,
		config.initializeFeatureOneShot,
		config.initializeFeatureRemovable,
		config.initializeFeatureStream,
		config.initializeFeaturePassword,
	}

	for _, initialization := range initializations {
		err := initialization()
		if err != nil {
			return err
		}
	}

	return nil
}

func (config *Configuration) initializeFeatureAuthentication() error {
	if config.FeatureAuthentication == "" {
		// Use legacy feature flags
		if config.NoAnonymousUploads {
			config.FeatureAuthentication = FeatureForced
		} else {
			if config.Authentication {
				config.FeatureAuthentication = FeatureEnabled
			} else {
				config.FeatureAuthentication = FeatureDisabled
			}
		}
	}

	err := ValidateFeatureFlag(config.FeatureAuthentication)
	if err != nil {
		return fmt.Errorf("Invalid value for FeatureAuthentication : %s", err)
	}

	// Set legacy feature flag for backward compatibility
	config.Authentication = IsFeatureAvailable(config.FeatureAuthentication)
	config.NoAnonymousUploads = config.FeatureAuthentication == FeatureForced

	return nil
}

func (config *Configuration) initializeFeatureOneShot() error {
	if config.FeatureOneShot == "" {
		// Use legacy feature flags
		if config.OneShot {
			config.FeatureOneShot = FeatureEnabled
		} else {
			config.FeatureOneShot = FeatureDisabled
		}
	}

	err := ValidateFeatureFlag(config.FeatureOneShot)
	if err != nil {
		return fmt.Errorf("Invalid value for FeatureOneShot : %s", err)
	}

	// Set legacy feature flag for backward compatibility
	config.OneShot = IsFeatureAvailable(config.FeatureOneShot)

	return nil
}

func (config *Configuration) initializeFeatureRemovable() error {
	if config.FeatureRemovable == "" {
		// Use legacy feature flags
		if config.Removable {
			config.FeatureRemovable = FeatureEnabled
		} else {
			config.FeatureRemovable = FeatureDisabled
		}
	}

	err := ValidateFeatureFlag(config.FeatureRemovable)
	if err != nil {
		return fmt.Errorf("Invalid value for FeatureRemovable : %s", err)
	}

	// Set legacy feature flag for backward compatibility
	config.Removable = IsFeatureAvailable(config.FeatureRemovable)

	return nil
}

func (config *Configuration) initializeFeatureStream() error {
	if config.FeatureStream == "" {
		// Use legacy feature flags
		if config.Stream {
			config.FeatureStream = FeatureEnabled
		} else {
			config.FeatureStream = FeatureDisabled
		}
	}

	err := ValidateFeatureFlag(config.FeatureStream)
	if err != nil {
		return fmt.Errorf("Invalid value for FeatureStream : %s", err)
	}

	// Set legacy feature flag for backward compatibility
	config.Stream = IsFeatureAvailable(config.FeatureStream)

	return nil
}

func (config *Configuration) initializeFeaturePassword() error {
	if config.FeaturePassword == "" {
		// Use legacy feature flags
		if config.ProtectedByPassword {
			config.FeaturePassword = FeatureEnabled
		} else {
			config.FeaturePassword = FeatureDisabled
		}
	}

	err := ValidateFeatureFlag(config.FeaturePassword)
	if err != nil {
		return fmt.Errorf("Invalid value for FeaturePassword : %s", err)
	}

	// Set legacy feature flag for backward compatibility
	config.ProtectedByPassword = IsFeatureAvailable(config.FeaturePassword)

	return nil
}

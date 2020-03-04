package common

// AuthenticationSignatureKeySettingKey setting key for authentication_signature_key
const AuthenticationSignatureKeySettingKey = "authentication_signature_key"

// Setting is a config object meant to be shard by all Plik instances using the metadata backend
type Setting struct {
	Key   string `gorm:"primary_key"`
	Value string
}

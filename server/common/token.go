package common

import (
	"hash/crc32"
	"strings"
	"time"
)

const (
	// TokenPrefix is the fixed prefix for all new Plik CLI tokens.
	// Enables secret scanners to identify leaked tokens and improves human recognition.
	TokenPrefix = "plik_"

	// TokenRandomLength is the number of Base62 random characters in the token body.
	// 30 chars of Base62 ≈ 178 bits of entropy.
	TokenRandomLength = 30

	// TokenChecksumLength is the number of Base62 characters used for the CRC32 checksum.
	// CRC32 produces a 32-bit value; 6 Base62 chars can represent up to ~47 bits (62^6 ≈ 5.7×10^10 > 2^32).
	TokenChecksumLength = 6

	// TokenTotalLength is the total length of a prefixed token:
	// len("plik_") + 30 + 6 = 41
	TokenTotalLength = len(TokenPrefix) + TokenRandomLength + TokenChecksumLength
)

// Token provide a very basic authentication mechanism
type Token struct {
	Token   string `json:"token" gorm:"primary_key"`
	Comment string `json:"comment,omitempty"`

	UserID string `json:"-" gorm:"size:256;constraint:OnUpdate:RESTRICT,OnDelete:RESTRICT;"`

	// Stats wraps the canonical nested usage payload for this token in the same
	// usage{} envelope as the server and user scopes, so a client always reads
	// "...usage.current" / "...usage.lifetime" regardless of scope. The single
	// canonical location for the token's last-upload timestamp is
	// Stats.Usage.LastUploadAt (there is no separate token-level field).
	Stats *TokenStats `json:"stats,omitempty" gorm:"-"`

	CreatedAt time.Time `json:"createdAt"`
}

// TokenStats is the envelope wrapping a token's usage payload, matching the
// shape of ServerStats.Usage and UserStats.Usage so the "usage.current" /
// "usage.lifetime" suffix is identical across all three API scopes.
type TokenStats struct {
	Usage *UsageStatsResponse `json:"usage,omitempty"`
}

// NewToken create a new Token instance
func NewToken() (t *Token) {
	t = &Token{}
	t.Initialize()
	return t
}

// Initialize generate the prefixed token string and sets the creation date
func (t *Token) Initialize() {
	t.Token = GeneratePrefixedToken()
}

// GeneratePrefixedToken creates a new token in the format: plik_<30 Base62 random chars><6 Base62 CRC32 checksum>
func GeneratePrefixedToken() string {
	// Generate the random body
	body := GenerateRandomID(TokenRandomLength)

	// Build the prefix + body string to checksum
	payload := TokenPrefix + body

	// Compute CRC32 checksum of the payload
	checksum := crc32.ChecksumIEEE([]byte(payload))

	// Encode CRC32 as Base62 (zero-padded to TokenChecksumLength chars)
	checksumStr := encodeBase62(checksum, TokenChecksumLength)

	return payload + checksumStr
}

// ValidateTokenChecksum checks if a token has a valid plik_ prefix and matching CRC32 checksum.
// Returns false for legacy UUIDv4 tokens (no prefix) without error.
func ValidateTokenChecksum(token string) bool {
	if !strings.HasPrefix(token, TokenPrefix) {
		return false
	}

	if len(token) != TokenTotalLength {
		return false
	}

	// Split into payload (prefix + random body) and checksum
	payload := token[:len(TokenPrefix)+TokenRandomLength]
	checksumStr := token[len(TokenPrefix)+TokenRandomLength:]

	// Recompute and compare
	expected := crc32.ChecksumIEEE([]byte(payload))
	expectedStr := encodeBase62(expected, TokenChecksumLength)

	return checksumStr == expectedStr
}

// encodeBase62 encodes a uint32 value as a Base62 string, zero-padded to the given length.
func encodeBase62(value uint32, length int) string {
	v := uint64(value)
	base := uint64(len(Base62Charset))

	result := make([]byte, length)
	for i := length - 1; i >= 0; i-- {
		result[i] = Base62Charset[v%base]
		v /= base
	}

	return string(result)
}

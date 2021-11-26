package cursor

import "errors"

// Errors for encoder
var (
	ErrInvalidCursor = errors.New("invalid cursor")
	ErrInvalidModel  = errors.New("invalid model")
)

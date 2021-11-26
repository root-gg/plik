package paginator

import "errors"

// Errors for paginator
var (
	ErrInvalidCursor = errors.New("invalid cursor for paginating")
	ErrInvalidLimit  = errors.New("limit should be greater than 0")
	ErrInvalidModel  = errors.New("model fields should match rules or keys specified for paginator")
	ErrInvalidOrder  = errors.New("order should be ASC or DESC")
	ErrNoRule        = errors.New("paginator should have at least one rule")
)

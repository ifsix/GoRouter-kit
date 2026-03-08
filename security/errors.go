package security

import "errors"

var (
	ErrAccessDenied      = errors.New("access denied")
	ErrAuthRequired      = errors.New("authentication required")
	ErrRateLimited       = errors.New("rate limit exceeded")
	ErrInvalidToken      = errors.New("invalid token")
	ErrTokenCreate       = errors.New("token creation failed")
	ErrDangerousArgument = errors.New("dangerous argument detected")
	ErrAuthConfig        = errors.New("invalid authentication config")
)

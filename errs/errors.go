package errs

import "errors"

var (
	ErrBadRequest = errors.New("bad request")
	ErrDecode     = errors.New("decode response failed")
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	if e == nil {
		return "api error"
	}
	if e.Message == "" {
		return "openrouter request failed"
	}
	return e.Message
}

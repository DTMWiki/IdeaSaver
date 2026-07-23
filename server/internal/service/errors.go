package service

import "errors"

var (
	// ErrPermission indicates the caller is not allowed to perform the action.
	ErrPermission = errors.New("permission denied")
	// ErrNotFound indicates a resource does not exist.
	ErrNotFound = errors.New("not found")
)

// ClientError is a safe, user-facing business error (may be returned to clients as-is).
type ClientError struct {
	Msg string
}

func (e *ClientError) Error() string {
	if e == nil {
		return "client error"
	}
	return e.Msg
}

// ClientErr wraps a Chinese business message for API clients.
func ClientErr(msg string) error {
	return &ClientError{Msg: msg}
}

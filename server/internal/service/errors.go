package service

import "errors"

var (
	// ErrPermission indicates the caller is not allowed to perform the action.
	ErrPermission = errors.New("permission denied")
	// ErrNotFound indicates a resource does not exist.
	ErrNotFound = errors.New("not found")
)

package update

import "errors"

var (
	ErrInvalidInput  = errors.New("invalid input")
	ErrNotFound      = errors.New("service not found")
	ErrPersistFailed = errors.New("failed to persist update")
	ErrInternalError = errors.New("internal error")
)

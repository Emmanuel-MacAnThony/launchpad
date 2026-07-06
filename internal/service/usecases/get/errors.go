package get

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrNotFound     = errors.New("service not found")
	ErrInternalError = errors.New("internal error")
)

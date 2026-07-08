package create

import "errors"

var (
	ErrValidation      = errors.New("invalid input")
	ErrServiceNotFound = errors.New("service not found")
	ErrInternal        = errors.New("internal error")
)

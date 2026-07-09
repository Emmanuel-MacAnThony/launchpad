package listdeploys

import "errors"

var (
	ErrValidation = errors.New("invalid input")
	ErrInternal   = errors.New("internal error")
)

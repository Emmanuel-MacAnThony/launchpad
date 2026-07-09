package getdeploy

import "errors"

var (
	ErrValidation = errors.New("invalid input")
	ErrNotFound   = errors.New("deploy not found")
	ErrInternal   = errors.New("internal error")
)

package refreshlock

import "errors"

var (
	ErrValidation    = errors.New("invalid input")
	ErrDeployNotFound = errors.New("deploy not found")
	ErrInvalidState  = errors.New("deploy is not building — no lock to refresh")
	ErrInternal      = errors.New("internal error")
)

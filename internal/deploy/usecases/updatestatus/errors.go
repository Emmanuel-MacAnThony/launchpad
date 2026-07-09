package updatestatus

import "errors"

var (
	ErrValidation        = errors.New("invalid input")
	ErrDeployNotFound    = errors.New("deploy not found")
	ErrInvalidTransition = errors.New("invalid status transition")
	ErrInternal          = errors.New("internal error")
)

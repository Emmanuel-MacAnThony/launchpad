package activate

import "errors"

var (
	ErrValidation     = errors.New("invalid input")
	ErrDeployNotFound = errors.New("deploy not found")
	ErrInvalidState   = errors.New("deploy is not in building state")
	ErrNginxFailed    = errors.New("nginx operation failed")
	ErrInternal       = errors.New("internal error")
)

package rollback

import "errors"

var (
	ErrValidation           = errors.New("invalid input")
	ErrServiceNotFound      = errors.New("service not found")
	ErrNoActiveDeployment   = errors.New("no active deployment to roll back")
	ErrNoPreviousDeployment = errors.New("no previous deployment on inactive slot")
	ErrSSHFailed            = errors.New("failed to connect to host")
	ErrNginxFailed          = errors.New("nginx operation failed")
	ErrInternal             = errors.New("internal error")
)

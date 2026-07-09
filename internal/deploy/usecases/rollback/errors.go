package rollback

import "errors"

var (
	ErrValidation           = errors.New("invalid input")
	ErrServiceNotFound      = errors.New("service not found")
	ErrNoActiveDeployment   = errors.New("no active deployment to roll back")
	ErrNoPreviousDeployment = errors.New("no previous deployment on inactive slot")
	ErrNginxFailed          = errors.New("nginx switch failed")
	ErrInternal             = errors.New("internal error")
)

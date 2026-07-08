package domain

import "errors"

var (
	ErrNotFound        = errors.New("deploy not found")
	ErrServiceNotFound = errors.New("service not found")
)

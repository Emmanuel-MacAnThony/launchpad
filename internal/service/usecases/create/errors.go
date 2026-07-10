package create

import "errors"

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrDomainTaken        = errors.New("domain already taken")
	ErrSSHFailed          = errors.New("failed to connect to host")
	ErrDockerNotInstalled = errors.New("docker is not installed on host")
	ErrNginxNotInstalled  = errors.New("nginx is not installed on host")
	ErrBootstrapFailed    = errors.New("failed to bootstrap nginx on host")
	ErrPortConflict       = errors.New("port already in use on host")
	ErrPortScanFailed     = errors.New("failed to check ports on host")
	ErrPersistFailed      = errors.New("failed to persist service")

	// ErrIDConflict means we generated a UUID that already exists in the DB.
	// Astronomically rare but theoretically possible. Surfaced as ErrPersistFailed
	// so callers treat it as a transient error and retry (new UUID generated each time).
	ErrIDConflict = errors.New("id conflict after max retries")
)

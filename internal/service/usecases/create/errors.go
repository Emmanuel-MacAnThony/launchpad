package create

import "errors"

var (
	ErrInvalidInput      = errors.New("invalid input")
	ErrDomainTaken       = errors.New("domain already taken")
	ErrPortConflict      = errors.New("port already in use on host")
	ErrPortScanFailed    = errors.New("failed to check ports on host")
	ErrPersistFailed     = errors.New("failed to persist service")
	ErrNginxConfigFailed = errors.New("failed to write nginx config")
	ErrNginxReloadFailed = errors.New("failed to reload nginx")

	// ErrIDConflict means we generated a UUID that already exists in the DB.
	// This is astronomically rare but theoretically possible.
	// It is not the client's fault — they cannot control UUID generation.
	// We surface it as ErrPersistFailed so the caller treats it as a transient
	// server error and can retry. A retry will generate a fresh UUID and almost
	// certainly succeed.
	ErrIDConflict = errors.New("id conflict after max retries")
)

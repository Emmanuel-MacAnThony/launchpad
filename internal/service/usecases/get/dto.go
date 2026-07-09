package get

import "time"

type GetInput struct {
	ID string
}

type GetOutput struct {
	ID             string
	Name           string
	RepoURL        string
	Domain         string
	HealthCheckURL string
	Host           string
	SSHUser        string
	SSHKeyPath     string
	WebhookSecret  string // decrypted; server-side only — never expose in HTTP responses
	CreatedAt      time.Time
}

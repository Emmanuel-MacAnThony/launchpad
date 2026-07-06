package create

import "time"

type CreateInput struct {
	Name           string
	RepoURL        string
	Domain         string
	HealthCheckURL string
	WebhookSecret  string
	Host           string
	SSHUser        string
	SSHKeyPath     string
}

type CreateOutput struct {
	ID             string
	Name           string
	RepoURL        string
	Domain         string
	HealthCheckURL string
	Host           string
	SSHUser        string
	SSHKeyPath     string
	WebhookURL     string
	CreatedAt      time.Time
}

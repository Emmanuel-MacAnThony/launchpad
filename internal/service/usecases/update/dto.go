package update

import "time"

type UpdateInput struct {
	ID             string
	Name           string
	HealthCheckURL string
}

type UpdateOutput struct {
	ID             string
	Name           string
	RepoURL        string
	Domain         string
	HealthCheckURL string
	Host           string
	SSHUser        string
	SSHKeyPath     string
	CreatedAt      time.Time
}

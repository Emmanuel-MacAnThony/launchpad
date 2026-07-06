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
	CreatedAt      time.Time
}

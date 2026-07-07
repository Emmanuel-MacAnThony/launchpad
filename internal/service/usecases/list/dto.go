package list

import "time"

type ListInput struct{}

type ListItem struct {
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

type ListOutput struct {
	Services []ListItem
}

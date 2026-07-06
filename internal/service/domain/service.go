package domain

import "time"

type Service struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	RepoURL        string    `json:"repo_url"`
	Domain         string    `json:"domain"`
	HealthCheckURL string    `json:"health_check_url"`
	WebhookSecret  string    `json:"-"`
	Host           string    `json:"host"`
	SSHUser        string    `json:"ssh_user"`
	SSHKeyPath     string    `json:"ssh_key_path"`
	CreatedAt      time.Time `json:"created_at"`
}

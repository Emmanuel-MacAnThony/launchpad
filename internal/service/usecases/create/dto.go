package create

import (
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

type CreateInput struct {
	Name           string
	RepoURL        string
	Domain         string
	HealthCheckURL string
	WebhookSecret  string
	Host           string
	SSHUser        string
	SSHKeyPath     string
	BluePort       int
	GreenPort      int
	ContainerPort  int
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
	BluePort       int
	GreenPort      int
	ContainerPort  int
	ActiveSlot     *domain.Slot // nil = never deployed
	CreatedAt      time.Time
}

package update

import (
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

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
	SSHKey     string
	ComposeSvc     string
	ActiveSlot     *domain.Slot
	CreatedAt      time.Time
}

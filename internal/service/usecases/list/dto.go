package list

import (
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

type ListInput struct{}

type ListItem struct {
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

type ListOutput struct {
	Services []ListItem
}

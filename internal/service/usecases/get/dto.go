package get

import (
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

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
	SSHKey     string
	WebhookSecret  string // decrypted; server-side only — never expose in HTTP responses
	BluePort       int
	GreenPort      int
	ContainerPort  int
	ComposeSvc     string
	ActiveSlot     *domain.Slot
	CreatedAt      time.Time
}

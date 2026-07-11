package domain

import "time"

type Slot string

const (
	SlotBlue  Slot = "blue"
	SlotGreen Slot = "green"
)

type Service struct {
	ID             string    `json:"id"`
	Name           string    `json:"name"`
	RepoURL        string    `json:"repo_url"`
	Domain         string    `json:"domain"`
	HealthCheckURL string    `json:"health_check_url"`
	WebhookSecret  string    `json:"-"`
	Host           string    `json:"host"`
	SSHUser        string    `json:"ssh_user"`
	SSHKey         string    `json:"-"` // encrypted at rest; never returned in API responses
	BluePort       int       `json:"blue_port"`
	GreenPort      int       `json:"green_port"`
	ContainerPort  int       `json:"container_port"`
	ComposeSvc     string    `json:"compose_service"` // the service name in docker-compose.yml to override
	ActiveSlot     *Slot     `json:"active_slot"`     // nil = never deployed
	CreatedAt      time.Time `json:"created_at"`
}

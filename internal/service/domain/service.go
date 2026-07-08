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
	SSHKeyPath     string    `json:"ssh_key_path"`
	BluePort       int       `json:"blue_port"`
	GreenPort      int       `json:"green_port"`
	ContainerPort  int       `json:"container_port"`
	ActiveSlot     *Slot     `json:"active_slot"` // nil = never deployed
	CreatedAt      time.Time `json:"created_at"`
}

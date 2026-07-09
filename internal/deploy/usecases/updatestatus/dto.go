package updatestatus

import "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"

type UpdateStatusInput struct {
	DeployID  string
	NewStatus domain.DeployStatus
	Slot      *domain.Slot // required only when NewStatus = building
}

type UpdateStatusOutput struct {
	Deploy domain.Deploy
}

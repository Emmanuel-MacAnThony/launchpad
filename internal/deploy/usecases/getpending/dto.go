package getpending

import "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"

type GetPendingOutput struct {
	Deploys []domain.Deploy
}

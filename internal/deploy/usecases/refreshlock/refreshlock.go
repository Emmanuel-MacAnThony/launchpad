package refreshlock

import (
	"errors"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/updatestatus"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type DeployRepo interface {
	GetByID(deployID string) (deploydomain.Deploy, error)
}

type LockRepo interface {
	RefreshLock(deployID string, newExpiresAt time.Time) error
}

type RefreshLockInput struct {
	DeployID string
}

type UseCase struct {
	deployRepo DeployRepo
	lockRepo   LockRepo
}

func New(deployRepo DeployRepo, lockRepo LockRepo) *UseCase {
	return &UseCase{deployRepo: deployRepo, lockRepo: lockRepo}
}

func (uc *UseCase) Execute(input RefreshLockInput) result.Result[struct{}] {
	if input.DeployID == "" {
		return result.Fail[struct{}](ErrValidation)
	}

	deploy, err := uc.deployRepo.GetByID(input.DeployID)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrDeployNotFound)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	if deploy.Status != deploydomain.StatusBuilding {
		return result.Fail[struct{}](ErrInvalidState)
	}

	if err := uc.lockRepo.RefreshLock(input.DeployID, time.Now().UTC().Add(updatestatus.LockDuration)); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	return result.Ok(struct{}{})
}

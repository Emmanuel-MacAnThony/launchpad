package updatestatus

import (
	"errors"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

// LockDuration is how long the deploy lock lives before the dead man's switch triggers.
// The agent worker must call RefreshLock before this elapses while the build is running.
const LockDuration = 10 * time.Minute

type DeployRepo interface {
	GetByID(deployID string) (deploydomain.Deploy, error)
	SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error
}

type LockRepo interface {
	CreateLock(deployID string, expiresAt time.Time) error
	ReleaseLock(deployID string) error
}

type UseCase struct {
	deployRepo DeployRepo
	lockRepo   LockRepo
}

func New(deployRepo DeployRepo, lockRepo LockRepo) *UseCase {
	return &UseCase{deployRepo: deployRepo, lockRepo: lockRepo}
}

var knownStatuses = map[deploydomain.DeployStatus]bool{
	deploydomain.StatusPending:    true,
	deploydomain.StatusBuilding:   true,
	deploydomain.StatusActive:     true,
	deploydomain.StatusFailed:     true,
	deploydomain.StatusRolledBack: true,
}

var validTransitions = map[deploydomain.DeployStatus]map[deploydomain.DeployStatus]bool{
	deploydomain.StatusPending:  {deploydomain.StatusBuilding: true},
	deploydomain.StatusBuilding: {deploydomain.StatusActive: true, deploydomain.StatusFailed: true},
	deploydomain.StatusActive:   {deploydomain.StatusRolledBack: true},
}

func (uc *UseCase) Execute(input UpdateStatusInput) result.Result[UpdateStatusOutput] {
	if input.DeployID == "" {
		return result.Fail[UpdateStatusOutput](ErrValidation)
	}
	if !knownStatuses[input.NewStatus] {
		return result.Fail[UpdateStatusOutput](ErrValidation)
	}
	if input.NewStatus == deploydomain.StatusBuilding && input.Slot == nil {
		return result.Fail[UpdateStatusOutput](ErrValidation)
	}

	deploy, err := uc.deployRepo.GetByID(input.DeployID)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[UpdateStatusOutput](ErrDeployNotFound)
		}
		return result.Fail[UpdateStatusOutput](ErrInternal)
	}

	if !validTransitions[deploy.Status][input.NewStatus] {
		return result.Fail[UpdateStatusOutput](ErrInvalidTransition)
	}

	if err := uc.deployRepo.SetStatus(input.DeployID, input.NewStatus, input.Slot); err != nil {
		return result.Fail[UpdateStatusOutput](ErrInternal)
	}

	now := time.Now().UTC()
	deploy.Status = input.NewStatus

	switch input.NewStatus {
	case deploydomain.StatusBuilding:
		deploy.Slot = input.Slot
		deploy.StartedAt = &now
		if err := uc.lockRepo.CreateLock(input.DeployID, now.Add(LockDuration)); err != nil {
			return result.Fail[UpdateStatusOutput](ErrInternal)
		}

	case deploydomain.StatusActive, deploydomain.StatusFailed:
		deploy.FinishedAt = &now
		if err := uc.lockRepo.ReleaseLock(input.DeployID); err != nil {
			return result.Fail[UpdateStatusOutput](ErrInternal)
		}

	case deploydomain.StatusRolledBack:
		deploy.FinishedAt = &now
		// no lock to release — deploy was active, never held a build lock
	}

	return result.Ok(UpdateStatusOutput{Deploy: deploy})
}

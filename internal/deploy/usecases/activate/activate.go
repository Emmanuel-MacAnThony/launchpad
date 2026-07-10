package activate

import (
	"errors"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type NginxClient interface {
	Switch(serviceID string, slot deploydomain.Slot) error
	ReloadNginx() error
}

type ServiceRepo interface {
	UpdateActiveSlot(serviceID string, slot servicedomain.Slot) error
}

type DeployRepo interface {
	GetByID(deployID string) (deploydomain.Deploy, error)
	SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error
}

type LockRepo interface {
	ReleaseLock(deployID string) error
}

type ActivateInput struct {
	DeployID  string
	ServiceID string
	Slot      deploydomain.Slot
}

type UseCase struct {
	nginx       NginxClient
	serviceRepo ServiceRepo
	deployRepo  DeployRepo
	lockRepo    LockRepo
}

func New(nginx NginxClient, serviceRepo ServiceRepo, deployRepo DeployRepo, lockRepo LockRepo) *UseCase {
	return &UseCase{
		nginx:       nginx,
		serviceRepo: serviceRepo,
		deployRepo:  deployRepo,
		lockRepo:    lockRepo,
	}
}

func (uc *UseCase) Execute(input ActivateInput) result.Result[struct{}] {
	if input.DeployID == "" || input.ServiceID == "" || input.Slot == "" {
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

	if err := uc.nginx.Switch(input.ServiceID, input.Slot); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if err := uc.nginx.ReloadNginx(); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if err := uc.serviceRepo.UpdateActiveSlot(input.ServiceID, servicedomain.Slot(input.Slot)); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.deployRepo.SetStatus(input.DeployID, deploydomain.StatusActive, nil); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.lockRepo.ReleaseLock(input.DeployID); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	return result.Ok(struct{}{})
}

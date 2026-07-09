package rollback

import (
	"errors"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type ServiceRepo interface {
	GetByID(serviceID string) (servicedomain.Service, error)
	UpdateActiveSlot(serviceID string, slot servicedomain.Slot) error
}

type DeployRepo interface {
	GetActiveForService(serviceID string) (deploydomain.Deploy, error)
	GetLatestOnSlot(serviceID string, slot deploydomain.Slot) (deploydomain.Deploy, error)
	SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error
}

type NginxClient interface {
	Switch(serviceID string, slot deploydomain.Slot) error
	ReloadNginx() error
}

type RollbackInput struct {
	ServiceID string
}

type UseCase struct {
	serviceRepo ServiceRepo
	deployRepo  DeployRepo
	nginx       NginxClient
}

func New(serviceRepo ServiceRepo, deployRepo DeployRepo, nginx NginxClient) *UseCase {
	return &UseCase{serviceRepo: serviceRepo, deployRepo: deployRepo, nginx: nginx}
}

func (uc *UseCase) Execute(input RollbackInput) result.Result[struct{}] {
	if input.ServiceID == "" {
		return result.Fail[struct{}](ErrValidation)
	}

	svc, err := uc.serviceRepo.GetByID(input.ServiceID)
	if err != nil {
		if errors.Is(err, servicedomain.ErrNotFound) {
			return result.Fail[struct{}](ErrServiceNotFound)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	if svc.ActiveSlot == nil {
		return result.Fail[struct{}](ErrNoActiveDeployment)
	}

	currentActive, err := uc.deployRepo.GetActiveForService(input.ServiceID)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrNoActiveDeployment)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	inactiveSlot := inactive(*svc.ActiveSlot)

	_, err = uc.deployRepo.GetLatestOnSlot(input.ServiceID, inactiveSlot)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[struct{}](ErrNoPreviousDeployment)
		}
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.nginx.Switch(input.ServiceID, inactiveSlot); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	if err := uc.nginx.ReloadNginx(); err != nil {
		return result.Fail[struct{}](ErrNginxFailed)
	}

	newActiveSlot := servicedomain.Slot(inactiveSlot)
	if err := uc.serviceRepo.UpdateActiveSlot(input.ServiceID, newActiveSlot); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	if err := uc.deployRepo.SetStatus(currentActive.ID, deploydomain.StatusRolledBack, nil); err != nil {
		return result.Fail[struct{}](ErrInternal)
	}

	return result.Ok(struct{}{})
}

func inactive(slot servicedomain.Slot) deploydomain.Slot {
	if slot == servicedomain.SlotBlue {
		return deploydomain.SlotGreen
	}
	return deploydomain.SlotBlue
}

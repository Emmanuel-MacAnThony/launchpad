package update

import (
	"errors"
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	GetByID(id string) (domain.Service, error)
	Update(id, name, healthCheckURL string) error
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input UpdateInput) result.Result[UpdateOutput] {
	if input.ID == "" || (input.Name == "" && input.HealthCheckURL == "") {
		return result.Fail[UpdateOutput](ErrInvalidInput)
	}

	svc, err := uc.repo.GetByID(input.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return result.Fail[UpdateOutput](ErrNotFound)
		}
		return result.Fail[UpdateOutput](fmt.Errorf("%w: %s", ErrInternalError, err))
	}

	if input.Name != "" {
		svc.Name = input.Name
	}
	if input.HealthCheckURL != "" {
		svc.HealthCheckURL = input.HealthCheckURL
	}

	if err := uc.repo.Update(svc.ID, svc.Name, svc.HealthCheckURL); err != nil {
		return result.Fail[UpdateOutput](fmt.Errorf("%w: %s", ErrPersistFailed, err))
	}

	return result.Ok(UpdateOutput{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoURL:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckURL: svc.HealthCheckURL,
		Host:           svc.Host,
		SSHUser:        svc.SSHUser,
		SSHKeyPath:     svc.SSHKeyPath,
		CreatedAt:      svc.CreatedAt,
	})
}

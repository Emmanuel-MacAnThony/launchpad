package get

import (
	"errors"
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	GetByID(id string) (domain.Service, error)
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input GetInput) result.Result[GetOutput] {
	if input.ID == "" {
		return result.Fail[GetOutput](ErrInvalidInput)
	}

	svc, err := uc.repo.GetByID(input.ID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return result.Fail[GetOutput](ErrNotFound)
		}
		return result.Fail[GetOutput](fmt.Errorf("%w: %s", ErrInternalError, err))
	}

	return result.Ok(GetOutput{
		ID:             svc.ID,
		Name:           svc.Name,
		RepoURL:        svc.RepoURL,
		Domain:         svc.Domain,
		HealthCheckURL: svc.HealthCheckURL,
		Host:           svc.Host,
		SSHUser:        svc.SSHUser,
		SSHKeyPath:     svc.SSHKeyPath,
		WebhookSecret:  svc.WebhookSecret,
		CreatedAt:      svc.CreatedAt,
	})
}

package list

import (
	"fmt"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	ListAll() ([]domain.Service, error)
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(_ ListInput) result.Result[ListOutput] {
	svcs, err := uc.repo.ListAll()
	if err != nil {
		return result.Fail[ListOutput](fmt.Errorf("%w: %s", ErrInternalError, err))
	}

	items := make([]ListItem, len(svcs))
	for i, svc := range svcs {
		items[i] = ListItem{
			ID:             svc.ID,
			Name:           svc.Name,
			RepoURL:        svc.RepoURL,
			Domain:         svc.Domain,
			HealthCheckURL: svc.HealthCheckURL,
			Host:           svc.Host,
			SSHUser:        svc.SSHUser,
			SSHKeyPath:     svc.SSHKeyPath,
			CreatedAt:      svc.CreatedAt,
		}
	}

	return result.Ok(ListOutput{Services: items})
}

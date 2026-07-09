package listdeploys

import (
	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	List(serviceID string) ([]deploydomain.Deploy, error)
}

type ListDeploysInput struct {
	ServiceID string
}

type ListDeploysOutput struct {
	Deploys []deploydomain.Deploy
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input ListDeploysInput) result.Result[ListDeploysOutput] {
	if input.ServiceID == "" {
		return result.Fail[ListDeploysOutput](ErrValidation)
	}

	deploys, err := uc.repo.List(input.ServiceID)
	if err != nil {
		return result.Fail[ListDeploysOutput](ErrInternal)
	}

	return result.Ok(ListDeploysOutput{Deploys: deploys})
}

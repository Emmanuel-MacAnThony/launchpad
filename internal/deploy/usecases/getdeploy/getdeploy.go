package getdeploy

import (
	"errors"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	GetByID(deployID string) (deploydomain.Deploy, error)
}

type GetDeployInput struct {
	DeployID string
}

type GetDeployOutput struct {
	Deploy deploydomain.Deploy
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input GetDeployInput) result.Result[GetDeployOutput] {
	if input.DeployID == "" {
		return result.Fail[GetDeployOutput](ErrValidation)
	}

	deploy, err := uc.repo.GetByID(input.DeployID)
	if err != nil {
		if errors.Is(err, deploydomain.ErrNotFound) {
			return result.Fail[GetDeployOutput](ErrNotFound)
		}
		return result.Fail[GetDeployOutput](ErrInternal)
	}

	return result.Ok(GetDeployOutput{Deploy: deploy})
}

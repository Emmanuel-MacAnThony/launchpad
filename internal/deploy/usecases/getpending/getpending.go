package getpending

import (
	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	ListPending() ([]deploydomain.Deploy, error)
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute() result.Result[GetPendingOutput] {
	deploys, err := uc.repo.ListPending()
	if err != nil {
		return result.Fail[GetPendingOutput](ErrInternal)
	}

	return result.Ok(GetPendingOutput{Deploys: deploys})
}

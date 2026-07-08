package create

import (
	"errors"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type Repo interface {
	EnqueueDeploy(serviceID, commitSHA, commitMessage string, pushedAt time.Time) (deploydomain.Deploy, CreateResult, error)
}

type UseCase struct {
	repo Repo
}

func New(repo Repo) *UseCase {
	return &UseCase{repo: repo}
}

func (uc *UseCase) Execute(input CreateInput) result.Result[CreateOutput] {
	if input.ServiceID == "" || input.CommitSHA == "" || input.PushedAt.IsZero() {
		return result.Fail[CreateOutput](ErrValidation)
	}

	deploy, createResult, err := uc.repo.EnqueueDeploy(
		input.ServiceID,
		input.CommitSHA,
		input.CommitMessage,
		input.PushedAt,
	)
	if err != nil {
		if errors.Is(err, deploydomain.ErrServiceNotFound) {
			return result.Fail[CreateOutput](ErrServiceNotFound)
		}
		return result.Fail[CreateOutput](ErrInternal)
	}

	return result.Ok(CreateOutput{
		Deploy: deploy,
		Result: createResult,
	})
}

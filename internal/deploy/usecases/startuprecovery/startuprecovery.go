package startuprecovery

import "github.com/Emmanuel-MacAnThony/launchpad/pkg/result"

type DeployRepo interface {
	StartupRecovery() (int64, error)
}

type Output struct {
	Count int64
}

type UseCase struct {
	deployRepo DeployRepo
}

func New(deployRepo DeployRepo) *UseCase {
	return &UseCase{deployRepo: deployRepo}
}

func (uc *UseCase) Execute() result.Result[Output] {
	count, err := uc.deployRepo.StartupRecovery()
	if err != nil {
		return result.Fail[Output](ErrInternal)
	}
	return result.Ok(Output{Count: count})
}

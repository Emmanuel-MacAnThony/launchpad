package adapters

import (
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/activate"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/rollback"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
	sharedssh "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/ssh"
)

// CreateSSHExecutorAdapter wraps sharedssh.Executor to satisfy create.SSHExecutor.
// The two SSHResult types are structurally identical but are different named types,
// so the conversion happens here rather than in either package.
type CreateSSHExecutorAdapter struct{ Ex *sharedssh.Executor }

func (a *CreateSSHExecutorAdapter) Run(cmd string) (create.SSHResult, error) {
	res, err := a.Ex.Run(cmd)
	return create.SSHResult{Stdout: res.Stdout, Stderr: res.Stderr}, err
}

func (a *CreateSSHExecutorAdapter) Close() error { return a.Ex.Close() }

// CreateSSHFactoryAdapter bridges sharedssh.Factory to create.SSHExecutorFactory.
type CreateSSHFactoryAdapter struct{ F *sharedssh.Factory }

func (a *CreateSSHFactoryAdapter) NewExecutor(cfg create.SSHConfig) (create.SSHExecutor, error) {
	ex, err := a.F.NewExecutor(sharedssh.SSHConfig{Host: cfg.Host, User: cfg.User, KeyBytes: cfg.KeyBytes})
	if err != nil {
		return nil, err
	}
	return &CreateSSHExecutorAdapter{Ex: ex}, nil
}

// ActivateSSHExecutorAdapter wraps sharedssh.Executor to satisfy activate.SSHExecutor.
type ActivateSSHExecutorAdapter struct{ Ex *sharedssh.Executor }

func (a *ActivateSSHExecutorAdapter) Run(cmd string) (activate.SSHResult, error) {
	res, err := a.Ex.Run(cmd)
	return activate.SSHResult{Stdout: res.Stdout, Stderr: res.Stderr}, err
}

func (a *ActivateSSHExecutorAdapter) Upload(local, remote string) error {
	return a.Ex.Upload(local, remote)
}

func (a *ActivateSSHExecutorAdapter) Close() error { return a.Ex.Close() }

// ActivateSSHFactoryAdapter bridges sharedssh.Factory to activate.SSHExecutorFactory.
type ActivateSSHFactoryAdapter struct{ F *sharedssh.Factory }

func (a *ActivateSSHFactoryAdapter) NewExecutor(cfg activate.SSHConfig) (activate.SSHExecutor, error) {
	ex, err := a.F.NewExecutor(sharedssh.SSHConfig{Host: cfg.Host, User: cfg.User, KeyBytes: cfg.KeyBytes})
	if err != nil {
		return nil, err
	}
	return &ActivateSSHExecutorAdapter{Ex: ex}, nil
}

// RollbackSSHExecutorAdapter wraps sharedssh.Executor to satisfy rollback.SSHExecutor.
type RollbackSSHExecutorAdapter struct{ Ex *sharedssh.Executor }

func (a *RollbackSSHExecutorAdapter) Run(cmd string) (rollback.SSHResult, error) {
	res, err := a.Ex.Run(cmd)
	return rollback.SSHResult{Stdout: res.Stdout, Stderr: res.Stderr}, err
}

func (a *RollbackSSHExecutorAdapter) Upload(local, remote string) error {
	return a.Ex.Upload(local, remote)
}

func (a *RollbackSSHExecutorAdapter) Close() error { return a.Ex.Close() }

// RollbackSSHFactoryAdapter bridges sharedssh.Factory to rollback.SSHExecutorFactory.
type RollbackSSHFactoryAdapter struct{ F *sharedssh.Factory }

func (a *RollbackSSHFactoryAdapter) NewExecutor(cfg rollback.SSHConfig) (rollback.SSHExecutor, error) {
	ex, err := a.F.NewExecutor(sharedssh.SSHConfig{Host: cfg.Host, User: cfg.User, KeyBytes: cfg.KeyBytes})
	if err != nil {
		return nil, err
	}
	return &RollbackSSHExecutorAdapter{Ex: ex}, nil
}

package agent

import (
	"context"
	"sync"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/activate"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/getpending"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/recoverybuild"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/refreshlock"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/startuprecovery"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/updatestatus"
	serviceget "github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	sharedssh "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/ssh"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/logger"
)

// SSHExecutorFactory is defined here so the agent owns its dependency contract.
// The concrete implementation is sharedssh.Factory; it satisfies this interface
// structurally — no adapter needed.
type SSHExecutorFactory interface {
	NewExecutor(cfg sharedssh.SSHConfig) (*sharedssh.Executor, error)
}

type Agent struct {
	mu            sync.Mutex
	activeWorkers map[string]bool

	startupRecovery *startuprecovery.UseCase
	recoverBuild    *recoverybuild.UseCase
	getPending      *getpending.UseCase
	getService      *serviceget.UseCase
	updateStatus    *updatestatus.UseCase
	refreshLock     *refreshlock.UseCase
	activate        *activate.UseCase

	sshFactory SSHExecutorFactory
	log        *logger.Logger
}

func New(
	log *logger.Logger,
	startupRecovery *startuprecovery.UseCase,
	recoverBuild *recoverybuild.UseCase,
	getPending *getpending.UseCase,
	getService *serviceget.UseCase,
	updateStatus *updatestatus.UseCase,
	refreshLock *refreshlock.UseCase,
	activate *activate.UseCase,
	sshFactory SSHExecutorFactory,
) *Agent {
	return &Agent{
		activeWorkers:   make(map[string]bool),
		startupRecovery: startupRecovery,
		recoverBuild:    recoverBuild,
		getPending:      getPending,
		getService:      getService,
		updateStatus:    updateStatus,
		refreshLock:     refreshLock,
		activate:        activate,
		sshFactory:      sshFactory,
		log:             log,
	}
}

func (a *Agent) Start(ctx context.Context) {
	res := a.startupRecovery.Execute()
	if !res.IsOk() {
		a.log.Error("startup recovery failed", "err", res.Err)
	} else if res.Value.Count > 0 {
		a.log.Info("startup recovery reset stale deploys", "count", res.Value.Count)
	}

	go a.runScheduler(ctx)
	go a.runRecoveryScanner(ctx)
}

func (a *Agent) workerDone(serviceID string) {
	a.mu.Lock()
	delete(a.activeWorkers, serviceID)
	a.mu.Unlock()
}

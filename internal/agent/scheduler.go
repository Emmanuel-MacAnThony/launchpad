package agent

import (
	"context"
	"time"
)

const schedulerInterval = 5 * time.Second

func (a *Agent) runScheduler(ctx context.Context) {
	ticker := time.NewTicker(schedulerInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.dispatch(ctx)
		}
	}
}

func (a *Agent) dispatch(ctx context.Context) {
	res := a.getPending.Execute()
	if !res.IsOk() {
		a.log.Error("scheduler: failed to fetch pending deploys", "err", res.Err)
		return
	}

	for _, deploy := range res.Value.Deploys {
		a.mu.Lock()
		if a.activeWorkers[deploy.ServiceID] {
			// A worker is already running for this service. Only one deploy
			// per service at a time — the next pending will be picked up
			// after the current worker finishes and removes itself from the map.
			a.mu.Unlock()
			continue
		}
		a.activeWorkers[deploy.ServiceID] = true
		a.mu.Unlock()

		go a.runWorker(ctx, deploy)
	}
}

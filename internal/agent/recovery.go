package agent

import (
	"context"
	"time"
)

// recoveryInterval is longer than the scheduler interval because there is no
// point scanning before the lock duration (10min) has elapsed — a lock cannot
// expire sooner than that, so scanning more frequently would always find nothing.
const recoveryInterval = 2 * time.Minute

func (a *Agent) runRecoveryScanner(ctx context.Context) {
	ticker := time.NewTicker(recoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Agent is shutting down — stop the scanner.
			return
		case <-ticker.C:
			res := a.recoverBuild.Execute()
			if !res.IsOk() {
				a.log.Error("recovery scanner: failed to reset expired deploys", "err", res.Err)
				continue
			}
			if res.Value.Count > 0 {
				// Non-zero means a worker goroutine crashed mid-build and stopped
				// refreshing its lock. The deploy is back in pending — the scheduler
				// will dispatch a new worker for it on the next tick.
				a.log.Warn("recovery scanner: reset expired building deploys to pending", "count", res.Value.Count)
			}
		}
	}
}

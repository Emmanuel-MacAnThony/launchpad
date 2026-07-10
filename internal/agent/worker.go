package agent

import (
	"context"
	"fmt"
	"net/url"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/activate"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/refreshlock"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/updatestatus"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	serviceget "github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	sharedssh "github.com/Emmanuel-MacAnThony/launchpad/internal/shared/ssh"
)

const (
	refreshInterval = 5 * time.Minute
	buildDir        = "/tmp/launchpad-builds"
)

func (a *Agent) runWorker(ctx context.Context, deploy deploydomain.Deploy) {
	// Always remove this service from the active map when the worker exits,
	// regardless of success or failure. This unblocks the scheduler from
	// dispatching a new worker for this service on the next tick.
	defer a.workerDone(deploy.ServiceID)

	log := a.log.With("deployID", deploy.ID, "serviceID", deploy.ServiceID)

	// Fetch service config — we need SSH credentials, ports, and active slot
	// to determine which slot this deploy targets.
	svcRes := a.getService.Execute(serviceget.GetInput{ID: deploy.ServiceID})
	if !svcRes.IsOk() {
		log.Error("worker: failed to fetch service", "err", svcRes.Err)
		return
	}
	svc := svcRes.Value

	// The deploy always targets the slot opposite to the currently active one.
	// If no deploy has ever succeeded (ActiveSlot == nil), blue is the default.
	slot := oppositeSlot(svc.ActiveSlot)
	port := slotPort(svc, slot)

	// Transition pending → building. This also acquires the deploy lock, which
	// acts as a liveness signal — the refresher below keeps it alive while the
	// build runs. If this worker crashes, the lock expires and the recovery
	// scanner resets the deploy back to pending.
	buildRes := a.updateStatus.Execute(updatestatus.UpdateStatusInput{
		DeployID:  deploy.ID,
		NewStatus: deploydomain.StatusBuilding,
		Slot:      &slot,
	})
	if !buildRes.IsOk() {
		log.Error("worker: failed to transition to building", "err", buildRes.Err)
		// Deploy stays pending — scheduler will pick it up again on next tick.
		return
	}

	// markFailed transitions building → failed and releases the lock.
	// Called on any step failure so the deploy doesn't get stuck in building.
	markFailed := func() {
		res := a.updateStatus.Execute(updatestatus.UpdateStatusInput{
			DeployID:  deploy.ID,
			NewStatus: deploydomain.StatusFailed,
		})
		if !res.IsOk() {
			log.Error("worker: failed to mark deploy as failed", "err", res.Err)
		}
	}

	// Start the refresher as a child goroutine controlled by its own context.
	// We cancel it explicitly before any terminal operation (activate or markFailed)
	// to avoid refreshing a lock that is about to be released.
	refreshCtx, cancelRefresh := context.WithCancel(ctx)
	defer cancelRefresh()
	go a.runRefresher(refreshCtx, deploy.ID)

	// Open one SSH connection for the entire build — each command reuses this
	// session instead of re-dialing for every step.
	executor, err := a.sshFactory.NewExecutor(sharedssh.SSHConfig{
		Host:    svc.Host,
		User:    svc.SSHUser,
		KeyPath: svc.SSHKeyPath,
	})
	if err != nil {
		log.Error("worker: failed to open SSH connection", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	defer executor.Close()

	imageName := fmt.Sprintf("%s:%s", svc.Name, deploy.CommitSHA[:7])
	containerName := fmt.Sprintf("%s-%s", svc.Name, slot)
	deployBuildDir := fmt.Sprintf("%s/%s", buildDir, deploy.ID)

	steps := []struct {
		name string
		cmd  string
	}{
		{
			"clone",
			fmt.Sprintf("git clone %s %s", svc.RepoURL, deployBuildDir),
		},
		{
			// Pin to the exact commit that triggered this deploy, not whatever
			// HEAD is at the time the worker runs.
			"checkout",
			fmt.Sprintf("git -C %s checkout %s", deployBuildDir, deploy.CommitSHA),
		},
		{
			"build image",
			fmt.Sprintf("docker build -t %s %s", imageName, deployBuildDir),
		},
		{
			// Stop and remove the existing container on this slot if one is running.
			// The || true prevents failure when no container exists (e.g. first deploy).
			"stop old container",
			fmt.Sprintf("docker stop %s 2>/dev/null || true && docker rm %s 2>/dev/null || true", containerName, containerName),
		},
		{
			"run container",
			fmt.Sprintf("docker run -d -p %d:%d --name %s --restart unless-stopped %s",
				port, svc.ContainerPort, containerName, imageName),
		},
		{
			// Health check hits the container directly on the host port via localhost,
			// not through the public URL — nginx still points to the old slot at this
			// stage, so a request to the domain would miss this container entirely.
			// Retry for up to ~30s to give the container time to start.
			"health check",
			fmt.Sprintf("curl -sf --retry 10 --retry-delay 3 --retry-all-errors http://localhost:%d%s", port, healthPath(svc.HealthCheckURL)),
		},
		{
			// Clean up the build directory — the image is already stored in Docker,
			// so the source is no longer needed on disk.
			"cleanup",
			fmt.Sprintf("rm -rf %s", deployBuildDir),
		},
	}

	for _, step := range steps {
		if _, err := executor.Run(step.cmd); err != nil {
			log.Error("worker: build step failed", "step", step.name, "err", err)
			cancelRefresh()
			markFailed()
			return
		}
		log.Info("worker: step complete", "step", step.name)
	}

	// All build steps passed. Cancel the refresher before activating —
	// activate releases the lock as its final step, and a concurrent refresh
	// on a released lock would be a no-op at best, misleading at worst.
	cancelRefresh()

	// Activate: switch nginx to the new slot, reload nginx, update the service's
	// active_slot, set deploy status to active, and release the lock.
	activateRes := a.activate.Execute(activate.ActivateInput{
		DeployID:   deploy.ID,
		ServiceID:  deploy.ServiceID,
		Slot:       slot,
		Host:       svc.Host,
		SSHUser:    svc.SSHUser,
		SSHKeyPath: svc.SSHKeyPath,
		Domain:     svc.Domain,
		ActivePort: port,
	})
	if !activateRes.IsOk() {
		log.Error("worker: activation failed", "err", activateRes.Err)
		return
	}

	log.Info("worker: deploy complete", "slot", slot, "image", imageName)
}

func (a *Agent) runRefresher(ctx context.Context, deployID string) {
	ticker := time.NewTicker(refreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Worker cancelled us — either it finished or failed.
			// The terminal operation (activate or markFailed) handles lock release.
			return
		case <-ticker.C:
			res := a.refreshLock.Execute(refreshlock.RefreshLockInput{DeployID: deployID})
			if !res.IsOk() {
				// A single failed refresh doesn't abort the build — log and continue.
				// If refreshes keep failing and the lock expires, the recovery scanner
				// will reset the deploy to pending so it can be retried.
				a.log.Warn("refresher: failed to refresh lock", "deployID", deployID, "err", res.Err)
			}
		}
	}
}

// oppositeSlot returns the slot that is NOT currently active.
// If no slot is active (first deploy ever), blue is the default starting slot.
func oppositeSlot(current *servicedomain.Slot) deploydomain.Slot {
	if current != nil && *current == servicedomain.SlotBlue {
		return deploydomain.SlotGreen
	}
	// active is green → target blue; nil (first deploy) → also target blue
	return deploydomain.SlotBlue
}

// slotPort returns the host-side port for the given deployment slot.
func slotPort(svc serviceget.GetOutput, slot deploydomain.Slot) int {
	if slot == deploydomain.SlotGreen {
		return svc.GreenPort
	}
	return svc.BluePort
}

// healthPath extracts the path from a full URL for the SSH health check curl.
// Falls back to /health if the URL cannot be parsed or has no path.
func healthPath(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil || u.Path == "" {
		return "/health"
	}
	return u.Path
}

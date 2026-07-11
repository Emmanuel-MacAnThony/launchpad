package agent

import (
	"context"
	"fmt"
	"net/url"
	"os"
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
		Host:     svc.Host,
		User:     svc.SSHUser,
		KeyBytes: []byte(svc.SSHKey),
	})
	if err != nil {
		log.Error("worker: failed to open SSH connection", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	defer executor.Close()

	projectName := fmt.Sprintf("%s-%s", svc.Name, slot)
	deployBuildDir := fmt.Sprintf("%s/%s", buildDir, deploy.ID)
	overridePath := fmt.Sprintf("/tmp/launchpad-overrides/%s.yml", projectName)

	// Generate the Compose override locally and upload it to the customer server.
	// The override pins the host port and container name for this slot without
	// touching the user's docker-compose.yml — their repo stays unchanged.
	overrideYAML := fmt.Sprintf(
		"services:\n  %s:\n    container_name: %s\n    ports:\n      - \"%d:%d\"\n    restart: unless-stopped\n",
		svc.ComposeSvc,
		projectName, port, svc.ContainerPort,
	)
	tmpOverride, err := os.CreateTemp("", "launchpad-override-*.yml")
	if err != nil {
		log.Error("worker: failed to create override temp file", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	defer os.Remove(tmpOverride.Name())
	if _, err := tmpOverride.WriteString(overrideYAML); err != nil {
		tmpOverride.Close()
		log.Error("worker: failed to write override file", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	tmpOverride.Close()

	if _, err := executor.Run("mkdir -p /tmp/launchpad-overrides"); err != nil {
		log.Error("worker: failed to create overrides dir on host", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	if err := executor.Upload(tmpOverride.Name(), overridePath); err != nil {
		log.Error("worker: failed to upload compose override", "err", err)
		cancelRefresh()
		markFailed()
		return
	}
	log.Info("worker: compose override uploaded", "path", overridePath)

	steps := []struct {
		name string
		cmd  string
	}{
		{
			"clone",
			fmt.Sprintf("git clone %s %s", svc.RepoURL, deployBuildDir),
		},
		{
			// Pin to the exact commit that triggered this deploy.
			"checkout",
			fmt.Sprintf("git -C %s checkout %s", deployBuildDir, deploy.CommitSHA),
		},
		{
			// Compose merges docker-compose.yml with the Launchpad override, which
			// sets the host port and container name for this slot. The user's repo
			// is never modified. --build forces a fresh image on every deploy.
			"compose up",
			fmt.Sprintf(
				"docker compose -f %s/docker-compose.yml -f %s -p %s up -d --build",
				deployBuildDir, overridePath, projectName,
			),
		},
		{
			// Health check via localhost so we bypass nginx (still pointing at old slot).
			// Compose builds take longer than raw docker run — allow up to 2 minutes.
			"health check",
			fmt.Sprintf(
				"curl -sf --retry 30 --retry-delay 4 --retry-all-errors http://localhost:%d%s",
				port, healthPath(svc.HealthCheckURL),
			),
		},
		{
			// Remove the cloned source — the image layers are cached in Docker.
			// The override file at /tmp/launchpad-overrides/ is intentionally kept
			// so docker compose can reference it if the stack needs to be torn down.
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
		SSHKey: svc.SSHKey,
		Domain:     svc.Domain,
		ActivePort: port,
	})
	if !activateRes.IsOk() {
		log.Error("worker: activation failed", "err", activateRes.Err)
		markFailed()
		return
	}

	log.Info("worker: deploy complete", "slot", slot, "project", projectName)
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

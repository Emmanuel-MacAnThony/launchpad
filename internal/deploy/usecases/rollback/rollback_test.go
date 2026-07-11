package rollback_test

import (
	"errors"
	"strings"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/rollback"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

// --- stubs ---

type stubServiceRepo struct {
	service     servicedomain.Service
	getErr      error
	updateErr   error
	updatedSlot servicedomain.Slot
}

func (r *stubServiceRepo) GetByID(_ string) (servicedomain.Service, error) {
	return r.service, r.getErr
}

func (r *stubServiceRepo) UpdateActiveSlot(_ string, slot servicedomain.Slot) error {
	r.updatedSlot = slot
	return r.updateErr
}

type stubDeployRepo struct {
	activeDeploy    deploydomain.Deploy
	activeErr       error
	latestDeploy    deploydomain.Deploy
	latestErr       error
	setStatusErr    error
	setStatusCalled bool
}

// GetLatestOnSlot is called once per slot: for the active slot (blue in these
// fixtures) it returns the current live deploy; for the inactive slot (green) it
// returns the deploy being rolled back to. The stub keys off the slot so both
// calls can be exercised independently.
func (r *stubDeployRepo) GetLatestOnSlot(_ string, slot deploydomain.Slot) (deploydomain.Deploy, error) {
	if slot == deploydomain.SlotBlue {
		return r.activeDeploy, r.activeErr
	}
	return r.latestDeploy, r.latestErr
}

func (r *stubDeployRepo) SetStatus(_ string, _ deploydomain.DeployStatus, _ *deploydomain.Slot) error {
	r.setStatusCalled = true
	return r.setStatusErr
}

type stubSSHExecutor struct {
	runFn      func(cmd string) (rollback.SSHResult, error)
	uploadErr  error
	uploadedTo string
}

func (s *stubSSHExecutor) Run(cmd string) (rollback.SSHResult, error) {
	if s.runFn != nil {
		return s.runFn(cmd)
	}
	return rollback.SSHResult{}, nil
}

func (s *stubSSHExecutor) Upload(_, remote string) error {
	s.uploadedTo = remote
	return s.uploadErr
}

func (s *stubSSHExecutor) Close() error { return nil }

type stubSSHFactory struct {
	executor rollback.SSHExecutor
	dialErr  error
}

func (f *stubSSHFactory) NewExecutor(_ rollback.SSHConfig) (rollback.SSHExecutor, error) {
	return f.executor, f.dialErr
}

// --- fixtures ---

var blueSlot = servicedomain.SlotBlue

var activeService = servicedomain.Service{
	ID:         "svc-1",
	Host:       "10.0.0.1",
	SSHUser:    "ubuntu",
	SSHKey:     "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key\n-----END OPENSSH PRIVATE KEY-----",
	Domain:     "app.example.com",
	BluePort:   3001,
	GreenPort:  3002,
	ActiveSlot: &blueSlot,
}

var activeDeploy = deploydomain.Deploy{
	ID:        "dep-active",
	ServiceID: "svc-1",
	Status:    deploydomain.StatusActive,
	Slot:      slotPtr(deploydomain.SlotBlue),
}

var previousDeploy = deploydomain.Deploy{
	ID:        "dep-prev",
	ServiceID: "svc-1",
	Status:    deploydomain.StatusRolledBack,
	Slot:      slotPtr(deploydomain.SlotGreen),
}

func slotPtr(s deploydomain.Slot) *deploydomain.Slot { return &s }

func happyExecutor() *stubSSHExecutor { return &stubSSHExecutor{} }
func happyFactory() *stubSSHFactory   { return &stubSSHFactory{executor: happyExecutor()} }

// --- tests ---

func TestRollback_EmptyServiceID(t *testing.T) {
	res := rollback.New(&stubServiceRepo{}, &stubDeployRepo{}, happyFactory()).Execute(rollback.RollbackInput{})
	if !errors.Is(res.Err, rollback.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestRollback_ServiceNotFound(t *testing.T) {
	res := rollback.New(&stubServiceRepo{getErr: servicedomain.ErrNotFound}, &stubDeployRepo{}, happyFactory()).
		Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", res.Err)
	}
}

func TestRollback_ServiceRepoError(t *testing.T) {
	res := rollback.New(&stubServiceRepo{getErr: errors.New("db down")}, &stubDeployRepo{}, happyFactory()).
		Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_NeverDeployed(t *testing.T) {
	svc := activeService
	svc.ActiveSlot = nil
	res := rollback.New(&stubServiceRepo{service: svc}, &stubDeployRepo{}, happyFactory()).
		Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNoActiveDeployment) {
		t.Fatalf("expected ErrNoActiveDeployment, got %v", res.Err)
	}
}

func TestRollback_NoActiveDeployment(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeErr: deploydomain.ErrNotFound},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNoActiveDeployment) {
		t.Fatalf("expected ErrNoActiveDeployment, got %v", res.Err)
	}
}

func TestRollback_ActiveDeployRepoError(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeErr: errors.New("db down")},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_NoPreviousDeployment(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestErr: deploydomain.ErrNotFound},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNoPreviousDeployment) {
		t.Fatalf("expected ErrNoPreviousDeployment, got %v", res.Err)
	}
}

func TestRollback_LatestOnSlotRepoError(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestErr: errors.New("db down")},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_SSHFailed(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubSSHFactory{dialErr: errors.New("connection refused")},
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrSSHFailed) {
		t.Fatalf("expected ErrSSHFailed, got %v", res.Err)
	}
}

func TestRollback_UploadFailed(t *testing.T) {
	svcRepo := &stubServiceRepo{service: activeService}
	res := rollback.New(
		svcRepo,
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubSSHFactory{executor: &stubSSHExecutor{uploadErr: errors.New("scp failed")}},
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when upload fails")
	}
}

func TestRollback_NginxTFailed(t *testing.T) {
	var cleanupRan bool
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (rollback.SSHResult, error) {
			if cmd == "nginx -t" {
				return rollback.SSHResult{Stderr: "config test failed"}, errors.New("exit status 1")
			}
			if strings.HasPrefix(cmd, "rm -f") {
				cleanupRan = true
			}
			return rollback.SSHResult{}, nil
		},
	}
	svcRepo := &stubServiceRepo{service: activeService}
	res := rollback.New(
		svcRepo,
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubSSHFactory{executor: ex},
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if !cleanupRan {
		t.Error("expected rm -f cleanup to run after nginx -t failure")
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx -t fails")
	}
}

func TestRollback_NginxReloadFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (rollback.SSHResult, error) {
			if cmd == "nginx -s reload" {
				return rollback.SSHResult{}, errors.New("exit status 1")
			}
			return rollback.SSHResult{}, nil
		},
	}
	svcRepo := &stubServiceRepo{service: activeService}
	res := rollback.New(
		svcRepo,
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubSSHFactory{executor: ex},
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx reload fails")
	}
}

func TestRollback_UpdateActiveSlotFailed(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService, updateErr: errors.New("db down")},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_SetStatusFailed(t *testing.T) {
	res := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy, setStatusErr: errors.New("db down")},
		happyFactory(),
	).Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_Success(t *testing.T) {
	svcRepo := &stubServiceRepo{service: activeService}
	deployRepo := &stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy}
	ex := happyExecutor()

	res := rollback.New(svcRepo, deployRepo, &stubSSHFactory{executor: ex}).
		Execute(rollback.RollbackInput{ServiceID: "svc-1"})

	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	// active slot was blue → rolled back to green
	if ex.uploadedTo != "/etc/nginx/launchpad/svc-1.conf" {
		t.Errorf("expected upload to /etc/nginx/launchpad/svc-1.conf, got %q", ex.uploadedTo)
	}
	if svcRepo.updatedSlot != servicedomain.SlotGreen {
		t.Errorf("expected active_slot=green, got %v", svcRepo.updatedSlot)
	}
	if !deployRepo.setStatusCalled {
		t.Error("expected SetStatus to be called")
	}
}

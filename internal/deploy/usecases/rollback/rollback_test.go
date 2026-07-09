package rollback_test

import (
	"errors"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/rollback"
)

// --- stubs ---

type stubServiceRepo struct {
	service     servicedomain.Service
	getErr      error
	updateErr   error
	updatedSlot servicedomain.Slot
}

func (r *stubServiceRepo) GetByID(serviceID string) (servicedomain.Service, error) {
	return r.service, r.getErr
}

func (r *stubServiceRepo) UpdateActiveSlot(serviceID string, slot servicedomain.Slot) error {
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

func (r *stubDeployRepo) GetActiveForService(serviceID string) (deploydomain.Deploy, error) {
	return r.activeDeploy, r.activeErr
}

func (r *stubDeployRepo) GetLatestOnSlot(serviceID string, slot deploydomain.Slot) (deploydomain.Deploy, error) {
	return r.latestDeploy, r.latestErr
}

func (r *stubDeployRepo) SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error {
	r.setStatusCalled = true
	return r.setStatusErr
}

type stubNginxClient struct {
	err    error
	called bool
	slot   deploydomain.Slot
}

func (n *stubNginxClient) Switch(host, domain string, slot deploydomain.Slot) error {
	n.called = true
	n.slot = slot
	return n.err
}

// --- fixtures ---

var blueSlot = servicedomain.SlotBlue

var activeService = servicedomain.Service{
	ID:         "svc-1",
	Host:       "10.0.0.1",
	Domain:     "app.example.com",
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

// --- tests ---

func TestRollback_EmptyServiceID(t *testing.T) {
	uc := rollback.New(&stubServiceRepo{}, &stubDeployRepo{}, &stubNginxClient{})
	res := uc.Execute(rollback.RollbackInput{})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestRollback_ServiceNotFound(t *testing.T) {
	uc := rollback.New(&stubServiceRepo{getErr: servicedomain.ErrNotFound}, &stubDeployRepo{}, &stubNginxClient{})
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", res.Err)
	}
}

func TestRollback_ServiceRepoError(t *testing.T) {
	uc := rollback.New(&stubServiceRepo{getErr: errors.New("db down")}, &stubDeployRepo{}, &stubNginxClient{})
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_NeverDeployed(t *testing.T) {
	svc := activeService
	svc.ActiveSlot = nil
	uc := rollback.New(&stubServiceRepo{service: svc}, &stubDeployRepo{}, &stubNginxClient{})
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrNoActiveDeployment) {
		t.Fatalf("expected ErrNoActiveDeployment, got %v", res.Err)
	}
}

func TestRollback_NoActiveDeployment(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeErr: deploydomain.ErrNotFound},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrNoActiveDeployment) {
		t.Fatalf("expected ErrNoActiveDeployment, got %v", res.Err)
	}
}

func TestRollback_ActiveDeployRepoError(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeErr: errors.New("db down")},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_NoPreviousDeployment(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestErr: deploydomain.ErrNotFound},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrNoPreviousDeployment) {
		t.Fatalf("expected ErrNoPreviousDeployment, got %v", res.Err)
	}
}

func TestRollback_LatestOnSlotRepoError(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestErr: errors.New("db down")},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_NginxFailed(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubNginxClient{err: errors.New("nginx error")},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
}

func TestRollback_UpdateActiveSlotFailed(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService, updateErr: errors.New("db down")},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_SetStatusFailed(t *testing.T) {
	uc := rollback.New(
		&stubServiceRepo{service: activeService},
		&stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy, setStatusErr: errors.New("db down")},
		&stubNginxClient{},
	)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, rollback.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRollback_Success(t *testing.T) {
	svcRepo := &stubServiceRepo{service: activeService}
	deployRepo := &stubDeployRepo{activeDeploy: activeDeploy, latestDeploy: previousDeploy}
	nginx := &stubNginxClient{}

	uc := rollback.New(svcRepo, deployRepo, nginx)
	res := uc.Execute(rollback.RollbackInput{ServiceID: "svc-1"})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}

	// nginx switched to inactive slot (green, since active was blue)
	if !nginx.called {
		t.Fatal("expected nginx.Switch to be called")
	}
	if nginx.slot != deploydomain.SlotGreen {
		t.Fatalf("expected nginx to switch to green, got %v", nginx.slot)
	}

	// service active_slot updated to green
	if svcRepo.updatedSlot != servicedomain.SlotGreen {
		t.Fatalf("expected service active_slot=green, got %v", svcRepo.updatedSlot)
	}

	// current active deploy marked rolled_back
	if !deployRepo.setStatusCalled {
		t.Fatal("expected SetStatus to be called")
	}
}

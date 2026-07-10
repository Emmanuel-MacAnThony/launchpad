package activate_test

import (
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/activate"
	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

// --- stubs ---

type stubDeployRepo struct {
	deploy       deploydomain.Deploy
	getErr       error
	setStatusErr error
	statusSet    deploydomain.DeployStatus
}

func (r *stubDeployRepo) GetByID(_ string) (deploydomain.Deploy, error) {
	return r.deploy, r.getErr
}

func (r *stubDeployRepo) SetStatus(_ string, status deploydomain.DeployStatus, _ *deploydomain.Slot) error {
	r.statusSet = status
	return r.setStatusErr
}

type stubServiceRepo struct {
	updateErr   error
	updatedSlot servicedomain.Slot
}

func (r *stubServiceRepo) UpdateActiveSlot(_ string, slot servicedomain.Slot) error {
	r.updatedSlot = slot
	return r.updateErr
}

type stubNginxClient struct {
	switchErr    error
	reloadErr    error
	switchCalled bool
	reloadCalled bool
	switchedSlot deploydomain.Slot
}

func (n *stubNginxClient) Switch(_ string, slot deploydomain.Slot) error {
	n.switchCalled = true
	n.switchedSlot = slot
	return n.switchErr
}

func (n *stubNginxClient) ReloadNginx() error {
	n.reloadCalled = true
	return n.reloadErr
}

type stubLockRepo struct {
	releaseErr     error
	releaseCalled  bool
}

func (l *stubLockRepo) ReleaseLock(_ string) error {
	l.releaseCalled = true
	return l.releaseErr
}

// --- fixtures ---

var buildingDeploy = deploydomain.Deploy{
	ID:        "dep-1",
	ServiceID: "svc-1",
	Status:    deploydomain.StatusBuilding,
}

var validInput = activate.ActivateInput{
	DeployID:  "dep-1",
	ServiceID: "svc-1",
	Slot:      deploydomain.SlotBlue,
}

func newUC(deployRepo *stubDeployRepo, svcRepo *stubServiceRepo, nginx *stubNginxClient, lockRepo *stubLockRepo) *activate.UseCase {
	return activate.New(nginx, svcRepo, deployRepo, lockRepo)
}

// --- tests ---

func TestActivate_HappyPath(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	svcRepo := &stubServiceRepo{}
	nginx := &stubNginxClient{}
	lockRepo := &stubLockRepo{}

	res := newUC(deployRepo, svcRepo, nginx, lockRepo).Execute(validInput)

	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if !nginx.switchCalled || nginx.switchedSlot != deploydomain.SlotBlue {
		t.Error("expected nginx.Switch to be called with blue slot")
	}
	if !nginx.reloadCalled {
		t.Error("expected nginx.ReloadNginx to be called")
	}
	if svcRepo.updatedSlot != servicedomain.SlotBlue {
		t.Errorf("expected active_slot=blue, got %v", svcRepo.updatedSlot)
	}
	if deployRepo.statusSet != deploydomain.StatusActive {
		t.Errorf("expected status=active, got %v", deployRepo.statusSet)
	}
	if !lockRepo.releaseCalled {
		t.Error("expected lock to be released")
	}
}

func TestActivate_MissingDeployID(t *testing.T) {
	input := validInput
	input.DeployID = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_MissingServiceID(t *testing.T) {
	input := validInput
	input.ServiceID = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_MissingSlot(t *testing.T) {
	input := validInput
	input.Slot = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_DeployNotFound(t *testing.T) {
	deployRepo := &stubDeployRepo{getErr: deploydomain.ErrNotFound}
	res := newUC(deployRepo, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrDeployNotFound) {
		t.Fatalf("expected ErrDeployNotFound, got %v", res.Err)
	}
}

func TestActivate_DeployRepoError(t *testing.T) {
	deployRepo := &stubDeployRepo{getErr: errors.New("db down")}
	res := newUC(deployRepo, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_DeployNotInBuildingState(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: deploydomain.Deploy{Status: deploydomain.StatusPending}}
	nginx := &stubNginxClient{}
	res := newUC(deployRepo, &stubServiceRepo{}, nginx, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", res.Err)
	}
	if nginx.switchCalled {
		t.Error("expected nginx.Switch not to be called when state is invalid")
	}
}

func TestActivate_NginxSwitchFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	svcRepo := &stubServiceRepo{}
	nginx := &stubNginxClient{switchErr: errors.New("disk error")}
	res := newUC(deployRepo, svcRepo, nginx, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx switch fails")
	}
}

func TestActivate_NginxReloadFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	svcRepo := &stubServiceRepo{}
	nginx := &stubNginxClient{reloadErr: errors.New("reload error")}
	res := newUC(deployRepo, svcRepo, nginx, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx reload fails")
	}
}

func TestActivate_UpdateActiveSlotFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	svcRepo := &stubServiceRepo{updateErr: errors.New("db down")}
	res := newUC(deployRepo, svcRepo, &stubNginxClient{}, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_SetStatusFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy, setStatusErr: errors.New("db down")}
	res := newUC(deployRepo, &stubServiceRepo{}, &stubNginxClient{}, &stubLockRepo{}).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_ReleaseLockFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	lockRepo := &stubLockRepo{releaseErr: errors.New("db down")}
	res := newUC(deployRepo, &stubServiceRepo{}, &stubNginxClient{}, lockRepo).Execute(validInput)
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

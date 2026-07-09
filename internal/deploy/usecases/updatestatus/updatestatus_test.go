package updatestatus_test

import (
	"errors"
	"testing"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/updatestatus"
)

type stubDeployRepo struct {
	deploy deploydomain.Deploy
	getErr error
	setErr error

	setCalled bool
	setStatus deploydomain.DeployStatus
	setSlot   *deploydomain.Slot
}

func (r *stubDeployRepo) GetByID(id string) (deploydomain.Deploy, error) {
	return r.deploy, r.getErr
}

func (r *stubDeployRepo) SetStatus(deployID string, newStatus deploydomain.DeployStatus, slot *deploydomain.Slot) error {
	r.setCalled = true
	r.setStatus = newStatus
	r.setSlot = slot
	return r.setErr
}

type stubLockRepo struct {
	createErr  error
	releaseErr error

	createCalled  bool
	expiresAt     time.Time
	releaseCalled bool
}

func (r *stubLockRepo) CreateLock(deployID string, expiresAt time.Time) error {
	r.createCalled = true
	r.expiresAt = expiresAt
	return r.createErr
}

func (r *stubLockRepo) ReleaseLock(deployID string) error {
	r.releaseCalled = true
	return r.releaseErr
}

var (
	slotBlue    = deploydomain.SlotBlue
	pendingDep  = deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusPending}
	buildingDep = deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusBuilding}
	activeDep   = deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusActive}
	failedDep   = deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusFailed}
)

func TestUpdateStatus_Validation(t *testing.T) {
	cases := []struct {
		name  string
		input updatestatus.UpdateStatusInput
	}{
		{"empty deploy ID", updatestatus.UpdateStatusInput{NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue}},
		{"unknown status", updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: "unknown"}},
		{"building with nil slot", updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := updatestatus.New(&stubDeployRepo{deploy: pendingDep}, &stubLockRepo{})
			res := uc.Execute(tc.input)
			if res.IsOk() {
				t.Fatal("expected error, got ok")
			}
			if !errors.Is(res.Err, updatestatus.ErrValidation) {
				t.Fatalf("expected ErrValidation, got %v", res.Err)
			}
		})
	}
}

func TestUpdateStatus_DeployNotFound(t *testing.T) {
	uc := updatestatus.New(&stubDeployRepo{getErr: deploydomain.ErrNotFound}, &stubLockRepo{})

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, updatestatus.ErrDeployNotFound) {
		t.Fatalf("expected ErrDeployNotFound, got %v", res.Err)
	}
}

func TestUpdateStatus_GetByIDError(t *testing.T) {
	uc := updatestatus.New(&stubDeployRepo{getErr: errors.New("db down")}, &stubLockRepo{})

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, updatestatus.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestUpdateStatus_InvalidTransitions(t *testing.T) {
	cases := []struct {
		name   string
		deploy deploydomain.Deploy
		status deploydomain.DeployStatus
		slot   *deploydomain.Slot
	}{
		{"pending → active", pendingDep, deploydomain.StatusActive, nil},
		{"pending → failed", pendingDep, deploydomain.StatusFailed, nil},
		{"building → pending", buildingDep, deploydomain.StatusPending, nil},
		{"failed → building", failedDep, deploydomain.StatusBuilding, &slotBlue},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			uc := updatestatus.New(&stubDeployRepo{deploy: tc.deploy}, &stubLockRepo{})
			res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: tc.status, Slot: tc.slot})
			if res.IsOk() {
				t.Fatal("expected error, got ok")
			}
			if !errors.Is(res.Err, updatestatus.ErrInvalidTransition) {
				t.Fatalf("expected ErrInvalidTransition, got %v", res.Err)
			}
		})
	}
}

func TestUpdateStatus_PendingToBuilding(t *testing.T) {
	before := time.Now()
	deployRepo := &stubDeployRepo{deploy: pendingDep}
	lockRepo := &stubLockRepo{}
	uc := updatestatus.New(deployRepo, lockRepo)

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if deployRepo.setStatus != deploydomain.StatusBuilding {
		t.Fatalf("expected SetStatus building, got %v", deployRepo.setStatus)
	}
	if deployRepo.setSlot == nil || *deployRepo.setSlot != slotBlue {
		t.Fatalf("expected slot blue, got %v", deployRepo.setSlot)
	}
	if !lockRepo.createCalled {
		t.Fatal("expected CreateLock to be called")
	}
	if lockRepo.expiresAt.Before(before.Add(updatestatus.LockDuration)) {
		t.Fatalf("expected expires_at >= now + LockDuration, got %v", lockRepo.expiresAt)
	}
	if lockRepo.releaseCalled {
		t.Fatal("expected ReleaseLock NOT to be called")
	}
	if res.Value.Deploy.Status != deploydomain.StatusBuilding {
		t.Fatalf("expected status building, got %v", res.Value.Deploy.Status)
	}
	if res.Value.Deploy.StartedAt == nil {
		t.Fatal("expected StartedAt to be set")
	}
	if res.Value.Deploy.FinishedAt != nil {
		t.Fatal("expected FinishedAt to be nil")
	}
}

func TestUpdateStatus_BuildingToActive(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDep}
	lockRepo := &stubLockRepo{}
	uc := updatestatus.New(deployRepo, lockRepo)

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusActive})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if deployRepo.setStatus != deploydomain.StatusActive {
		t.Fatalf("expected SetStatus active, got %v", deployRepo.setStatus)
	}
	if !lockRepo.releaseCalled {
		t.Fatal("expected ReleaseLock to be called")
	}
	if lockRepo.createCalled {
		t.Fatal("expected CreateLock NOT to be called")
	}
	if res.Value.Deploy.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
}

func TestUpdateStatus_BuildingToFailed(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDep}
	lockRepo := &stubLockRepo{}
	uc := updatestatus.New(deployRepo, lockRepo)

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusFailed})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if deployRepo.setStatus != deploydomain.StatusFailed {
		t.Fatalf("expected SetStatus failed, got %v", deployRepo.setStatus)
	}
	if !lockRepo.releaseCalled {
		t.Fatal("expected ReleaseLock to be called")
	}
	if lockRepo.createCalled {
		t.Fatal("expected CreateLock NOT to be called")
	}
	if res.Value.Deploy.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
}

func TestUpdateStatus_ActiveToRolledBack(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: activeDep}
	lockRepo := &stubLockRepo{}
	uc := updatestatus.New(deployRepo, lockRepo)

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusRolledBack})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if deployRepo.setStatus != deploydomain.StatusRolledBack {
		t.Fatalf("expected SetStatus rolled_back, got %v", deployRepo.setStatus)
	}
	if lockRepo.createCalled {
		t.Fatal("expected CreateLock NOT to be called — deploy was active, not building")
	}
	if lockRepo.releaseCalled {
		t.Fatal("expected ReleaseLock NOT to be called — no lock held on an active deploy")
	}
	if res.Value.Deploy.Status != deploydomain.StatusRolledBack {
		t.Fatalf("expected status rolled_back, got %v", res.Value.Deploy.Status)
	}
	if res.Value.Deploy.FinishedAt == nil {
		t.Fatal("expected FinishedAt to be set")
	}
}

func TestUpdateStatus_SetStatusError(t *testing.T) {
	uc := updatestatus.New(&stubDeployRepo{deploy: pendingDep, setErr: errors.New("db error")}, &stubLockRepo{})

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, updatestatus.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestUpdateStatus_CreateLockError(t *testing.T) {
	uc := updatestatus.New(&stubDeployRepo{deploy: pendingDep}, &stubLockRepo{createErr: errors.New("lock failed")})

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusBuilding, Slot: &slotBlue})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, updatestatus.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestUpdateStatus_ReleaseLockError(t *testing.T) {
	uc := updatestatus.New(&stubDeployRepo{deploy: buildingDep}, &stubLockRepo{releaseErr: errors.New("release failed")})

	res := uc.Execute(updatestatus.UpdateStatusInput{DeployID: "dep-1", NewStatus: deploydomain.StatusActive})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, updatestatus.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

package refreshlock_test

import (
	"errors"
	"testing"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/refreshlock"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/updatestatus"
)

type stubDeployRepo struct {
	deploy deploydomain.Deploy
	err    error
}

func (r *stubDeployRepo) GetByID(id string) (deploydomain.Deploy, error) {
	return r.deploy, r.err
}

type stubLockRepo struct {
	err           error
	called        bool
	newExpiresAt  time.Time
}

func (r *stubLockRepo) RefreshLock(deployID string, newExpiresAt time.Time) error {
	r.called = true
	r.newExpiresAt = newExpiresAt
	return r.err
}

var buildingDep = deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusBuilding}

func TestRefreshLock_EmptyDeployID(t *testing.T) {
	uc := refreshlock.New(&stubDeployRepo{}, &stubLockRepo{})
	res := uc.Execute(refreshlock.RefreshLockInput{})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, refreshlock.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestRefreshLock_DeployNotFound(t *testing.T) {
	uc := refreshlock.New(&stubDeployRepo{err: deploydomain.ErrNotFound}, &stubLockRepo{})
	res := uc.Execute(refreshlock.RefreshLockInput{DeployID: "dep-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, refreshlock.ErrDeployNotFound) {
		t.Fatalf("expected ErrDeployNotFound, got %v", res.Err)
	}
}

func TestRefreshLock_GetByIDError(t *testing.T) {
	uc := refreshlock.New(&stubDeployRepo{err: errors.New("db down")}, &stubLockRepo{})
	res := uc.Execute(refreshlock.RefreshLockInput{DeployID: "dep-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, refreshlock.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestRefreshLock_NotBuilding(t *testing.T) {
	cases := []deploydomain.DeployStatus{
		deploydomain.StatusPending,
		deploydomain.StatusActive,
		deploydomain.StatusFailed,
		deploydomain.StatusRolledBack,
	}
	for _, status := range cases {
		dep := deploydomain.Deploy{ID: "dep-1", Status: status}
		uc := refreshlock.New(&stubDeployRepo{deploy: dep}, &stubLockRepo{})
		res := uc.Execute(refreshlock.RefreshLockInput{DeployID: "dep-1"})
		if res.IsOk() {
			t.Fatalf("status %v: expected error, got ok", status)
		}
		if !errors.Is(res.Err, refreshlock.ErrInvalidState) {
			t.Fatalf("status %v: expected ErrInvalidState, got %v", status, res.Err)
		}
	}
}

func TestRefreshLock_Success(t *testing.T) {
	before := time.Now()
	lockRepo := &stubLockRepo{}
	uc := refreshlock.New(&stubDeployRepo{deploy: buildingDep}, lockRepo)

	res := uc.Execute(refreshlock.RefreshLockInput{DeployID: "dep-1"})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if !lockRepo.called {
		t.Fatal("expected RefreshLock to be called")
	}
	expected := before.Add(updatestatus.LockDuration)
	if lockRepo.newExpiresAt.Before(expected) {
		t.Fatalf("expected new expires_at >= now + LockDuration, got %v", lockRepo.newExpiresAt)
	}
}

func TestRefreshLock_LockRepoError(t *testing.T) {
	uc := refreshlock.New(&stubDeployRepo{deploy: buildingDep}, &stubLockRepo{err: errors.New("db error")})
	res := uc.Execute(refreshlock.RefreshLockInput{DeployID: "dep-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, refreshlock.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

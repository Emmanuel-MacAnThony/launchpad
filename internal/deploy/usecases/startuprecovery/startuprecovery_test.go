package startuprecovery_test

import (
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/startuprecovery"
)

type stubDeployRepo struct {
	count int64
	err   error
}

func (r *stubDeployRepo) StartupRecovery() (int64, error) {
	return r.count, r.err
}

func TestStartupRecovery_NothingBuilding(t *testing.T) {
	res := startuprecovery.New(&stubDeployRepo{count: 0}).Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Count != 0 {
		t.Errorf("expected count=0, got %d", res.Value.Count)
	}
}

func TestStartupRecovery_ResetsBuilding(t *testing.T) {
	res := startuprecovery.New(&stubDeployRepo{count: 3}).Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Count != 3 {
		t.Errorf("expected count=3, got %d", res.Value.Count)
	}
}

func TestStartupRecovery_DBError(t *testing.T) {
	res := startuprecovery.New(&stubDeployRepo{err: errors.New("db down")}).Execute()
	if !errors.Is(res.Err, startuprecovery.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

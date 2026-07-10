package recoverybuild_test

import (
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/recoverybuild"
)

type stubDeployRepo struct {
	count int64
	err   error
}

func (r *stubDeployRepo) ResetExpiredBuilding() (int64, error) {
	return r.count, r.err
}

func TestRecoverBuild_NoneExpired(t *testing.T) {
	res := recoverybuild.New(&stubDeployRepo{count: 0}).Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Count != 0 {
		t.Errorf("expected count=0, got %d", res.Value.Count)
	}
}

func TestRecoverBuild_ResetsExpired(t *testing.T) {
	res := recoverybuild.New(&stubDeployRepo{count: 2}).Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Count != 2 {
		t.Errorf("expected count=2, got %d", res.Value.Count)
	}
}

func TestRecoverBuild_DBError(t *testing.T) {
	res := recoverybuild.New(&stubDeployRepo{err: errors.New("db down")}).Execute()
	if !errors.Is(res.Err, recoverybuild.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

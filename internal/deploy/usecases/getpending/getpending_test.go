package getpending_test

import (
	"errors"
	"testing"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/getpending"
)

type stubRepo struct {
	deploys []deploydomain.Deploy
	err     error
}

func (r *stubRepo) ListPending() ([]deploydomain.Deploy, error) {
	return r.deploys, r.err
}

func TestGetPending_EmptyQueue(t *testing.T) {
	repo := &stubRepo{deploys: []deploydomain.Deploy{}}
	uc := getpending.New(repo)

	res := uc.Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if len(res.Value.Deploys) != 0 {
		t.Fatalf("expected empty deploys, got %d", len(res.Value.Deploys))
	}
}

func TestGetPending_ReturnsPendingDeploys(t *testing.T) {
	now := time.Now()
	deploys := []deploydomain.Deploy{
		{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusPending, CreatedAt: now.Add(-2 * time.Minute)},
		{ID: "dep-2", ServiceID: "svc-2", Status: deploydomain.StatusPending, CreatedAt: now.Add(-1 * time.Minute)},
	}
	repo := &stubRepo{deploys: deploys}
	uc := getpending.New(repo)

	res := uc.Execute()
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if len(res.Value.Deploys) != 2 {
		t.Fatalf("expected 2 deploys, got %d", len(res.Value.Deploys))
	}
	if res.Value.Deploys[0].ID != "dep-1" {
		t.Fatalf("expected dep-1 first (oldest), got %v", res.Value.Deploys[0].ID)
	}
	if res.Value.Deploys[1].ID != "dep-2" {
		t.Fatalf("expected dep-2 second, got %v", res.Value.Deploys[1].ID)
	}
}

func TestGetPending_RepoError(t *testing.T) {
	repo := &stubRepo{err: errors.New("db connection lost")}
	uc := getpending.New(repo)

	res := uc.Execute()
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, getpending.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

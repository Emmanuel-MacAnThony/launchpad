package list_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/list"
)

// --- fakes ---

type fakeRepo struct {
	svcs []domain.Service
	err  error
}

func (r *fakeRepo) ListAll() ([]domain.Service, error) {
	return r.svcs, r.err
}

// --- tests ---

func TestList_HappyPath(t *testing.T) {
	now := time.Now().UTC()
	repo := &fakeRepo{svcs: []domain.Service{
		{ID: "svc-1", Name: "app-one", RepoURL: "git@github.com:u/a.git", Domain: "a.com", HealthCheckURL: "http://a.com/health", Host: "1.1.1.1", SSHUser: "ubuntu", SSHKeyPath: "/key", CreatedAt: now},
		{ID: "svc-2", Name: "app-two", RepoURL: "git@github.com:u/b.git", Domain: "b.com", HealthCheckURL: "http://b.com/health", Host: "2.2.2.2", SSHUser: "ubuntu", SSHKeyPath: "/key", CreatedAt: now},
	}}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if len(res.Value.Services) != 2 {
		t.Fatalf("expected 2 services, got %d", len(res.Value.Services))
	}
	if res.Value.Services[0].ID != "svc-1" || res.Value.Services[1].ID != "svc-2" {
		t.Error("services not mapped correctly")
	}
}

func TestList_Empty(t *testing.T) {
	repo := &fakeRepo{svcs: []domain.Service{}}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if len(res.Value.Services) != 0 {
		t.Errorf("expected empty slice, got %d items", len(res.Value.Services))
	}
}

func TestList_RepoFails(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db down")}
	uc := list.New(repo)

	res := uc.Execute(list.ListInput{})

	if !errors.Is(res.Err, list.ErrInternalError) {
		t.Fatalf("expected ErrInternalError, got %v", res.Err)
	}
}

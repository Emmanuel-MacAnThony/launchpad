package get_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
)

// --- fakes ---

type fakeRepo struct {
	svc domain.Service
	err error
	calls []string
}

func (r *fakeRepo) GetByID(id string) (domain.Service, error) {
	r.calls = append(r.calls, id)
	return r.svc, r.err
}

// --- helpers ---

func stubService() domain.Service {
	return domain.Service{
		ID:             "svc-1",
		Name:           "my-app",
		RepoURL:        "git@github.com:user/my-app.git",
		Domain:         "my-app.com",
		HealthCheckURL: "http://my-app.com/health",
		Host:           "192.168.1.1",
		SSHUser:        "ubuntu",
		SSHKeyPath:     "/home/ubuntu/.ssh/id_rsa",
		CreatedAt:      time.Now().UTC(),
	}
}

// --- tests ---

func TestGet_HappyPath(t *testing.T) {
	svc := stubService()
	repo := &fakeRepo{svc: svc}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "svc-1"})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if res.Value.ID != svc.ID {
		t.Errorf("expected id %s, got %s", svc.ID, res.Value.ID)
	}
	if res.Value.Name != svc.Name {
		t.Errorf("expected name %s, got %s", svc.Name, res.Value.Name)
	}
	if res.Value.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
	if len(repo.calls) != 1 || repo.calls[0] != "svc-1" {
		t.Error("expected repo to be called with the given id")
	}
}

func TestGet_InvalidInput(t *testing.T) {
	repo := &fakeRepo{}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: ""})

	if !errors.Is(res.Err, get.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
	if len(repo.calls) != 0 {
		t.Error("expected repo not to be called")
	}
}

func TestGet_NotFound(t *testing.T) {
	repo := &fakeRepo{err: domain.ErrNotFound}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "missing"})

	if !errors.Is(res.Err, get.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", res.Err)
	}
}

func TestGet_InternalError(t *testing.T) {
	repo := &fakeRepo{err: errors.New("db connection lost")}
	uc := get.New(repo)

	res := uc.Execute(get.GetInput{ID: "svc-1"})

	if !errors.Is(res.Err, get.ErrInternalError) {
		t.Fatalf("expected ErrInternalError, got %v", res.Err)
	}
}

package update_test

import (
	"errors"
	"testing"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/update"
)

// --- fakes ---

type fakeRepo struct {
	svc       domain.Service
	getErr    error
	updateErr error
	getCalls  []string
	updated   *updateCall
}

type updateCall struct {
	id             string
	name           string
	healthCheckURL string
}

func (r *fakeRepo) GetByID(id string) (domain.Service, error) {
	r.getCalls = append(r.getCalls, id)
	return r.svc, r.getErr
}

func (r *fakeRepo) Update(id, name, healthCheckURL string) error {
	r.updated = &updateCall{id: id, name: name, healthCheckURL: healthCheckURL}
	return r.updateErr
}

// --- helpers ---

func stubService() domain.Service {
	return domain.Service{
		ID:             "svc-1",
		Name:           "old-name",
		RepoURL:        "git@github.com:user/app.git",
		Domain:         "app.com",
		HealthCheckURL: "http://app.com/health",
		Host:           "192.168.1.1",
		SSHUser:        "ubuntu",
		SSHKeyPath:     "/home/ubuntu/.ssh/id_rsa",
		CreatedAt:      time.Now().UTC(),
	}
}

// --- tests ---

func TestUpdate_BothFields(t *testing.T) {
	repo := &fakeRepo{svc: stubService()}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{
		ID:             "svc-1",
		Name:           "new-name",
		HealthCheckURL: "http://app.com/healthz",
	})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if res.Value.Name != "new-name" {
		t.Errorf("expected name new-name, got %s", res.Value.Name)
	}
	if res.Value.HealthCheckURL != "http://app.com/healthz" {
		t.Errorf("expected updated health_check_url, got %s", res.Value.HealthCheckURL)
	}
	if repo.updated == nil {
		t.Fatal("expected Update to be called")
	}
	if repo.updated.name != "new-name" || repo.updated.healthCheckURL != "http://app.com/healthz" {
		t.Error("Update called with wrong args")
	}
}

func TestUpdate_OnlyName(t *testing.T) {
	svc := stubService()
	repo := &fakeRepo{svc: svc}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "svc-1", Name: "new-name"})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if res.Value.Name != "new-name" {
		t.Errorf("expected name new-name, got %s", res.Value.Name)
	}
	// health_check_url must be preserved from fetched state
	if res.Value.HealthCheckURL != svc.HealthCheckURL {
		t.Errorf("expected health_check_url unchanged, got %s", res.Value.HealthCheckURL)
	}
	if repo.updated.healthCheckURL != svc.HealthCheckURL {
		t.Error("Update called with wrong health_check_url")
	}
}

func TestUpdate_OnlyHealthCheckURL(t *testing.T) {
	svc := stubService()
	repo := &fakeRepo{svc: svc}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "svc-1", HealthCheckURL: "http://app.com/healthz"})

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if res.Value.HealthCheckURL != "http://app.com/healthz" {
		t.Errorf("expected updated health_check_url, got %s", res.Value.HealthCheckURL)
	}
	// name must be preserved from fetched state
	if res.Value.Name != svc.Name {
		t.Errorf("expected name unchanged, got %s", res.Value.Name)
	}
	if repo.updated.name != svc.Name {
		t.Error("Update called with wrong name")
	}
}

func TestUpdate_InvalidInput_EmptyID(t *testing.T) {
	repo := &fakeRepo{}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{Name: "new-name"})

	if !errors.Is(res.Err, update.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
	if len(repo.getCalls) != 0 {
		t.Error("expected repo not to be called")
	}
}

func TestUpdate_InvalidInput_NoFields(t *testing.T) {
	repo := &fakeRepo{}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "svc-1"})

	if !errors.Is(res.Err, update.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
	if len(repo.getCalls) != 0 {
		t.Error("expected repo not to be called")
	}
}

func TestUpdate_NotFound(t *testing.T) {
	repo := &fakeRepo{getErr: domain.ErrNotFound}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "missing", Name: "new-name"})

	if !errors.Is(res.Err, update.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", res.Err)
	}
	if repo.updated != nil {
		t.Error("expected Update not to be called")
	}
}

func TestUpdate_FetchFails(t *testing.T) {
	repo := &fakeRepo{getErr: errors.New("db connection lost")}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "svc-1", Name: "new-name"})

	if !errors.Is(res.Err, update.ErrInternalError) {
		t.Fatalf("expected ErrInternalError, got %v", res.Err)
	}
	if repo.updated != nil {
		t.Error("expected Update not to be called")
	}
}

func TestUpdate_PersistFails(t *testing.T) {
	repo := &fakeRepo{svc: stubService(), updateErr: errors.New("db error")}
	uc := update.New(repo)

	res := uc.Execute(update.UpdateInput{ID: "svc-1", Name: "new-name"})

	if !errors.Is(res.Err, update.ErrPersistFailed) {
		t.Fatalf("expected ErrPersistFailed, got %v", res.Err)
	}
}

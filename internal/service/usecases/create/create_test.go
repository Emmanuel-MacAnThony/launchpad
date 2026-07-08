package create_test

import (
	"errors"
	"testing"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
)

// --- fakes ---

type fakeRepo struct {
	existsByDomain bool
	saveErr        error
	saved          *domain.Service
	deleted        []string
}

func (r *fakeRepo) ExistsByDomain(_ string) (bool, error) { return r.existsByDomain, nil }
func (r *fakeRepo) Save(s domain.Service) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = &s
	return nil
}
func (r *fakeRepo) Delete(id string) error {
	r.deleted = append(r.deleted, id)
	return nil
}

type fakeNginx struct {
	writeErr  error
	reloadErr error
	written   []string
	deleted   []string
	reloaded  int
}

func (n *fakeNginx) WriteConfig(serviceID string, opts ...func(*create.NginxConfig)) error {
	if n.writeErr != nil {
		return n.writeErr
	}
	n.written = append(n.written, serviceID)
	return nil
}
func (n *fakeNginx) ReloadNginx() error {
	if n.reloadErr != nil {
		return n.reloadErr
	}
	n.reloaded++
	return nil
}
func (n *fakeNginx) DeleteConfig(serviceID string) error {
	n.deleted = append(n.deleted, serviceID)
	return nil
}

type fakeSSHClient struct {
	free bool
	err  error
}

func (s *fakeSSHClient) AreFree(_ ...int) (bool, error) { return s.free, s.err }

type fakeSSHFactory struct{ client create.SSHClient }

func (f *fakeSSHFactory) New(_, _, _ string) create.SSHClient { return f.client }

// --- helpers ---

func validInput() create.CreateInput {
	return create.CreateInput{
		Name:           "my-app",
		RepoURL:        "git@github.com:user/my-app.git",
		Domain:         "my-app.com",
		HealthCheckURL: "http://my-app.com/health",
		WebhookSecret:  "secret",
		Host:           "192.168.1.1",
		SSHUser:        "ubuntu",
		SSHKeyPath:     "/home/ubuntu/.ssh/id_rsa",
		BluePort:       3001,
		GreenPort:      3002,
		ContainerPort:  8000,
	}
}

func happySSH() *fakeSSHFactory {
	return &fakeSSHFactory{client: &fakeSSHClient{free: true}}
}

// --- tests ---

func TestCreate_HappyPath(t *testing.T) {
	repo := &fakeRepo{}
	nginx := &fakeNginx{}
	uc := create.New(repo, nginx, happySSH())

	res := uc.Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected no error, got %v", res.Err)
	}
	if res.Value.ID == "" {
		t.Error("expected id to be generated")
	}
	if res.Value.CreatedAt.IsZero() {
		t.Error("expected created_at to be set")
	}
	if res.Value.BluePort != 3001 || res.Value.GreenPort != 3002 {
		t.Error("expected ports to be persisted")
	}
	if res.Value.ActiveSlot != nil {
		t.Error("expected active_slot to be nil on creation")
	}
	if repo.saved == nil {
		t.Error("expected service to be persisted")
	}
	if len(nginx.written) != 1 {
		t.Error("expected nginx config to be written")
	}
	if nginx.reloaded != 1 {
		t.Error("expected nginx to be reloaded")
	}
}

func TestCreate_InvalidInput_MissingField(t *testing.T) {
	repo := &fakeRepo{}
	uc := create.New(repo, &fakeNginx{}, happySSH())

	input := validInput()
	input.Name = ""

	res := uc.Execute(input)

	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
	if repo.saved != nil {
		t.Error("expected nothing persisted")
	}
}

func TestCreate_InvalidInput_PortsEqual(t *testing.T) {
	repo := &fakeRepo{}
	uc := create.New(repo, &fakeNginx{}, happySSH())

	input := validInput()
	input.BluePort = 3001
	input.GreenPort = 3001

	res := uc.Execute(input)

	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
	if repo.saved != nil {
		t.Error("expected nothing persisted")
	}
}

func TestCreate_DomainTaken(t *testing.T) {
	repo := &fakeRepo{existsByDomain: true}
	uc := create.New(repo, &fakeNginx{}, happySSH())

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrDomainTaken) {
		t.Fatalf("expected ErrDomainTaken, got %v", res.Err)
	}
	if repo.saved != nil {
		t.Error("expected nothing persisted")
	}
}

func TestCreate_PortConflict(t *testing.T) {
	ssh := &fakeSSHFactory{client: &fakeSSHClient{free: false}}
	uc := create.New(&fakeRepo{}, &fakeNginx{}, ssh)

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrPortConflict) {
		t.Fatalf("expected ErrPortConflict, got %v", res.Err)
	}
}

func TestCreate_PortScanFailed(t *testing.T) {
	ssh := &fakeSSHFactory{client: &fakeSSHClient{err: errors.New("connection refused")}}
	uc := create.New(&fakeRepo{}, &fakeNginx{}, ssh)

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrPortScanFailed) {
		t.Fatalf("expected ErrPortScanFailed, got %v", res.Err)
	}
}

func TestCreate_PersistFails(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("db error")}
	uc := create.New(repo, &fakeNginx{}, happySSH())

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrPersistFailed) {
		t.Fatalf("expected ErrPersistFailed, got %v", res.Err)
	}
}

func TestCreate_NginxConfigFails_RollsBackDB(t *testing.T) {
	repo := &fakeRepo{}
	nginx := &fakeNginx{writeErr: errors.New("disk error")}
	uc := create.New(repo, nginx, happySSH())

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrNginxConfigFailed) {
		t.Fatalf("expected ErrNginxConfigFailed, got %v", res.Err)
	}
	if len(repo.deleted) != 1 {
		t.Error("expected service to be deleted from DB on rollback")
	}
	if nginx.reloaded != 0 {
		t.Error("expected nginx not to be reloaded")
	}
}

func TestCreate_NginxReloadFails_RollsBackDBAndFiles(t *testing.T) {
	repo := &fakeRepo{}
	nginx := &fakeNginx{reloadErr: errors.New("reload error")}
	uc := create.New(repo, nginx, happySSH())

	res := uc.Execute(validInput())

	if !errors.Is(res.Err, create.ErrNginxReloadFailed) {
		t.Fatalf("expected ErrNginxReloadFailed, got %v", res.Err)
	}
	if len(repo.deleted) != 1 {
		t.Error("expected service to be deleted from DB on rollback")
	}
	if len(nginx.deleted) != 1 {
		t.Error("expected nginx config files to be deleted on rollback")
	}
}

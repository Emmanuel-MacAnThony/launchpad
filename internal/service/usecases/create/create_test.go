package create_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
)

// --- fakes ---

type fakeRepo struct {
	existsByDomain bool
	existsErr      error
	saveErr        error
	saved          *domain.Service
}

func (r *fakeRepo) ExistsByDomain(_ string) (bool, error) { return r.existsByDomain, r.existsErr }
func (r *fakeRepo) Save(s domain.Service) error {
	if r.saveErr != nil {
		return r.saveErr
	}
	r.saved = &s
	return nil
}
func (r *fakeRepo) Delete(_ string) error { return nil }

// stubSSHExecutor lets each test control what Run returns per command.
type stubSSHExecutor struct {
	runFn func(cmd string) (create.SSHResult, error)
}

func (s *stubSSHExecutor) Run(cmd string) (create.SSHResult, error) {
	if s.runFn != nil {
		return s.runFn(cmd)
	}
	return create.SSHResult{}, nil
}
func (s *stubSSHExecutor) Close() error { return nil }

type stubSSHFactory struct {
	executor create.SSHExecutor
	dialErr  error
}

func (f *stubSSHFactory) NewExecutor(_ create.SSHConfig) (create.SSHExecutor, error) {
	return f.executor, f.dialErr
}

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
		SSHKey:     "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key\n-----END OPENSSH PRIVATE KEY-----",
		BluePort:       3001,
		GreenPort:      3002,
		ContainerPort:  8000,
	}
}

// happyExecutor succeeds all bootstrap commands and reports no ports in use.
func happyExecutor() *stubSSHExecutor {
	return &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			return create.SSHResult{}, nil
		},
	}
}

func happyFactory() *stubSSHFactory {
	return &stubSSHFactory{executor: happyExecutor()}
}

// --- tests ---

func TestCreate_HappyPath(t *testing.T) {
	repo := &fakeRepo{}
	uc := create.New(repo, happyFactory())

	res := uc.Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.ID == "" {
		t.Error("expected ID to be generated")
	}
	if res.Value.CreatedAt.IsZero() {
		t.Error("expected CreatedAt to be set")
	}
	if res.Value.BluePort != 3001 || res.Value.GreenPort != 3002 {
		t.Error("expected ports to be persisted")
	}
	if res.Value.ActiveSlot != nil {
		t.Error("expected ActiveSlot to be nil on creation")
	}
	if repo.saved == nil {
		t.Error("expected service to be persisted")
	}
}

func TestCreate_InvalidInput_MissingName(t *testing.T) {
	input := validInput()
	input.Name = ""

	res := create.New(&fakeRepo{}, happyFactory()).Execute(input)

	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestCreate_InvalidInput_PortsEqual(t *testing.T) {
	input := validInput()
	input.GreenPort = input.BluePort

	res := create.New(&fakeRepo{}, happyFactory()).Execute(input)

	if !errors.Is(res.Err, create.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", res.Err)
	}
}

func TestCreate_DomainTaken(t *testing.T) {
	res := create.New(&fakeRepo{existsByDomain: true}, happyFactory()).Execute(validInput())

	if !errors.Is(res.Err, create.ErrDomainTaken) {
		t.Fatalf("expected ErrDomainTaken, got %v", res.Err)
	}
}

func TestCreate_DomainCheckError(t *testing.T) {
	repo := &fakeRepo{existsErr: errors.New("db down")}
	res := create.New(repo, happyFactory()).Execute(validInput())

	if !errors.Is(res.Err, create.ErrPersistFailed) {
		t.Fatalf("expected ErrPersistFailed, got %v", res.Err)
	}
}

func TestCreate_SSHConnectionFailed(t *testing.T) {
	factory := &stubSSHFactory{dialErr: errors.New("connection refused")}
	res := create.New(&fakeRepo{}, factory).Execute(validInput())

	if !errors.Is(res.Err, create.ErrSSHFailed) {
		t.Fatalf("expected ErrSSHFailed, got %v", res.Err)
	}
}

func TestCreate_DockerNotInstalled(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "docker") {
				return create.SSHResult{}, errors.New("exit status 1")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrDockerNotInstalled) {
		t.Fatalf("expected ErrDockerNotInstalled, got %v", res.Err)
	}
}

func TestCreate_NginxNotInstalled(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "nginx -v") {
				return create.SSHResult{}, errors.New("exit status 127")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrNginxNotInstalled) {
		t.Fatalf("expected ErrNginxNotInstalled, got %v", res.Err)
	}
}

func TestCreate_BootstrapMkdirFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "mkdir") {
				return create.SSHResult{}, errors.New("permission denied")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrBootstrapFailed) {
		t.Fatalf("expected ErrBootstrapFailed, got %v", res.Err)
	}
}

func TestCreate_BootstrapIncludeFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "grep") {
				return create.SSHResult{}, errors.New("permission denied")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrBootstrapFailed) {
		t.Fatalf("expected ErrBootstrapFailed, got %v", res.Err)
	}
}

func TestCreate_BootstrapNginxTFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if cmd == "nginx -t" {
				return create.SSHResult{Stderr: "nginx: configuration file test failed"}, errors.New("exit status 1")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrBootstrapFailed) {
		t.Fatalf("expected ErrBootstrapFailed, got %v", res.Err)
	}
}

func TestCreate_PortConflict(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "ss") {
				// bluePort 3001 is in use
				return create.SSHResult{Stdout: "LISTEN 0 128 0.0.0.0:3001 0.0.0.0:*"}, nil
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrPortConflict) {
		t.Fatalf("expected ErrPortConflict, got %v", res.Err)
	}
}

func TestCreate_PortScanFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (create.SSHResult, error) {
			if strings.HasPrefix(cmd, "ss") {
				return create.SSHResult{}, errors.New("connection lost")
			}
			return create.SSHResult{}, nil
		},
	}
	res := create.New(&fakeRepo{}, &stubSSHFactory{executor: ex}).Execute(validInput())

	if !errors.Is(res.Err, create.ErrPortScanFailed) {
		t.Fatalf("expected ErrPortScanFailed, got %v", res.Err)
	}
}

func TestCreate_PersistFails(t *testing.T) {
	repo := &fakeRepo{saveErr: errors.New("db error")}
	res := create.New(repo, happyFactory()).Execute(validInput())

	if !errors.Is(res.Err, create.ErrPersistFailed) {
		t.Fatalf("expected ErrPersistFailed, got %v", res.Err)
	}
}

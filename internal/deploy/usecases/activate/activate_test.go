package activate_test

import (
	"errors"
	"strings"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/activate"
	servicedomain "github.com/Emmanuel-MacAnThony/launchpad/internal/service/domain"
)

// --- stubs ---

type stubDeployRepo struct {
	deploy       deploydomain.Deploy
	getErr       error
	setStatusErr error
	statusSet    deploydomain.DeployStatus
}

func (r *stubDeployRepo) GetByID(_ string) (deploydomain.Deploy, error) {
	return r.deploy, r.getErr
}

func (r *stubDeployRepo) SetStatus(_ string, status deploydomain.DeployStatus, _ *deploydomain.Slot) error {
	r.statusSet = status
	return r.setStatusErr
}

type stubServiceRepo struct {
	updateErr   error
	updatedSlot servicedomain.Slot
}

func (r *stubServiceRepo) UpdateActiveSlot(_ string, slot servicedomain.Slot) error {
	r.updatedSlot = slot
	return r.updateErr
}

type stubLockRepo struct {
	releaseErr    error
	releaseCalled bool
}

func (l *stubLockRepo) ReleaseLock(_ string) error {
	l.releaseCalled = true
	return l.releaseErr
}

type stubSSHExecutor struct {
	runFn      func(cmd string) (activate.SSHResult, error)
	uploadErr  error
	uploadedTo string
}

func (s *stubSSHExecutor) Run(cmd string) (activate.SSHResult, error) {
	if s.runFn != nil {
		return s.runFn(cmd)
	}
	return activate.SSHResult{}, nil
}

func (s *stubSSHExecutor) Upload(_, remote string) error {
	s.uploadedTo = remote
	return s.uploadErr
}

func (s *stubSSHExecutor) Close() error { return nil }

type stubSSHFactory struct {
	executor activate.SSHExecutor
	dialErr  error
}

func (f *stubSSHFactory) NewExecutor(_ activate.SSHConfig) (activate.SSHExecutor, error) {
	return f.executor, f.dialErr
}

// --- fixtures ---

var buildingDeploy = deploydomain.Deploy{
	ID:        "dep-1",
	ServiceID: "svc-1",
	Status:    deploydomain.StatusBuilding,
}

func validInput() activate.ActivateInput {
	return activate.ActivateInput{
		DeployID:   "dep-1",
		ServiceID:  "svc-1",
		Slot:       deploydomain.SlotBlue,
		Host:       "192.168.1.1",
		SSHUser:    "ubuntu",
		SSHKey:     "-----BEGIN OPENSSH PRIVATE KEY-----\nfake-key\n-----END OPENSSH PRIVATE KEY-----",
		Domain:     "my-app.com",
		ActivePort: 3001,
	}
}

func happyExecutor() *stubSSHExecutor {
	return &stubSSHExecutor{}
}

func happyFactory() *stubSSHFactory {
	return &stubSSHFactory{executor: happyExecutor()}
}

func newUC(deployRepo *stubDeployRepo, svcRepo *stubServiceRepo, factory activate.SSHExecutorFactory, lockRepo *stubLockRepo) *activate.UseCase {
	return activate.New(factory, svcRepo, deployRepo, lockRepo)
}

// --- tests ---

func TestActivate_HappyPath(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy}
	svcRepo := &stubServiceRepo{}
	ex := happyExecutor()
	lockRepo := &stubLockRepo{}

	res := newUC(deployRepo, svcRepo, &stubSSHFactory{executor: ex}, lockRepo).Execute(validInput())

	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if ex.uploadedTo != "/etc/nginx/launchpad/svc-1.conf" {
		t.Errorf("expected upload to /etc/nginx/launchpad/svc-1.conf, got %q", ex.uploadedTo)
	}
	if svcRepo.updatedSlot != servicedomain.SlotBlue {
		t.Errorf("expected active_slot=blue, got %v", svcRepo.updatedSlot)
	}
	if deployRepo.statusSet != deploydomain.StatusActive {
		t.Errorf("expected status=active, got %v", deployRepo.statusSet)
	}
	if !lockRepo.releaseCalled {
		t.Error("expected lock to be released")
	}
}

func TestActivate_MissingDeployID(t *testing.T) {
	input := validInput()
	input.DeployID = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_MissingServiceID(t *testing.T) {
	input := validInput()
	input.ServiceID = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_MissingSlot(t *testing.T) {
	input := validInput()
	input.Slot = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_MissingHost(t *testing.T) {
	input := validInput()
	input.Host = ""
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_ZeroActivePort(t *testing.T) {
	input := validInput()
	input.ActivePort = 0
	res := newUC(&stubDeployRepo{}, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(input)
	if !errors.Is(res.Err, activate.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestActivate_DeployNotFound(t *testing.T) {
	deployRepo := &stubDeployRepo{getErr: deploydomain.ErrNotFound}
	res := newUC(deployRepo, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrDeployNotFound) {
		t.Fatalf("expected ErrDeployNotFound, got %v", res.Err)
	}
}

func TestActivate_DeployRepoError(t *testing.T) {
	deployRepo := &stubDeployRepo{getErr: errors.New("db down")}
	res := newUC(deployRepo, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_DeployNotInBuildingState(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: deploydomain.Deploy{Status: deploydomain.StatusPending}}
	ex := happyExecutor()
	res := newUC(deployRepo, &stubServiceRepo{}, &stubSSHFactory{executor: ex}, &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got %v", res.Err)
	}
	if ex.uploadedTo != "" {
		t.Error("expected Upload not to be called when state is invalid")
	}
}

func TestActivate_SSHFailed(t *testing.T) {
	factory := &stubSSHFactory{dialErr: errors.New("connection refused")}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, &stubServiceRepo{}, factory, &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrSSHFailed) {
		t.Fatalf("expected ErrSSHFailed, got %v", res.Err)
	}
}

func TestActivate_UploadFailed(t *testing.T) {
	ex := &stubSSHExecutor{uploadErr: errors.New("scp failed")}
	svcRepo := &stubServiceRepo{}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, svcRepo, &stubSSHFactory{executor: ex}, &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when upload fails")
	}
}

func TestActivate_NginxTFailed(t *testing.T) {
	var cleanupRan bool
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (activate.SSHResult, error) {
			if cmd == "nginx -t" {
				return activate.SSHResult{Stderr: "config test failed"}, errors.New("exit status 1")
			}
			if strings.HasPrefix(cmd, "rm -f") {
				cleanupRan = true
			}
			return activate.SSHResult{}, nil
		},
	}
	svcRepo := &stubServiceRepo{}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, svcRepo, &stubSSHFactory{executor: ex}, &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if !cleanupRan {
		t.Error("expected rm -f cleanup to run after nginx -t failure")
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx -t fails")
	}
}

func TestActivate_NginxReloadFailed(t *testing.T) {
	ex := &stubSSHExecutor{
		runFn: func(cmd string) (activate.SSHResult, error) {
			if cmd == "nginx -s reload" {
				return activate.SSHResult{}, errors.New("exit status 1")
			}
			return activate.SSHResult{}, nil
		},
	}
	svcRepo := &stubServiceRepo{}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, svcRepo, &stubSSHFactory{executor: ex}, &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrNginxFailed) {
		t.Fatalf("expected ErrNginxFailed, got %v", res.Err)
	}
	if svcRepo.updatedSlot != "" {
		t.Error("expected UpdateActiveSlot not to be called when nginx reload fails")
	}
}

func TestActivate_UpdateActiveSlotFails(t *testing.T) {
	svcRepo := &stubServiceRepo{updateErr: errors.New("db down")}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, svcRepo, happyFactory(), &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_SetStatusFails(t *testing.T) {
	deployRepo := &stubDeployRepo{deploy: buildingDeploy, setStatusErr: errors.New("db down")}
	res := newUC(deployRepo, &stubServiceRepo{}, happyFactory(), &stubLockRepo{}).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestActivate_ReleaseLockFails(t *testing.T) {
	lockRepo := &stubLockRepo{releaseErr: errors.New("db down")}
	res := newUC(&stubDeployRepo{deploy: buildingDeploy}, &stubServiceRepo{}, happyFactory(), lockRepo).Execute(validInput())
	if !errors.Is(res.Err, activate.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

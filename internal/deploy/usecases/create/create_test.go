package create_test

import (
	"errors"
	"testing"
	"time"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/create"
)

type stubRepo struct {
	deploy create.CreateResult
	dep    deploydomain.Deploy
	err    error
}

func (r *stubRepo) EnqueueDeploy(serviceID, commitSHA, commitMessage string, pushedAt time.Time) (deploydomain.Deploy, create.CreateResult, error) {
	return r.dep, r.deploy, r.err
}

var (
	now        = time.Now()
	validInput = create.CreateInput{
		ServiceID:     "svc-1",
		CommitSHA:     "abc123",
		CommitMessage: "feat: add feature",
		PushedAt:      now,
	}
)

func TestExecute_ValidationErrors(t *testing.T) {
	called := false
	repo := &stubRepo{}
	_ = repo
	uc := create.New(&trackingRepo{inner: repo, called: &called})

	cases := []struct {
		name  string
		input create.CreateInput
	}{
		{"empty service ID", create.CreateInput{CommitSHA: "abc123", PushedAt: now}},
		{"empty commit SHA", create.CreateInput{ServiceID: "svc-1", PushedAt: now}},
		{"zero pushed_at", create.CreateInput{ServiceID: "svc-1", CommitSHA: "abc123"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			called = false
			res := uc.Execute(tc.input)
			if res.IsOk() {
				t.Fatal("expected error, got ok")
			}
			if !errors.Is(res.Err, create.ErrValidation) {
				t.Fatalf("expected ErrValidation, got %v", res.Err)
			}
			if called {
				t.Fatal("repo must not be called on validation failure")
			}
		})
	}
}

func TestExecute_ServiceNotFound(t *testing.T) {
	repo := &stubRepo{err: deploydomain.ErrServiceNotFound}
	uc := create.New(repo)

	res := uc.Execute(validInput)
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrServiceNotFound) {
		t.Fatalf("expected ErrServiceNotFound, got %v", res.Err)
	}
}

func TestExecute_DeployQueued(t *testing.T) {
	dep := deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", CommitSHA: "abc123", PushedAt: now}
	repo := &stubRepo{dep: dep, deploy: create.DeployQueued}
	uc := create.New(repo)

	res := uc.Execute(validInput)
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Result != create.DeployQueued {
		t.Fatalf("expected DeployQueued, got %v", res.Value.Result)
	}
	if res.Value.Deploy.ID != "dep-1" {
		t.Fatalf("expected deploy ID dep-1, got %v", res.Value.Deploy.ID)
	}
}

func TestExecute_PendingPromoted(t *testing.T) {
	dep := deploydomain.Deploy{ID: "dep-1", ServiceID: "svc-1", CommitSHA: "abc123", PushedAt: now}
	repo := &stubRepo{dep: dep, deploy: create.PendingPromoted}
	uc := create.New(repo)

	res := uc.Execute(validInput)
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Result != create.PendingPromoted {
		t.Fatalf("expected PendingPromoted, got %v", res.Value.Result)
	}
	if res.Value.Deploy.CommitSHA != "abc123" {
		t.Fatalf("expected updated commit SHA abc123, got %v", res.Value.Deploy.CommitSHA)
	}
}

func TestExecute_PushDiscarded(t *testing.T) {
	existingDep := deploydomain.Deploy{ID: "dep-0", ServiceID: "svc-1", CommitSHA: "newer-sha", PushedAt: now.Add(time.Minute)}
	repo := &stubRepo{dep: existingDep, deploy: create.PushDiscarded}
	uc := create.New(repo)

	res := uc.Execute(validInput)
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Result != create.PushDiscarded {
		t.Fatalf("expected PushDiscarded, got %v", res.Value.Result)
	}
	if res.Value.Deploy.CommitSHA != "newer-sha" {
		t.Fatalf("expected existing deploy SHA newer-sha, got %v", res.Value.Deploy.CommitSHA)
	}
}

func TestExecute_RepoError(t *testing.T) {
	repo := &stubRepo{err: errors.New("db connection lost")}
	uc := create.New(repo)

	res := uc.Execute(validInput)
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, create.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

// trackingRepo wraps a stubRepo and records whether EnqueueDeploy was called.
type trackingRepo struct {
	inner  *stubRepo
	called *bool
}

func (r *trackingRepo) EnqueueDeploy(serviceID, commitSHA, commitMessage string, pushedAt time.Time) (deploydomain.Deploy, create.CreateResult, error) {
	*r.called = true
	return r.inner.EnqueueDeploy(serviceID, commitSHA, commitMessage, pushedAt)
}

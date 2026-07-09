package getdeploy_test

import (
	"errors"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/getdeploy"
)

type stubRepo struct {
	deploy deploydomain.Deploy
	err    error
}

func (r *stubRepo) GetByID(deployID string) (deploydomain.Deploy, error) {
	return r.deploy, r.err
}

var sampleDeploy = deploydomain.Deploy{
	ID:        "dep-1",
	ServiceID: "svc-1",
	Status:    deploydomain.StatusActive,
	CommitSHA: "abc123",
}

func TestGetDeploy_EmptyDeployID(t *testing.T) {
	uc := getdeploy.New(&stubRepo{})
	res := uc.Execute(getdeploy.GetDeployInput{})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, getdeploy.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestGetDeploy_NotFound(t *testing.T) {
	uc := getdeploy.New(&stubRepo{err: deploydomain.ErrNotFound})
	res := uc.Execute(getdeploy.GetDeployInput{DeployID: "dep-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, getdeploy.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", res.Err)
	}
}

func TestGetDeploy_RepoError(t *testing.T) {
	uc := getdeploy.New(&stubRepo{err: errors.New("db down")})
	res := uc.Execute(getdeploy.GetDeployInput{DeployID: "dep-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, getdeploy.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestGetDeploy_Success(t *testing.T) {
	uc := getdeploy.New(&stubRepo{deploy: sampleDeploy})
	res := uc.Execute(getdeploy.GetDeployInput{DeployID: "dep-1"})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if res.Value.Deploy.ID != sampleDeploy.ID {
		t.Fatalf("expected deploy ID %s, got %s", sampleDeploy.ID, res.Value.Deploy.ID)
	}
}

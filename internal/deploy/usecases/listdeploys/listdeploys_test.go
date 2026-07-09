package listdeploys_test

import (
	"errors"
	"testing"

	deploydomain "github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/domain"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/listdeploys"
)

type stubRepo struct {
	deploys []deploydomain.Deploy
	err     error
}

func (r *stubRepo) List(serviceID string) ([]deploydomain.Deploy, error) {
	return r.deploys, r.err
}

var sampleDeploys = []deploydomain.Deploy{
	{ID: "dep-1", ServiceID: "svc-1", Status: deploydomain.StatusActive},
	{ID: "dep-2", ServiceID: "svc-1", Status: deploydomain.StatusFailed},
}

func TestListDeploys_EmptyServiceID(t *testing.T) {
	uc := listdeploys.New(&stubRepo{})
	res := uc.Execute(listdeploys.ListDeploysInput{})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, listdeploys.ErrValidation) {
		t.Fatalf("expected ErrValidation, got %v", res.Err)
	}
}

func TestListDeploys_RepoError(t *testing.T) {
	uc := listdeploys.New(&stubRepo{err: errors.New("db down")})
	res := uc.Execute(listdeploys.ListDeploysInput{ServiceID: "svc-1"})
	if res.IsOk() {
		t.Fatal("expected error, got ok")
	}
	if !errors.Is(res.Err, listdeploys.ErrInternal) {
		t.Fatalf("expected ErrInternal, got %v", res.Err)
	}
}

func TestListDeploys_EmptyResult(t *testing.T) {
	uc := listdeploys.New(&stubRepo{deploys: []deploydomain.Deploy{}})
	res := uc.Execute(listdeploys.ListDeploysInput{ServiceID: "svc-1"})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if len(res.Value.Deploys) != 0 {
		t.Fatalf("expected empty slice, got %d items", len(res.Value.Deploys))
	}
}

func TestListDeploys_Success(t *testing.T) {
	uc := listdeploys.New(&stubRepo{deploys: sampleDeploys})
	res := uc.Execute(listdeploys.ListDeploysInput{ServiceID: "svc-1"})
	if !res.IsOk() {
		t.Fatalf("expected ok, got %v", res.Err)
	}
	if len(res.Value.Deploys) != len(sampleDeploys) {
		t.Fatalf("expected %d deploys, got %d", len(sampleDeploys), len(res.Value.Deploys))
	}
	if res.Value.Deploys[0].ID != "dep-1" {
		t.Fatalf("expected dep-1 first, got %s", res.Value.Deploys[0].ID)
	}
}

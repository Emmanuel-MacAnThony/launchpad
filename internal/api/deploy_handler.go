package api

import (
	"errors"
	"net/http"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/getdeploy"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/listdeploys"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type GetDeployUseCase interface {
	Execute(input getdeploy.GetDeployInput) result.Result[getdeploy.GetDeployOutput]
}

type ListDeploysUseCase interface {
	Execute(input listdeploys.ListDeploysInput) result.Result[listdeploys.ListDeploysOutput]
}

type DeployHandlerDeps struct {
	GetDeploy   GetDeployUseCase
	ListDeploys ListDeploysUseCase
}

type DeployHandler struct {
	deps DeployHandlerDeps
}

func NewDeployHandler(deps DeployHandlerDeps) *DeployHandler {
	return &DeployHandler{deps: deps}
}

func (h *DeployHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /deploys/{deployID}", h.Get)
	mux.HandleFunc("GET /services/{serviceID}/deploys", h.List)
}

func (h *DeployHandler) Get(w http.ResponseWriter, r *http.Request) {
	deployID := r.PathValue("deployID")

	res := h.deps.GetDeploy.Execute(getdeploy.GetDeployInput{DeployID: deployID})
	if res.Err != nil {
		if errors.Is(res.Err, getdeploy.ErrNotFound) {
			writeError(w, http.StatusNotFound, "deploy not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, res.Value.Deploy)
}

func (h *DeployHandler) List(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("serviceID")

	res := h.deps.ListDeploys.Execute(listdeploys.ListDeploysInput{ServiceID: serviceID})
	if res.Err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, res.Value.Deploys)
}

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/list"
	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/update"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type CreateServiceUseCase interface {
	Execute(input create.CreateInput) result.Result[create.CreateOutput]
}

type GetServiceUseCase interface {
	Execute(input get.GetInput) result.Result[get.GetOutput]
}

type UpdateServiceUseCase interface {
	Execute(input update.UpdateInput) result.Result[update.UpdateOutput]
}

type ListServicesUseCase interface {
	Execute(input list.ListInput) result.Result[list.ListOutput]
}

type ServiceHandlerDeps struct {
	BaseURL       string
	CreateService CreateServiceUseCase
	GetService    GetServiceUseCase
	UpdateService UpdateServiceUseCase
	ListServices  ListServicesUseCase
}

type ServiceHandler struct {
	deps ServiceHandlerDeps
}

func NewServiceHandler(deps ServiceHandlerDeps) *ServiceHandler {
	return &ServiceHandler{deps: deps}
}

func (h *ServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /services", h.Create)
	mux.HandleFunc("GET /services", h.List)
	mux.HandleFunc("GET /services/{id}", h.Get)
	mux.HandleFunc("PATCH /services/{id}", h.Update)
}

// ── shared response ───────────────────────────────────────

type serviceResponse struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	RepoURL        string `json:"repo_url"`
	Domain         string `json:"domain"`
	HealthCheckURL string `json:"health_check_url"`
	Host           string `json:"host"`
	SSHUser        string `json:"ssh_user"`
	SSHKeyPath     string `json:"ssh_key_path"`
	WebhookURL     string `json:"webhook_url"`
	CreatedAt      string `json:"created_at"`
}

func (h *ServiceHandler) webhookURL(id string) string {
	return fmt.Sprintf("%s/webhooks/%s", h.deps.BaseURL, id)
}

func toServiceResponse(id, name, repoURL, domain, healthCheckURL, host, sshUser, sshKeyPath, webhookURL string, createdAt time.Time) serviceResponse {
	return serviceResponse{
		ID:             id,
		Name:           name,
		RepoURL:        repoURL,
		Domain:         domain,
		HealthCheckURL: healthCheckURL,
		Host:           host,
		SSHUser:        sshUser,
		SSHKeyPath:     sshKeyPath,
		WebhookURL:     webhookURL,
		CreatedAt:      createdAt.UTC().Format("2006-01-02T15:04:05Z"),
	}
}

// ── create ────────────────────────────────────────────────

type createServiceRequest struct {
	Name           string `json:"name"`
	RepoURL        string `json:"repo_url"`
	Domain         string `json:"domain"`
	HealthCheckURL string `json:"health_check_url"`
	WebhookSecret  string `json:"webhook_secret"`
	Host           string `json:"host"`
	SSHUser        string `json:"ssh_user"`
	SSHKeyPath     string `json:"ssh_key_path"`
	BluePort       int    `json:"blue_port"`
	GreenPort      int    `json:"green_port"`
	ContainerPort  int    `json:"container_port"`
}

func (h *ServiceHandler) Create(w http.ResponseWriter, r *http.Request) {
	var req createServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res := h.deps.CreateService.Execute(create.CreateInput{
		Name:           req.Name,
		RepoURL:        req.RepoURL,
		Domain:         req.Domain,
		HealthCheckURL: req.HealthCheckURL,
		WebhookSecret:  req.WebhookSecret,
		Host:           req.Host,
		SSHUser:        req.SSHUser,
		SSHKeyPath:     req.SSHKeyPath,
		BluePort:       req.BluePort,
		GreenPort:      req.GreenPort,
		ContainerPort:  req.ContainerPort,
	})

	if res.Err != nil {
		h.handleCreateError(w, res.Err)
		return
	}

	v := res.Value
	writeJSON(w, http.StatusCreated, toServiceResponse(
		v.ID, v.Name, v.RepoURL, v.Domain, v.HealthCheckURL,
		v.Host, v.SSHUser, v.SSHKeyPath, h.webhookURL(v.ID), v.CreatedAt,
	))
}

func (h *ServiceHandler) handleCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, create.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, create.ErrDomainTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, create.ErrSSHFailed):
		writeError(w, http.StatusBadGateway, "could not connect to host")
	case errors.Is(err, create.ErrDockerNotInstalled), errors.Is(err, create.ErrNginxNotInstalled):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, create.ErrBootstrapFailed):
		writeError(w, http.StatusInternalServerError, "failed to bootstrap nginx on host")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ── get ───────────────────────────────────────────────────

func (h *ServiceHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	res := h.deps.GetService.Execute(get.GetInput{ID: id})
	if res.Err != nil {
		h.handleGetError(w, res.Err)
		return
	}

	v := res.Value
	writeJSON(w, http.StatusOK, toServiceResponse(
		v.ID, v.Name, v.RepoURL, v.Domain, v.HealthCheckURL,
		v.Host, v.SSHUser, v.SSHKeyPath, h.webhookURL(v.ID), v.CreatedAt,
	))
}

func (h *ServiceHandler) handleGetError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, get.ErrInvalidInput):
		writeError(w, http.StatusBadRequest, err.Error())
	case errors.Is(err, get.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ── update ────────────────────────────────────────────────

type updateServiceRequest struct {
	Name           string `json:"name"`
	HealthCheckURL string `json:"health_check_url"`
}

func (h *ServiceHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	var req updateServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	res := h.deps.UpdateService.Execute(update.UpdateInput{
		ID:             id,
		Name:           req.Name,
		HealthCheckURL: req.HealthCheckURL,
	})

	if res.Err != nil {
		h.handleUpdateError(w, res.Err)
		return
	}

	v := res.Value
	writeJSON(w, http.StatusOK, toServiceResponse(
		v.ID, v.Name, v.RepoURL, v.Domain, v.HealthCheckURL,
		v.Host, v.SSHUser, v.SSHKeyPath, h.webhookURL(v.ID), v.CreatedAt,
	))
}

func (h *ServiceHandler) handleUpdateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, update.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, update.ErrNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

// ── list ──────────────────────────────────────────────────

func (h *ServiceHandler) List(w http.ResponseWriter, r *http.Request) {
	res := h.deps.ListServices.Execute(list.ListInput{})
	if res.Err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	items := make([]serviceResponse, len(res.Value.Services))
	for i, svc := range res.Value.Services {
		items[i] = toServiceResponse(
			svc.ID, svc.Name, svc.RepoURL, svc.Domain, svc.HealthCheckURL,
			svc.Host, svc.SSHUser, svc.SSHKeyPath, h.webhookURL(svc.ID), svc.CreatedAt,
		)
	}

	writeJSON(w, http.StatusOK, items)
}

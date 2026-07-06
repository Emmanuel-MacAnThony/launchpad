package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/create"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type CreateServiceUseCase interface {
	Execute(input create.CreateInput) result.Result[create.CreateOutput]
}

type ServiceHandlerDeps struct {
	CreateService CreateServiceUseCase
}

type ServiceHandler struct {
	deps ServiceHandlerDeps
}

func NewServiceHandler(deps ServiceHandlerDeps) *ServiceHandler {
	return &ServiceHandler{deps: deps}
}

func (h *ServiceHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /services", h.Create)
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
}

type createServiceResponse struct {
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
	})

	if res.Err != nil {
		h.handleCreateError(w, res.Err)
		return
	}

	writeJSON(w, http.StatusCreated, createServiceResponse{
		ID:             res.Value.ID,
		Name:           res.Value.Name,
		RepoURL:        res.Value.RepoURL,
		Domain:         res.Value.Domain,
		HealthCheckURL: res.Value.HealthCheckURL,
		Host:           res.Value.Host,
		SSHUser:        res.Value.SSHUser,
		SSHKeyPath:     res.Value.SSHKeyPath,
		WebhookURL:     res.Value.WebhookURL,
		CreatedAt:      res.Value.CreatedAt.UTC().Format("2006-01-02T15:04:05Z"),
	})
}

func (h *ServiceHandler) handleCreateError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, create.ErrInvalidInput):
		writeError(w, http.StatusUnprocessableEntity, err.Error())
	case errors.Is(err, create.ErrDomainTaken):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, create.ErrNginxConfigFailed), errors.Is(err, create.ErrNginxReloadFailed):
		writeError(w, http.StatusInternalServerError, "failed to configure routing")
	default:
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

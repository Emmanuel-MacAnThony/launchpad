package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Emmanuel-MacAnThony/launchpad/internal/deploy/usecases/create"
	serviceget "github.com/Emmanuel-MacAnThony/launchpad/internal/service/usecases/get"
	"github.com/Emmanuel-MacAnThony/launchpad/pkg/result"
)

type CreateDeployUseCase interface {
	Execute(input create.CreateInput) result.Result[create.CreateOutput]
}

type GetServiceForWebhookUseCase interface {
	Execute(input serviceget.GetInput) result.Result[serviceget.GetOutput]
}

type WebhookHandlerDeps struct {
	GetService   GetServiceForWebhookUseCase
	CreateDeploy CreateDeployUseCase
}

type WebhookHandler struct {
	deps WebhookHandlerDeps
}

func NewWebhookHandler(deps WebhookHandlerDeps) *WebhookHandler {
	return &WebhookHandler{deps: deps}
}

func (h *WebhookHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /webhooks/{serviceID}", h.Handle)
}

type githubPushPayload struct {
	Ref        string `json:"ref"`
	HeadCommit struct {
		ID      string `json:"id"`
		Message string `json:"message"`
	} `json:"head_commit"`
	Repository struct {
		PushedAt int64 `json:"pushed_at"`
	} `json:"repository"`
}

func (h *WebhookHandler) Handle(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("serviceID")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "failed to read request body")
		return
	}

	svcRes := h.deps.GetService.Execute(serviceget.GetInput{ID: serviceID})
	if svcRes.Err != nil {
		if errors.Is(svcRes.Err, serviceget.ErrNotFound) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	if !verifySignature(body, svcRes.Value.WebhookSecret, r.Header.Get("X-Hub-Signature-256")) {
		writeError(w, http.StatusUnauthorized, "invalid signature")
		return
	}

	var payload githubPushPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}

	if payload.Ref != "refs/heads/main" {
		writeJSON(w, http.StatusOK, map[string]string{"result": "skipped"})
		return
	}

	if payload.HeadCommit.ID == "" {
		writeError(w, http.StatusBadRequest, "missing head commit")
		return
	}

	res := h.deps.CreateDeploy.Execute(create.CreateInput{
		ServiceID:     serviceID,
		CommitSHA:     payload.HeadCommit.ID,
		CommitMessage: payload.HeadCommit.Message,
		PushedAt:      time.Unix(payload.Repository.PushedAt, 0).UTC(),
	})

	if res.Err != nil {
		if errors.Is(res.Err, create.ErrServiceNotFound) {
			writeError(w, http.StatusNotFound, "service not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusAccepted, map[string]string{
		"deploy_id": res.Value.Deploy.ID,
		"result":    string(res.Value.Result),
	})
}

func verifySignature(body []byte, secret, signatureHeader string) bool {
	if signatureHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := fmt.Sprintf("sha256=%s", hex.EncodeToString(mac.Sum(nil)))
	return hmac.Equal([]byte(expected), []byte(signatureHeader))
}

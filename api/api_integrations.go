package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/PlakarKorp/pkg"
)

type IntegrationsMessage struct {
	Date    time.Time `json:"date"`
	Message string    `json:"message"`
}

type IntegrationsResponse struct {
	Type       string                `json:"type"`
	Status     string                `json:"status"`
	StartedAt  time.Time             `json:"started_at"`
	FinishedAt time.Time             `json:"finished_at"`
	Messages   []IntegrationsMessage `json:"messages"`
}

func NewIntegrationsResponse(Type string) *IntegrationsResponse {
	return &IntegrationsResponse{
		Type:      Type,
		Status:    "completed",
		StartedAt: time.Now(),
		Messages:  make([]IntegrationsMessage, 0),
	}
}

func (r *IntegrationsResponse) AddMessage(msg string) {
	r.Messages = append(r.Messages, IntegrationsMessage{
		Date:    time.Now(),
		Message: msg,
	})
}

type IntegrationsInstallRequest struct {
	Id      string `json:"id"`
	Version string `json:"version"`
}

func (ui *uiserver) integrationsInstall(w http.ResponseWriter, r *http.Request) error {
	var req IntegrationsInstallRequest

	resp := NewIntegrationsResponse("pkg_install")
	resp.Status = "failed"

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("failed to decode request body: %v", err))
		goto done
	}

	err = ui.ctx.GetPkgManager().Add(req.Id, &pkg.AddOptions{
		Version:       req.Version,
		ImplicitFetch: true,
	})
	if err != nil {
		resp.AddMessage(fmt.Sprintf("install command failed: %v", err))
		goto done
	}

	resp.Status = "ok"
	resp.AddMessage(fmt.Sprintf("plugin %q installed successfully", req.Id))

done:
	return json.NewEncoder(w).Encode(resp)
}

func (ui *uiserver) integrationsUninstall(w http.ResponseWriter, r *http.Request) error {
	resp := NewIntegrationsResponse("pkg_uninstall")
	resp.Status = "failed"

	id := r.PathValue("id")

	err := ui.ctx.GetPkgManager().Del(id, nil)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("uninstall command failed: %v", err))
		goto done
	}

	resp.Status = "ok"
	resp.AddMessage(fmt.Sprintf("plugin %q uninstalled successfully", id))

done:
	return json.NewEncoder(w).Encode(resp)
}

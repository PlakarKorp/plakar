package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/PlakarKorp/plakar/subcommands/pkg"
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
	resp := NewIntegrationsResponse("pkg_install")
	pkgName := r.PathValue("pkg")

	var cmd pkg.PkgAdd
	cmd.Args = append(cmd.Args, pkgName)
	_, err := cmd.Execute(ui.ctx, ui.repository)
	if err != nil {
		resp.Status = "failed"
		resp.AddMessage(fmt.Sprintf("unistall command failed: %v", err))
	} else {
		resp.AddMessage(fmt.Sprintf("plugin %s installed successfully", pkgName))
	}

	return json.NewEncoder(w).Encode(resp)
}

func (ui *uiserver) integrationsUninstall(w http.ResponseWriter, r *http.Request) error {
	resp := NewIntegrationsResponse("pkg_uninstall")
	pkgName := r.PathValue("pkg")

	var cmd pkg.PkgRm
	cmd.Args = append(cmd.Args, pkgName)
	_, err := cmd.Execute(ui.ctx, ui.repository)
	if err != nil {
		resp.Status = "failed"
		resp.AddMessage(fmt.Sprintf("unistall command failed: %v", err))
	} else {
		resp.AddMessage(fmt.Sprintf("plugin %s uninstalled successfully", pkgName))
	}

	return json.NewEncoder(w).Encode(resp)
}

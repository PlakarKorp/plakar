package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/PlakarKorp/plakar/plugins"
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
	var cmd pkg.PkgAdd
	var pkgName string
	var req IntegrationsInstallRequest

	resp := NewIntegrationsResponse("pkg_install")
	resp.Status = "failed"

	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("failed to decode request body: %v", err))
		goto done
	}

	pkgName = fmt.Sprintf("%s_%s_%s_%s.ptar", req.Id, req.Version, runtime.GOOS, runtime.GOARCH)
	cmd.Args = append(cmd.Args, pkgName)
	_, err = cmd.Execute(ui.ctx, ui.repository)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("install command failed: %v", err))
		goto done
	}

	resp.Status = "ok"
	resp.AddMessage(fmt.Sprintf("plugin %q installed successfully", pkgName))

done:
	return json.NewEncoder(w).Encode(resp)
}

func (ui *uiserver) integrationsUninstall(w http.ResponseWriter, r *http.Request) error {
	var cmd pkg.PkgRm
	var plugin *plugins.InstalledPlugin

	resp := NewIntegrationsResponse("pkg_uninstall")
	resp.Status = "failed"

	id := r.PathValue("id")

	p, err := plugins.ListInstalledPlugins(ui.ctx)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("%v", err))
		goto done
	}
	plugin = p.GetPlugin(id)
	if plugin == nil {
		resp.AddMessage("plugin not installed")
		goto done
	}

	cmd.Args = append(cmd.Args, plugin.PkgName())
	_, err = cmd.Execute(ui.ctx, ui.repository)
	if err != nil {
		resp.AddMessage(fmt.Sprintf("uninstall command failed: %v", err))
		goto done
	}

	resp.Status = "ok"
	resp.AddMessage(fmt.Sprintf("plugin %q uninstalled successfully", id))

done:
	return json.NewEncoder(w).Encode(resp)
}

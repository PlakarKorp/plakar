package rclone

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	stdpath "path"
	"strings"

	"github.com/PlakarKorp/plakar/objects"
	"github.com/PlakarKorp/plakar/snapshot/exporter"

	_ "github.com/rclone/rclone/backend/all"
	"github.com/rclone/rclone/librclone/librclone"
)

type RcloneExporter struct {
	remote   string
	base     string
	provider string
}

var specialCases = []string{
	"googlephoto",
}

func init() {
	exporter.Register("onedrive", NewRcloneExporter)
	exporter.Register("opendrive", NewRcloneExporter)
	exporter.Register("googledrive", NewRcloneExporter)
	exporter.Register("googlephoto", NewRcloneExporter)
}

func NewRcloneExporter(config map[string]string) (exporter.Exporter, error) {
	provider, location, _ := strings.Cut(config["location"], "://")
	remote, base, found := strings.Cut(location, ":")

	if !found {
		return nil, fmt.Errorf("invalid location: %s. Expected format: remote:path/to/dir", location)
	}

	librclone.Initialize()

	return &RcloneExporter{
		remote:   remote,
		base:     base,
		provider: provider,
	}, nil
}

// getPathInBackup returns the full normalized path of a file within the backup.
//
// The resulting path is constructed by joining the base path of the backup (p.base)
// with the provided relative path. If the base path (p.base) is not absolute,
// it is treated as relative to the root ("/").
func (p *RcloneExporter) getPathInBackup(path string) string {
	path = stdpath.Join(p.base, path)

	if !stdpath.IsAbs(p.base) {
		path = "/" + path
	}

	return stdpath.Clean(path)
}

func (p *RcloneExporter) Root() string {
	return p.getPathInBackup("")
}

func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func (p *RcloneExporter) CreateDirectory(pathname string) error {
	if contains(specialCases, p.provider) {
		return nil
	}
	relativePath := strings.TrimPrefix(pathname, p.getPathInBackup(""))

	payload := map[string]string{
		"fs":     fmt.Sprintf("%s:%s", p.remote, p.base),
		"remote": relativePath,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body, resp := librclone.RPC("operations/mkdir", string(jsonPayload))
	if resp != http.StatusOK {
		return fmt.Errorf("failed to create directory: %s", body)
	}

	return nil
}

// XXX: it seems there is a race condition when restoring a directory: when
// exporting the first file, the root directory is created. When exporting the
// second file, it is possible that Google Drive doesn't see the root directory
// yet, and creates a new one. This results in a duplicated root directory, with
// some files in the first directory and the rest in the second.
func (p *RcloneExporter) StoreFile(pathname string, fp io.Reader) error {
	tmpFile, err := os.CreateTemp("", "tempfile-*.tmp")
	if err != nil {
		return err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	_, err = io.Copy(tmpFile, fp)
	if err != nil {
		return err
	}

	relativePath := strings.TrimPrefix(pathname, p.getPathInBackup(""))

	payload := map[string]string{
		"srcFs":     "/",
		"srcRemote": tmpFile.Name(),
		"dstFs":     fmt.Sprintf("%s:%s", p.remote, p.base),
		"dstRemote": relativePath,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	body, resp := librclone.RPC("operations/copyfile", string(jsonPayload))

	if resp != http.StatusOK {
		return fmt.Errorf("failed to copy file: %s", body)
	}

	return nil
}

func (p *RcloneExporter) SetPermissions(pathname string, fileinfo *objects.FileInfo) error {
	return nil
}

func (p *RcloneExporter) Close() error {
	return nil
}

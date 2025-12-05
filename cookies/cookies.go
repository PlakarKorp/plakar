package cookies

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

const COOKIES_VERSION = "1.0.0"

type Manager struct {
	cookiesDir string
}

func NewManager(cookiesDir string) *Manager {
	cookiesDir = filepath.Join(cookiesDir, "cookies", COOKIES_VERSION)
	if err := os.MkdirAll(cookiesDir, 0700); err != nil {
		panic(fmt.Errorf("cannot create cookies directory: %w", err))
	}
	if err := os.Chmod(cookiesDir, 0700); err != nil {
		panic(fmt.Errorf("cannot set permissions for cookies directory: %w", err))
	}
	return &Manager{
		cookiesDir: cookiesDir,
	}
}

func (m *Manager) Close() error {
	return nil
}

func (m *Manager) GetDir() string {
	return m.cookiesDir
}

func (c *Manager) GetAuthToken() (string, error) {
	token := os.Getenv("PLAKAR_TOKEN")
	if token != "" {
		return token, nil
	}

	data, err := os.ReadFile(filepath.Join(c.cookiesDir, ".auth-token"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("no auth token found: use `plakar login` first")
		}
		return "", err
	}
	if len(data) == 0 {
		return "", fmt.Errorf("no auth token found: use `plakar login` first")
	}
	return string(data), nil
}

func (c *Manager) HasAuthToken() bool {
	_, err := os.Stat(filepath.Join(c.cookiesDir, ".auth-token"))
	return err == nil
}

func (c *Manager) DeleteAuthToken() error {
	return os.Remove(filepath.Join(c.cookiesDir, ".auth-token"))
}

func (c *Manager) PutAuthToken(token string) error {
	return os.WriteFile(filepath.Join(c.cookiesDir, ".auth-token"), []byte(token), 0600)
}

func (c *Manager) HasRepositoryCookie(repositoryId uuid.UUID, name string) bool {
	name = strings.ReplaceAll(name, "/", "_")
	_, err := os.Stat(filepath.Join(c.cookiesDir, repositoryId.String(), name))
	return err == nil
}

func (c *Manager) PutRepositoryCookie(repositoryId uuid.UUID, name string) error {
	err := os.MkdirAll(filepath.Join(c.cookiesDir, repositoryId.String()), 0700)
	if err != nil {
		return err
	}
	name = strings.ReplaceAll(name, "/", "_")
	_, err = os.Create(filepath.Join(c.cookiesDir, repositoryId.String(), name))
	return err
}

func (c *Manager) IsFirstRun() bool {
	_, err := os.Stat(filepath.Join(c.cookiesDir, ".first-run"))
	if os.IsNotExist(err) {
		return true
	} else if err != nil {
		return false
	}
	return false
}

func (c *Manager) SetFirstRun() error {
	return os.WriteFile(filepath.Join(c.cookiesDir, ".first-run"), []byte{}, 0600)
}

func (c *Manager) IsDisabledSecurityCheck() bool {
	_, err := os.Stat(filepath.Join(c.cookiesDir, ".disabled-security-check"))
	if os.IsNotExist(err) {
		return false
	} else {
		return true
	}
}

func (c *Manager) SetDisabledSecurityCheck() error {
	return os.WriteFile(filepath.Join(c.cookiesDir, ".disabled-security-check"), []byte{}, 0600)
}

func (c *Manager) RemoveDisabledSecurityCheck() error {
	return os.Remove(filepath.Join(c.cookiesDir, ".disabled-security-check"))
}

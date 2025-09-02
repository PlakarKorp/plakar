package plugins

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/kcontext"
	"golang.org/x/mod/semver"
)

type Cache[T any] struct {
	ttl  time.Duration
	get  func() (T, error)
	last time.Time
	v    T
}

func (c *Cache[T]) Get() (T, error) {
	if time.Since(c.last) > c.ttl {
		v, err := c.get()
		if err != nil {
			return c.v, err
		}
		c.v = v
		c.last = time.Now()
	}
	return c.v, nil
}

type Manager struct {
	ApiVersion string
	Os         string
	Arch       string

	PluginsDir string // Where plugins are installed
	CacheDir   string // where plugins are decompressed

	PackagesUrl string // Where prebuilt packages are retrieved from

	pluginsMtx   sync.Mutex
	plugins      map[Package]*Plugin  // list of loaded plugins
	packages     Cache[[]Package]     // list of available packages
	integrations Cache[[]Integration] // list of integrations
}

func NewManager(pluginsDir, cacheDir string) *Manager {
	mgr := &Manager{
		ApiVersion:  PLUGIN_API_VERSION,
		Os:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		PluginsDir:  filepath.Join(pluginsDir, "plugins", PLUGIN_API_VERSION),
		CacheDir:    filepath.Join(cacheDir, "plugins", PLUGIN_API_VERSION),
		PackagesUrl: "https://plugins.plakar.io/kloset/pkg/" + PLUGIN_API_VERSION + "/",

		plugins: make(map[Package]*Plugin),
	}
	mgr.packages = Cache[[]Package]{
		ttl: 5 * time.Minute,
		get: func() ([]Package, error) { return fetchPackages(mgr.PackagesUrl) },
	}
	mgr.integrations = Cache[[]Integration]{
		ttl: 5 * time.Minute,
		get: func() ([]Integration, error) { return fetchIntegrationList(mgr.ApiVersion) },
	}
	return mgr
}

func (mgr *Manager) PackageUrl(pkg Package) string {
	return mgr.PackagesUrl + pkg.PkgName()
}

func (mgr *Manager) ListAvailablePackages() ([]Package, error) {
	packages, err := mgr.packages.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to list available packages: %w", err)
	}

	var res []Package
	for _, pkg := range packages {
		if pkg.Os == mgr.Os && pkg.Arch == mgr.Arch {
			res = append(res, pkg)
		}
	}

	return res, nil
}

func (mgr *Manager) IsAvailable(pkg Package) (bool, error) {
	packages, err := mgr.ListAvailablePackages()
	if err != nil {
		return false, err
	}
	if slices.Contains(packages, pkg) {
		return true, nil
	}
	return false, nil
}

func (mgr *Manager) FindAvailablePackage(name string) (Package, error) {
	packages, err := mgr.packages.Get()
	if err != nil {
		return Package{}, fmt.Errorf("failed to list packages")
	}
	for _, pkg := range packages {
		if pkg.Name == name {
			return pkg, nil
		}
	}
	return Package{}, fmt.Errorf("package not available")
}

func (mgr *Manager) ListInstalledPackages() ([]Package, error) {
	var packages []Package

	dirEntries, err := os.ReadDir(mgr.PluginsDir)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to list installed packages: %w", err)
	}

	for _, entry := range dirEntries {
		if !entry.Type().IsRegular() {
			continue
		}
		var pkg Package
		err := ParsePackageName(entry.Name(), &pkg)
		if err == nil {
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

func (mgr *Manager) FindInstalledPackage(name string) (Package, error) {
	packages, err := mgr.ListInstalledPackages()
	if err != nil {
		return Package{}, fmt.Errorf("failed to list installed packages")
	}
	for _, pkg := range packages {
		if pkg.Name == name {
			return pkg, nil
		}
		if pkg.PkgName() == name {
			return pkg, nil
		}
		if pkg.PkgNameAndVersion() == name {
			return pkg, nil
		}
	}
	return Package{}, fmt.Errorf("package not installed")
}

func (mgr *Manager) IsInstalled(pkg Package) (bool, Package, error) {
	packages, err := mgr.ListInstalledPackages()
	if err != nil {
		return false, Package{}, fmt.Errorf("failed to list installed packages")
	}
	for _, p := range packages {
		if p.Name == pkg.Name {
			return true, p, nil
		}
	}
	return false, Package{}, nil
}
func GetStageFromVersion(version string) string {
	stage := strings.TrimPrefix(semver.Prerelease(version), "-")

	if stage == "" {
		return "stable"
	}
	if stage == "rc" {
		return "testing"
	}

	stage, _, _ = strings.Cut(stage, ".")

	return stage
}

func (mgr *Manager) ListIntegrations(filter IntegrationFilter) ([]Integration, error) {
	integrations, err := mgr.integrations.Get()
	if err != nil {
		return nil, fmt.Errorf("failed to list integrations")
	}

	var res []Integration
	for _, info := range integrations {
		if filter.Type == "storage" && !info.Types.Storage {
			continue
		}
		if filter.Type == "source" && !info.Types.Source {
			continue
		}
		if filter.Type == "destination" && !info.Types.Destination {
			continue
		}
		if filter.Tag != "" && !slices.Contains(info.Tags, filter.Tag) {
			continue
		}
		info.Installation.Status = "not-installed"

		pkg, err := mgr.FindInstalledPackage(info.Name)
		if err == nil {
			info.Installation.Status = "installed"
			info.Installation.Version = pkg.Version
		}

		if filter.Status != "" && filter.Status != info.Installation.Status {
			continue
		}

		ok, _ := mgr.IsAvailable(mgr.IntegrationAsPackage(&info))
		if ok {
			info.Installation.Available = true
		}

		info.Stage = GetStageFromVersion(info.LatestVersion)

		res = append(res, info)
	}

	return res, nil
}

func (mgr *Manager) IntegrationAsPackage(int *Integration) Package {
	return Package{
		Name:    int.Name,
		Version: int.LatestVersion,
		Os:      mgr.Os,
		Arch:    mgr.Arch,
	}
}

func (mgr *Manager) PluginFile(pkg Package) string {
	return filepath.Join(mgr.PluginsDir, pkg.PkgName())
}

func (mgr *Manager) PluginCache(pkg Package) string {
	return filepath.Join(mgr.CacheDir, pkg.PluginName())
}

func (mgr *Manager) doUnloadPlugins(ctx *kcontext.KContext) {
	for _, plugin := range mgr.plugins {
		plugin.TearDown(ctx)
	}
}

func (mgr *Manager) UnloadPlugins(ctx *kcontext.KContext) {
	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()
	mgr.doUnloadPlugins(ctx)
}

func (mgr *Manager) doLoadPlugins(ctx *kcontext.KContext) error {
	packages, err := mgr.ListInstalledPackages()
	if err != nil {
		return err
	}

	for _, pkg := range packages {
		var plugin Plugin
		err := plugin.SetUp(ctx, mgr.PluginFile(pkg), pkg.PluginName(), mgr.CacheDir)
		if err != nil {
			ctx.GetLogger().Warn("failed to load plugin %q: %v", mgr.PluginFile(pkg), err)
			continue

		}
		mgr.plugins[pkg] = &plugin
	}

	return nil
}

func (mgr *Manager) checkIfPluginsNeedReload() (bool, error) {
	var loaded []Package
	for pkg := range mgr.plugins {
		loaded = append(loaded, pkg)
	}
	installed, err := mgr.ListInstalledPackages()
	if err != nil {
		return false, err
	}

	slices.SortFunc(loaded, PackageCmp)
	slices.SortFunc(installed, PackageCmp)

	return !slices.Equal(loaded, installed), nil
}

func (mgr *Manager) LoadPlugins(ctx *kcontext.KContext) error {
	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()
	return mgr.doLoadPlugins(ctx)
}

func (mgr *Manager) ReloadPlugins(ctx *kcontext.KContext) error {
	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()

	needed, err := mgr.checkIfPluginsNeedReload()
	if err != nil {
		return err
	}

	if !needed {
		return nil
	}
	mgr.doUnloadPlugins(ctx)
	return mgr.doLoadPlugins(ctx)
}

func (mgr *Manager) ForceReloadPlugins(ctx *kcontext.KContext) error {
	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()

	mgr.doUnloadPlugins(ctx)
	return mgr.doLoadPlugins(ctx)
}

func (mgr *Manager) UninstallPackage(ctx *kcontext.KContext, pkg Package) error {
	err := pkg.Validate()
	if err != nil {
		return err
	}

	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()
	plugin, ok := mgr.plugins[pkg]
	if ok {
		delete(mgr.plugins, pkg)
		plugin.TearDown(ctx)
	}

	pluginFile := mgr.PluginFile(pkg)

	err = os.Remove(pluginFile)
	if err != nil {
		return fmt.Errorf("failed to remove %q: %w", pluginFile, err)
	}

	err = os.RemoveAll(mgr.PluginCache(pkg))
	if err != nil {
		return fmt.Errorf("failed to remove cache for %q: %w", pkg.PluginName(), err)
	}

	return nil
}

func (mgr *Manager) InstallPackage(ctx *kcontext.KContext, pkg Package, filename string) error {
	err := pkg.Validate()
	if err != nil {
		return err
	}

	mgr.pluginsMtx.Lock()
	defer mgr.pluginsMtx.Unlock()

	// Check if installed
	installed, err := mgr.ListInstalledPackages()
	if err != nil {
		return fmt.Errorf("failed to list installed packages: %w", err)
	}
	for _, p := range installed {
		if p == pkg {
			return fmt.Errorf("plugin %q already installed", pkg.Name)
		}
		if p.Name == pkg.Name {
			return fmt.Errorf("plugin %q already installed in a different version, remove first", pkg.Name)
		}
	}

	var plugin Plugin

	// Try to setup the plugin
	err = plugin.SetUp(ctx, filename, pkg.PluginName(), mgr.CacheDir)
	if err != nil {
		return fmt.Errorf("failed to load plugin %q: %w", pkg.Name, err)
	}

	err = installPlugin(filename, mgr.PluginFile(pkg))
	if err != nil {
		os.RemoveAll(mgr.PluginCache(pkg))
		plugin.TearDown(ctx)
		return fmt.Errorf("failed to install plugin file %s: %w", filename, err)
	}

	mgr.plugins[pkg] = &plugin
	return nil
}

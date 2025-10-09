package plugins

import (
	"cmp"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"golang.org/x/mod/semver"
)

type Package struct {
	Name    string
	Version string
	Os      string
	Arch    string
}

func ParsePackageName(name string, pkg *Package) error {
	if !strings.HasSuffix(name, ".ptar") {
		return fmt.Errorf("package name %q does not end with .ptar", name)
	}

	baseName := strings.TrimSuffix(name, ".ptar")
	atoms := strings.Split(baseName, "_")
	if len(atoms) != 4 {
		return fmt.Errorf("package name %q does not contain all atoms (name, version, OS, architecture)", name)
	}

	pkg.Name = atoms[0]
	pkg.Version = atoms[1]
	pkg.Os = atoms[2]
	pkg.Arch = atoms[3]

	err := pkg.Validate()
	if err != nil {
		*pkg = Package{}
		return err
	}

	return nil
}

func PackageCmp(a, b Package) int {
	return cmp.Compare(a.PkgName(), b.PkgName())
}

func isNameChar(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '-'
}

func isOsArchChar(c byte) bool {
	return 'A' <= c && c <= 'Z' || 'a' <= c && c <= 'z' || '0' <= c && c <= '9'
}

func (pkg Package) Validate() error {
	if pkg.Name == "" {
		return fmt.Errorf("package has no name")
	}
	for i := 0; i < len(pkg.Name); i++ {
		if !isNameChar(pkg.Name[i]) {
			return fmt.Errorf("package name contains invalid char '%c'", pkg.Name[i])
		}
	}

	if !semver.IsValid(pkg.Version) {
		return fmt.Errorf("package has invalid version %q", pkg.Version)
	}

	for i := 0; i < len(pkg.Os); i++ {
		if !isOsArchChar(pkg.Os[i]) {
			return fmt.Errorf("package OS contains invalid char '%c'", pkg.Os[i])
		}
	}

	for i := 0; i < len(pkg.Arch); i++ {
		if !isOsArchChar(pkg.Arch[i]) {
			return fmt.Errorf("package Arch contains invalid char '%c'", pkg.Arch[i])
		}
	}

	return nil
}

func (pkg Package) PkgName() string {
	return fmt.Sprintf("%s_%s_%s_%s.ptar", pkg.Name, pkg.Version, pkg.Os, pkg.Arch)
}

func (pkg Package) PkgNameAndVersion() string {
	return fmt.Sprintf("%s@%s", pkg.Name, pkg.Version)
}

func (pkg Package) PluginName() string {
	return fmt.Sprintf("%s_%s_%s_%s", pkg.Name, pkg.Version, pkg.Os, pkg.Arch)
}

func fetchPackages(url string) ([]Package, error) {
	var packages []Package
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s: %w", url, err)
	}
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, resp.Status)
	}
	defer resp.Body.Close()

	var lst []struct {
		Name  string
		Type  string
		Mtime string
		Size  uint64
	}
	err = json.NewDecoder(resp.Body).Decode(&lst)
	if err != nil {
		return nil, fmt.Errorf("failed to decode package list: %w", err)
	}

	for _, e := range lst {
		var pkg Package
		if ParsePackageName(e.Name, &pkg) == nil {
			//fmt.Println(e.Name)
			packages = append(packages, pkg)
		}
	}

	return packages, nil
}

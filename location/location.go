package location

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/PlakarKorp/plakar/appcontext"
)

type Location[T any] struct {
	mtx      sync.Mutex
	items    map[string]T
	fallback string
}

func New[T any](fallback string) *Location[T] {
	return &Location[T]{
		items:    make(map[string]T),
		fallback: fallback,
	}
}

func (l *Location[T]) Register(name string, item T) bool {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	if _, ok := l.items[name]; ok {
		return false
	}
	l.items[name] = item
	return true
}

func (l *Location[T]) Names() []string {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	ret := make([]string, 0, len(l.items))
	for name := range l.items {
		ret = append(ret, name)
	}
	slices.Sort(ret)
	return ret
}

func allowedInUri(c rune) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') ||
		c == '+' || c == '-' || c == '.'
}

func (l *Location[T]) Lookup(uri string) (proto, location string, item T, ok bool) {
	l.mtx.Lock()
	defer l.mtx.Unlock()

	proto = uri
	location = uri

	for i, c := range uri {
		if !allowedInUri(c) {
			if i != 0 && strings.HasPrefix(uri[i:], ":") {
				proto = uri[:i]
				location = uri[i+1:]
				location = strings.TrimPrefix(location, "//")
			}
			break
		}
	}

	if proto == location {
		proto = l.fallback
	}

	item, ok = l.items[proto]
	return
}

type PluginInfo struct {
	Name        string `json:"backupName"`
}

func (l *Location[T]) LoadPlugin(ctx *appcontext.AppContext) error {
	l.mtx.Lock()
	defer l.mtx.Unlock()
	fmt.Printf("Loading plugins from %s\n", ctx.PluginsDir)
	fmt.Printf("fallback: %s\n", l.fallback)

	dirEntries, err := os.ReadDir(ctx.PluginsDir)
	if err != nil {
		return nil
	}
	fmt.Printf("Found %d plugins\n", len(dirEntries))
	for i := range dirEntries {
		dirEntry := dirEntries[i]
		name := dirEntry.Name()
		if dirEntry.IsDir() {
			fmt.Printf("Skipping directory: %s\n", filepath.Join(ctx.PluginsDir, name))
			pluginDirEntry, err := os.ReadDir(filepath.Join(ctx.PluginsDir, name))
			if err != nil {
				return nil
			}
			fmt.Printf("Found plugin directory: %s\n", name)
			for _, entry := range pluginDirEntry {
				if entry.IsDir() {
					continue
				}
				pluginPath := filepath.Join(ctx.PluginsDir, name, entry.Name())
				file := filepath.Base(pluginPath)
				fmt.Printf("Found plugin: %s %s\n", pluginPath, file)
				if file != ".info.json" {
					fmt.Printf("Skipping non-info file: %s\n", file)
					continue
				}
				pInfo, err := os.ReadFile(pluginPath)
				if err != nil {
					return nil
				}
				fmt.Printf("Plugin info: %s\n", string(pInfo))
				var jsonContent PluginInfo
				json.Unmarshal(pInfo, &jsonContent)	
				fmt.Printf("Plugin name: %s\n", jsonContent.Name)
				//l.Register(jsonContent.Name, nil)
			}
		}
	}
	return nil
}

package plugins

import (
	"testing"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/pkg"
	"github.com/stretchr/testify/require"
)

// Registration only stores a constructor closure; the plugin executable is never
// spawned unless the backend is actually opened, so these tests stay hermetic.

func TestRegisterImporter(t *testing.T) {
	proto := "test-plugin-importer"
	err := RegisterImporter(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.NoError(t, err)
	t.Cleanup(func() { importer.Unregister(proto) })

	// re-registering the same proto must fail
	err = RegisterImporter(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.Error(t, err)
}

func TestRegisterExporter(t *testing.T) {
	proto := "test-plugin-exporter"
	err := RegisterExporter(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.NoError(t, err)
	t.Cleanup(func() { exporter.Unregister(proto) })

	err = RegisterExporter(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.Error(t, err)
}

func TestRegisterStorage(t *testing.T) {
	proto := "test-plugin-storage"
	err := RegisterStorage(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.NoError(t, err)
	t.Cleanup(func() { storage.Unregister(proto) })

	err = RegisterStorage(proto, location.Flags(0), "/nonexistent/exe", nil)
	require.Error(t, err)
}

func TestLoadEmptyManifest(t *testing.T) {
	m := &pkg.Manifest{}
	require.NoError(t, Load(m, t.TempDir()))
}

func TestLoadUnknownConnectorTypeIgnored(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{
				Type:      "bogus-type",
				Protocols: []string{"whatever"},
			},
		},
	}
	// Unknown types are silently ignored, so Load succeeds without registering.
	require.NoError(t, Load(m, t.TempDir()))
}

func TestLoadAndUnloadImporter(t *testing.T) {
	proto := "test-load-importer"
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{
				Type:       "importer",
				Protocols:  []string{proto},
				Executable: "plugin-bin",
			},
		},
	}
	pkgdir := t.TempDir()

	require.NoError(t, Load(m, pkgdir))

	// Loading again must fail because the protocol is already registered.
	err := Load(m, pkgdir)
	require.Error(t, err)

	// Unload removes the registration so a subsequent Load works again.
	require.NoError(t, Unload(m))
	require.NoError(t, Load(m, pkgdir))
	require.NoError(t, Unload(m))
}

func TestLoadExporterAndStorage(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "exporter", Protocols: []string{"test-load-exporter"}, Executable: "e"},
			{Type: "storage", Protocols: []string{"test-load-storage"}, Executable: "s"},
		},
	}
	require.NoError(t, Load(m, t.TempDir()))
	t.Cleanup(func() { Unload(m) })

	require.NoError(t, Unload(m))
}

func TestUnloadUnknownTypeIgnored(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "bogus", Protocols: []string{"x"}},
		},
	}
	require.NoError(t, Unload(m))
}

func TestLoadInvalidLocationFlag(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{
				Type:          "importer",
				Protocols:     []string{"test-bad-flag"},
				LocationFlags: []string{"this-is-not-a-valid-flag"},
			},
		},
	}
	err := Load(m, t.TempDir())
	require.Error(t, err)
}

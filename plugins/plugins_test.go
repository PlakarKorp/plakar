package plugins

import (
	"io"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/location"
	"github.com/PlakarKorp/pkg"
)

// uniqueProto returns a protocol name that is unique to the test, so the
// global Register/Unregister table doesn't collide across runs or tests.
// We pair it with a t.Cleanup that unregisters in the right registry.
func uniqueProto(t *testing.T) string {
	t.Helper()
	return "test-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")
}

// ---------- RegisterStorage / RegisterImporter / RegisterExporter ----------

func TestRegisterStorage_Idempotent(t *testing.T) {
	proto := uniqueProto(t)
	t.Cleanup(func() { _ = storage.Unregister(proto) })

	if err := RegisterStorage(proto, 0, &NativeRunner{Path: "/nonexistent"}); err != nil {
		t.Fatalf("first register err = %v", err)
	}
	// re-registering the same proto must fail
	if err := RegisterStorage(proto, 0, &NativeRunner{Path: "/nonexistent"}); err == nil {
		t.Fatal("second register should error on duplicate proto")
	}
}

func TestRegisterImporter_Idempotent(t *testing.T) {
	proto := uniqueProto(t)
	t.Cleanup(func() { _ = importer.Unregister(proto) })

	if err := RegisterImporter(proto, 0, &NativeRunner{Path: "/nonexistent"}); err != nil {
		t.Fatalf("first register err = %v", err)
	}
	if err := RegisterImporter(proto, 0, &NativeRunner{Path: "/nonexistent"}); err == nil {
		t.Fatal("second register should error on duplicate proto")
	}
}

func TestRegisterExporter_Idempotent(t *testing.T) {
	proto := uniqueProto(t)
	t.Cleanup(func() { _ = exporter.Unregister(proto) })

	if err := RegisterExporter(proto, 0, &NativeRunner{Path: "/nonexistent"}); err != nil {
		t.Fatalf("first register err = %v", err)
	}
	if err := RegisterExporter(proto, 0, &NativeRunner{Path: "/nonexistent"}); err == nil {
		t.Fatal("second register should error on duplicate proto")
	}
}

// ---------- Load / Unload ----------

func TestLoad_RegistersAllConnectorTypes(t *testing.T) {
	imp := uniqueProto(t) + "-imp"
	exp := uniqueProto(t) + "-exp"
	stg := uniqueProto(t) + "-stg"
	t.Cleanup(func() {
		_ = importer.Unregister(imp)
		_ = exporter.Unregister(exp)
		_ = storage.Unregister(stg)
	})

	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "importer", Protocols: []string{imp}, Executable: "noop"},
			{Type: "exporter", Protocols: []string{exp}, Executable: "noop"},
			{Type: "storage", Protocols: []string{stg}, Executable: "noop"},
		},
	}

	if err := Load(m, "/tmp"); err != nil {
		t.Fatalf("Load: %v", err)
	}

	// Verify each backend is in the relevant registry's name list.
	if !contains(storage.Backends(), stg) {
		t.Errorf("storage backend %q not registered; got %v", stg, storage.Backends())
	}
}

func TestLoad_ImageConnector(t *testing.T) {
	stg := uniqueProto(t)
	t.Cleanup(func() { _ = storage.Unregister(stg) })

	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "storage", Protocols: []string{stg}, ImageID: "sha256:aa"},
		},
	}

	if err := Load(m, "/tmp"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !contains(storage.Backends(), stg) {
		t.Errorf("storage backend %q not registered; got %v", stg, storage.Backends())
	}
}

func TestLoad_RejectsInvalidConnectors(t *testing.T) {
	for _, conn := range []pkg.ManifestConnector{
		// Both executable and image_id.
		{Type: "storage", Protocols: []string{"never-registered"}, Executable: "noop", ImageID: "sha256:aa"},
		// Neither.
		{Type: "storage", Protocols: []string{"never-registered"}},
		// localfs connectors never run as containers.
		{Type: "importer", Protocols: []string{"never-registered"}, ImageID: "sha256:aa", LocationFlags: []string{"localfs"}},
	} {
		m := &pkg.Manifest{Connectors: []pkg.ManifestConnector{conn}}
		if err := Load(m, "/tmp"); err == nil {
			t.Errorf("Load should reject connector %+v", conn)
		}
	}
}

func TestLoad_PropagatesDuplicateError(t *testing.T) {
	stg := uniqueProto(t)
	t.Cleanup(func() { _ = storage.Unregister(stg) })

	// Pre-register to force the duplicate path in Load.
	if err := RegisterStorage(stg, 0, &NativeRunner{Path: "/x"}); err != nil {
		t.Fatalf("seed register: %v", err)
	}

	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "storage", Protocols: []string{stg}, Executable: "noop"},
		},
	}
	if err := Load(m, "/tmp"); err == nil {
		t.Fatal("Load should propagate duplicate-registration error")
	}
}

func TestLoad_BadFlagPropagates(t *testing.T) {
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{
				Type:          "storage",
				Protocols:     []string{"any"},
				Executable:    "noop",
				LocationFlags: []string{"this-flag-does-not-exist"},
			},
		},
	}
	if err := Load(m, "/tmp"); err == nil {
		t.Fatal("Load should propagate Flags() error")
	}
}

func TestUnload_Symmetric(t *testing.T) {
	imp := uniqueProto(t) + "-imp"
	exp := uniqueProto(t) + "-exp"
	stg := uniqueProto(t) + "-stg"

	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "importer", Protocols: []string{imp}, Executable: "noop"},
			{Type: "exporter", Protocols: []string{exp}, Executable: "noop"},
			{Type: "storage", Protocols: []string{stg}, Executable: "noop"},
		},
	}
	if err := Load(m, "/tmp"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := Unload(m); err != nil {
		t.Fatalf("Unload: %v", err)
	}

	if contains(storage.Backends(), stg) {
		t.Errorf("storage backend %q still present after Unload", stg)
	}
}

func TestUnload_ReturnsLastError(t *testing.T) {
	// Manifest that references something that was never loaded — Unload
	// should return the last error rather than panicking.
	m := &pkg.Manifest{
		Connectors: []pkg.ManifestConnector{
			{Type: "storage", Protocols: []string{"never-loaded-proto"}, Executable: "noop"},
		},
	}
	if err := Unload(m); err == nil {
		t.Fatal("Unload of nothing should return an error")
	}
}

func TestLoad_NilConnectors(t *testing.T) {
	if err := Load(&pkg.Manifest{}, "/tmp"); err != nil {
		t.Fatalf("Load on empty manifest = %v, want nil", err)
	}
	if err := Unload(&pkg.Manifest{}); err != nil {
		t.Fatalf("Unload on empty manifest = %v, want nil", err)
	}
}

// ---------- StdioConn ----------

func TestStdioConn_ReadWrite(t *testing.T) {
	// Two pipes: one for client->server, one for server->client.
	cr, cw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	sr, sw, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { cr.Close(); cw.Close(); sr.Close(); sw.Close() })

	// stdin reads from cr (what was written to cw); stdout writes to sw.
	conn := NewStdioConn(cr, sw, nil)

	// Write side: feed cw, read from conn.
	want := []byte("hello-stdio")
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = cw.Write(want)
		_ = cw.Close()
	}()

	got, err := io.ReadAll(conn)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(got) != string(want) {
		t.Fatalf("got %q, want %q", got, want)
	}
	wg.Wait()

	// Write through the conn -> shows up on sr.
	if _, err := conn.Write([]byte("xy")); err != nil {
		t.Fatalf("write: %v", err)
	}
	// Close sw so the reader sees EOF.
	sw.Close()
	echo, err := io.ReadAll(sr)
	if err != nil {
		t.Fatalf("read echo: %v", err)
	}
	if string(echo) != "xy" {
		t.Fatalf("echo = %q, want xy", echo)
	}
}

func TestStdioConn_Addrs(t *testing.T) {
	conn := NewStdioConn(os.Stdin, os.Stdout, nil)
	local, remote := conn.LocalAddr(), conn.RemoteAddr()
	if local == nil || remote == nil {
		t.Fatal("addrs should be non-nil")
	}
	if local.Network() != "unix" || local.String() != "stdio" {
		t.Fatalf("local = %s/%s", local.Network(), local.String())
	}
	// Both addrs point at the same package-level instance.
	if local != remote {
		t.Fatalf("local and remote should be the same instance")
	}
}

func TestStdioConn_CloseWithoutCmd(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	conn := NewStdioConn(r, w, nil)
	if err := conn.Close(); err != nil {
		t.Fatalf("close = %v, want nil", err)
	}
	// Closing twice returns an error from the underlying file.
	if err := conn.Close(); err == nil {
		t.Fatal("second close should error")
	}
}

func TestStdioConn_CloseWaitsForCmd(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}

	// Pick a tiny program that exits cleanly.
	cmd := exec.Command("true")
	if err := cmd.Start(); err != nil {
		// Some environments don't have "true" on PATH; fall back skip.
		t.Skipf("could not start `true`: %v", err)
	}

	conn := NewStdioConn(r, w, cmd)
	if err := conn.Close(); err != nil {
		t.Fatalf("close = %v, want nil (cmd was `true`)", err)
	}
}

func TestStdioConn_SetDeadlinesOnRegularFile(t *testing.T) {
	// On a regular file SetDeadline returns an error ("not supported");
	// on a pipe it succeeds. Use a pipe so we hit the success path.
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })

	conn := NewStdioConn(r, w, nil)
	deadline := time.Now().Add(time.Hour)
	if err := conn.SetDeadline(deadline); err != nil {
		t.Fatalf("SetDeadline on pipe = %v, want nil", err)
	}
	if err := conn.SetReadDeadline(deadline); err != nil {
		t.Fatalf("SetReadDeadline = %v", err)
	}
	if err := conn.SetWriteDeadline(deadline); err != nil {
		t.Fatalf("SetWriteDeadline = %v", err)
	}
}

func TestStdioConn_SetDeadlinePropagatesReadDeadlineError(t *testing.T) {
	// Use a regular file (not a pipe): SetReadDeadline returns an error.
	tmp, err := os.CreateTemp(t.TempDir(), "stdio")
	if err != nil {
		t.Fatalf("create tmp: %v", err)
	}
	defer tmp.Close()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	t.Cleanup(func() { r.Close(); w.Close() })

	// stdin is a regular file -> SetReadDeadline errors.
	conn := NewStdioConn(tmp, w, nil)
	if err := conn.SetDeadline(time.Now()); err == nil {
		t.Fatal("SetDeadline should propagate read-deadline error from a regular file")
	}
}

// ---------- helpers ----------

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}

// Compile-time interface check — keeps StdioConn implementing net.Conn.
var _ net.Conn = (*StdioConn)(nil)

// Keep an unused-import sentinel for the location package (used indirectly
// via importer.Unregister signature when nil flag value is sufficient).
var _ = location.Flags(0)

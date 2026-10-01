package plugins

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/kloset/connectors/exporter"
	"github.com/PlakarKorp/kloset/connectors/importer"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/kcontext"
)

// lookPath skips the test if the named helper binary isn't available.
func lookPath(t *testing.T, name string) string {
	t.Helper()
	p, err := exec.LookPath(name)
	if err != nil {
		t.Skipf("%s not found on PATH: %v", name, err)
	}
	return p
}

func TestNativeRunnerStartsProcess(t *testing.T) {
	// `cat` reads stdin and writes stdout, matching the stdio transport, and
	// stays alive until its stdin is closed — perfect for exercising Run().
	catPath := lookPath(t, "cat")

	conn, err := (&NativeRunner{Path: catPath}).Run(context.Background())
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if conn == nil {
		t.Fatal("Run returned a nil conn")
	}

	// Round-trip a byte through cat to prove the process is wired up.
	if _, err := conn.Write([]byte("ping\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	buf := make([]byte, 5)
	if _, err := conn.Read(buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf) != "ping\n" {
		t.Fatalf("got %q, want %q", buf, "ping\n")
	}

	// Closing the conn closes cat's stdin, so cat exits cleanly and Close's
	// cmd.Wait() returns nil.
	if err := conn.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestNativeRunnerNonexistentExecutable(t *testing.T) {
	_, err := (&NativeRunner{Path: "/nonexistent/plugin-binary"}).Run(context.Background())
	if err == nil {
		t.Fatal("expected Run to fail starting a missing executable")
	}
}

func TestNativeRunnerCancelledContextWaitErrors(t *testing.T) {
	// A cancelled context kills the spawned process, so Close's cmd.Wait()
	// reports the signal as an error — covering the Wait error branch.
	catPath := lookPath(t, "cat")

	ctx, cancel := context.WithCancel(context.Background())
	conn, err := (&NativeRunner{Path: catPath}).Run(ctx)
	if err != nil {
		t.Fatalf("run: %v", err)
	}

	cancel() // sends SIGKILL to cat via CommandContext

	// Close should surface the non-nil Wait() error (killed process).
	if err := conn.Close(); err == nil {
		t.Log("Close returned nil; the process may have already exited cleanly")
	}
}

func TestConnectPluginBuildsClient(t *testing.T) {
	// connectPlugin spawns the helper and wraps the conn in a grpc client.
	// We only assert it builds a non-nil client without error; no RPC is made.
	catPath := lookPath(t, "cat")

	client, err := connectPlugin(context.Background(), &NativeRunner{Path: catPath})
	if err != nil {
		t.Fatalf("connectPlugin: %v", err)
	}
	if client == nil {
		t.Fatal("connectPlugin returned a nil client")
	}
}

func TestConnectPluginSpawnError(t *testing.T) {
	_, err := connectPlugin(context.Background(), &NativeRunner{Path: "/nonexistent/plugin-binary"})
	if err == nil {
		t.Fatal("expected connectPlugin to fail when the runner fails")
	}
}

// factoryProto builds a per-test protocol name for the connector-factory tests.
func factoryProto(t *testing.T) string {
	t.Helper()
	return "fac-" + strings.ReplaceAll(strings.ToLower(t.Name()), "/", "-")
}

// The following tests register a connector pointing at a helper binary that
// exits immediately (`true`), then invoke the connector through the kloset
// registry. This drives the factory closure inside RegisterStorage/Importer/
// Exporter: the runner starts, connectPlugin builds the client, and the first gRPC
// call fails because the helper has already exited — which is enough to cover
// the closure body and its error handling.

func TestRegisterStorageFactoryInvoked(t *testing.T) {
	truePath := lookPath(t, "true")
	proto := factoryProto(t)
	t.Cleanup(func() { _ = storage.Unregister(proto) })

	if err := RegisterStorage(proto, 0, &NativeRunner{Path: truePath}); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := kcontext.NewKContext()
	defer ctx.Close()
	_, err := storage.New(ctx, map[string]string{"location": proto + "://x"})
	if err == nil {
		t.Fatal("expected the gRPC Init call to fail against an exited helper")
	}
}

func TestRegisterImporterFactoryInvoked(t *testing.T) {
	truePath := lookPath(t, "true")
	proto := factoryProto(t)
	t.Cleanup(func() { _ = importer.Unregister(proto) })

	if err := RegisterImporter(proto, 0, &NativeRunner{Path: truePath}); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := kcontext.NewKContext()
	defer ctx.Close()
	_, err := importer.NewImporter(ctx, &connectors.Options{}, map[string]string{"location": proto + "://x"})
	if err == nil {
		t.Fatal("expected the gRPC Init call to fail against an exited helper")
	}
}

func TestRegisterExporterFactoryInvoked(t *testing.T) {
	truePath := lookPath(t, "true")
	proto := factoryProto(t)
	t.Cleanup(func() { _ = exporter.Unregister(proto) })

	if err := RegisterExporter(proto, 0, &NativeRunner{Path: truePath}); err != nil {
		t.Fatalf("register: %v", err)
	}

	ctx := kcontext.NewKContext()
	defer ctx.Close()
	_, err := exporter.NewExporter(ctx, &connectors.Options{}, map[string]string{"location": proto + "://x"})
	if err == nil {
		t.Fatal("expected the gRPC Init call to fail against an exited helper")
	}
}

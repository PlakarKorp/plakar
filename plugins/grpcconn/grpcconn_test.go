package grpcconn_test

import (
	"context"
	"errors"
	"net"
	"slices"
	"sync"
	"testing"
	"time"

	grpc_exporter "github.com/PlakarKorp/integration-grpc/exporter"
	grpc_importer "github.com/PlakarKorp/integration-grpc/importer"
	grpc_storage "github.com/PlakarKorp/integration-grpc/storage"
	"github.com/PlakarKorp/kloset/connectors"
	"github.com/PlakarKorp/plakar/plugins/grpcconn"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// fakePlugin records, in order, the Close RPC reaching the plugin and the
// host closing the transport.
type fakePlugin struct {
	mu       sync.Mutex
	events   []string
	initErr  error
	closeErr error

	// closeHang, when set, makes Close block until it is closed, like a
	// plugin stuck in its cleanup.
	closeHang chan struct{}
}

func (p *fakePlugin) record(ev string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, ev)
}

func (p *fakePlugin) Events() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.events)
}

func (p *fakePlugin) close() error {
	p.record("close rpc")
	if p.closeHang != nil {
		<-p.closeHang
	}
	return p.closeErr
}

// dial serves the plugin in-process over a net.Pipe and returns a client
// conn to it, built the way plakar builds the one to a plugin process.
func dial(t *testing.T, plugin *fakePlugin, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	host, remote := net.Pipe()
	srv := grpc.NewServer()
	register(srv)
	go srv.Serve(newOneConnListener(remote))
	t.Cleanup(srv.Stop)

	conn := &recordingConn{Conn: host, plugin: plugin}
	client, err := grpc.NewClient("127.0.0.1:0",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return conn, nil
		}),
	)
	if err != nil {
		t.Fatalf("grpc client: %v", err)
	}
	return client
}

type recordingConn struct {
	net.Conn
	plugin *fakePlugin
	once   sync.Once
}

func (c *recordingConn) Close() error {
	c.once.Do(func() { c.plugin.record("transport closed") })
	return c.Conn.Close()
}

// oneConnListener hands out a single conn, then blocks until closed.
type oneConnListener struct {
	conns chan net.Conn
	done  chan struct{}
	once  sync.Once
	addr  net.Addr
}

func newOneConnListener(conn net.Conn) *oneConnListener {
	l := &oneConnListener{conns: make(chan net.Conn, 1), done: make(chan struct{}), addr: conn.LocalAddr()}
	l.conns <- conn
	return l
}

func (l *oneConnListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.done:
		return nil, net.ErrClosed
	}
}

func (l *oneConnListener) Close() error {
	l.once.Do(func() { close(l.done) })
	return nil
}

func (l *oneConnListener) Addr() net.Addr { return l.addr }

type fakeStoreServer struct {
	grpc_storage.UnimplementedStoreServer
	plugin *fakePlugin
}

func (s *fakeStoreServer) Init(context.Context, *grpc_storage.InitRequest) (*grpc_storage.InitResponse, error) {
	return &grpc_storage.InitResponse{}, s.plugin.initErr
}

func (s *fakeStoreServer) Close(context.Context, *grpc_storage.CloseRequest) (*grpc_storage.CloseResponse, error) {
	return &grpc_storage.CloseResponse{}, s.plugin.close()
}

type fakeImporterServer struct {
	grpc_importer.UnimplementedImporterServer
	plugin *fakePlugin
}

func (s *fakeImporterServer) Init(context.Context, *grpc_importer.InitRequest) (*grpc_importer.InitResponse, error) {
	return &grpc_importer.InitResponse{}, s.plugin.initErr
}

func (s *fakeImporterServer) Close(context.Context, *grpc_importer.CloseRequest) (*grpc_importer.CloseResponse, error) {
	return &grpc_importer.CloseResponse{}, s.plugin.close()
}

type fakeExporterServer struct {
	grpc_exporter.UnimplementedExporterServer
	plugin *fakePlugin
}

func (s *fakeExporterServer) Init(context.Context, *grpc_exporter.InitRequest) (*grpc_exporter.InitResponse, error) {
	return &grpc_exporter.InitResponse{}, s.plugin.initErr
}

func (s *fakeExporterServer) Close(context.Context, *grpc_exporter.CloseRequest) (*grpc_exporter.CloseResponse, error) {
	return &grpc_exporter.CloseResponse{}, s.plugin.close()
}

type closer interface {
	Close(context.Context) error
}

// connectorKinds opens a connector of each kind against a fake plugin.
var connectorKinds = []struct {
	name string
	open func(t *testing.T, plugin *fakePlugin) (closer, error)
}{
	{"storage", func(t *testing.T, plugin *fakePlugin) (closer, error) {
		conn := dial(t, plugin, func(s *grpc.Server) {
			grpc_storage.RegisterStoreServer(s, &fakeStoreServer{plugin: plugin})
		})
		return grpcconn.NewStorage(context.Background(), conn, "fake", nil)
	}},
	{"importer", func(t *testing.T, plugin *fakePlugin) (closer, error) {
		conn := dial(t, plugin, func(s *grpc.Server) {
			grpc_importer.RegisterImporterServer(s, &fakeImporterServer{plugin: plugin})
		})
		return grpcconn.NewImporter(context.Background(), conn, &connectors.Options{}, "fake", nil)
	}},
	{"exporter", func(t *testing.T, plugin *fakePlugin) (closer, error) {
		conn := dial(t, plugin, func(s *grpc.Server) {
			grpc_exporter.RegisterExporterServer(s, &fakeExporterServer{plugin: plugin})
		})
		return grpcconn.NewExporter(context.Background(), conn, &connectors.Options{}, "fake", nil)
	}},
}

func TestCloseSendsRPCThenClosesTransport(t *testing.T) {
	for _, kind := range connectorKinds {
		t.Run(kind.name, func(t *testing.T) {
			plugin := &fakePlugin{}

			conn, err := kind.open(t, plugin)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if got := plugin.Events(); len(got) != 0 {
				t.Fatalf("events before Close = %q, want none", got)
			}
			if err := conn.Close(context.Background()); err != nil {
				t.Fatalf("close: %v", err)
			}

			want := []string{"close rpc", "transport closed"}
			if got := plugin.Events(); !slices.Equal(got, want) {
				t.Fatalf("events = %q, want %q", got, want)
			}
		})
	}
}

func TestCloseReachesPluginOnCancelledContext(t *testing.T) {
	for _, kind := range connectorKinds {
		t.Run(kind.name, func(t *testing.T) {
			plugin := &fakePlugin{}

			conn, err := kind.open(t, plugin)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if err := conn.Close(ctx); err != nil {
				t.Fatalf("close: %v", err)
			}

			want := []string{"close rpc", "transport closed"}
			if got := plugin.Events(); !slices.Equal(got, want) {
				t.Fatalf("events = %q, want %q", got, want)
			}
		})
	}
}

func TestCloseClosesTransportWhenRPCFails(t *testing.T) {
	for _, kind := range connectorKinds {
		t.Run(kind.name, func(t *testing.T) {
			plugin := &fakePlugin{closeErr: errors.New("flush failed")}

			conn, err := kind.open(t, plugin)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			if err := conn.Close(context.Background()); err == nil {
				t.Fatal("expected the plugin's Close error to be returned")
			}

			want := []string{"close rpc", "transport closed"}
			if got := plugin.Events(); !slices.Equal(got, want) {
				t.Fatalf("events = %q, want %q", got, want)
			}
		})
	}
}

func TestConstructorFailureClosesTransport(t *testing.T) {
	for _, kind := range connectorKinds {
		t.Run(kind.name, func(t *testing.T) {
			plugin := &fakePlugin{initErr: errors.New("bad config")}

			if _, err := kind.open(t, plugin); err == nil {
				t.Fatal("expected the plugin's Init error to be returned")
			}

			want := []string{"transport closed"}
			if got := plugin.Events(); !slices.Equal(got, want) {
				t.Fatalf("events = %q, want %q", got, want)
			}
		})
	}
}

func TestCloseIsBoundedWhenPluginHangs(t *testing.T) {
	grpcconn.SetCloseTimeout(t, 100*time.Millisecond)

	for _, kind := range connectorKinds {
		t.Run(kind.name, func(t *testing.T) {
			plugin := &fakePlugin{closeHang: make(chan struct{})}
			conn, err := kind.open(t, plugin)
			if err != nil {
				t.Fatalf("open: %v", err)
			}
			// Runs before dial's cleanup stops the server, which would
			// otherwise wait on the stuck handler.
			t.Cleanup(func() { close(plugin.closeHang) })

			start := time.Now()
			err = conn.Close(context.Background())
			if elapsed := time.Since(start); elapsed > 5*time.Second {
				t.Fatalf("Close took %s on a hung plugin", elapsed)
			}
			// integration-grpc flattens the gRPC status into a string, so
			// there is no deadline error to match on.
			if err == nil {
				t.Fatal("expected Close to fail on a hung plugin")
			}

			want := []string{"close rpc", "transport closed"}
			if got := plugin.Events(); !slices.Equal(got, want) {
				t.Fatalf("events = %q, want %q", got, want)
			}
		})
	}
}

package server

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	_ "github.com/PlakarKorp/integrations/fs/exporter"
	"github.com/PlakarKorp/kloset/connectors/storage"
	"github.com/PlakarKorp/kloset/hashing"
	"github.com/PlakarKorp/kloset/resources"
	"github.com/PlakarKorp/kloset/versioning"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

func init() {
	os.Setenv("TZ", "UTC")
}

func TestServerParseAllowDelete(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()
	_ = repo

	cmd := &Server{}
	require.NoError(t, cmd.Parse(ctx, []string{"-allow-delete"}))
	require.False(t, cmd.NoDelete, "-allow-delete should clear NoDelete")
}

func TestServerParseDefaultsToNoDelete(t *testing.T) {
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()
	_ = repo

	cmd := &Server{}
	require.NoError(t, cmd.Parse(ctx, []string{}))
	require.True(t, cmd.NoDelete, "delete must be disabled by default")
}

func TestServerExecuteTLSErrorReturns1(t *testing.T) {
	// cert+key set (https protocol branch) but pointing at missing files, so
	// httpd.Server returns immediately with an error and Execute reports 1.
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)
	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	defer ctx.Close()

	cmd := &Server{}
	require.NoError(t, cmd.Parse(ctx, []string{
		"-listen", "127.0.0.1:0",
		"-cert", "/nonexistent/cert.pem",
		"-key", "/nonexistent/key.pem",
	}))

	status, err := cmd.Execute(ctx, repo)
	require.Error(t, err)
	require.Equal(t, 1, status)
	// The https protocol branch was taken when logging the listen line.
	require.Contains(t, bufOut.String()+bufErr.String(), "https://")
}

func TestExecuteCmdServerDefault(t *testing.T) {
	bufOut := bytes.NewBuffer(nil)
	bufErr := bytes.NewBuffer(nil)

	repo, ctx := ptesting.GenerateRepository(t, bufOut, bufErr, nil)
	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockDir("another_subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
		ptesting.NewMockFile("subdir/foo.txt", 0644, "hello foo"),
		ptesting.NewMockFile("subdir/to_exclude", 0644, "*/subdir/to_exclude\n"),
		ptesting.NewMockFile("another_subdir/bar.txt", 0644, "hello bar"),
	})
	snap.Close()

	defer ctx.Close()

	args := []string{"-listen", "127.0.0.1:12345"}

	subcommand := &Server{}
	err := subcommand.Parse(ctx, args)

	require.NoError(t, err)
	require.NotNil(t, subcommand)

	go subcommand.Execute(ctx, repo)

	// wait for the server to start
	time.Sleep(100 * time.Millisecond)

	req, err := http.NewRequest("GET", "http://localhost:12345/", nil)
	require.NoError(t, err, "creating request")

	client := &http.Client{}
	w := httptest.NewRecorder()
	require.Equal(t, http.StatusOK, w.Code, "expected status code 200")

	response, err := client.Do(req)
	require.NoError(t, err, "making request")

	rawBody, err := io.ReadAll(response.Body)
	require.NoError(t, err, "reading response")

	hasher := hashing.GetHasher(hashing.DEFAULT_HASHING_ALGORITHM)
	version, unwrappedConfigRd, err := storage.Deserialize(hasher, resources.RT_CONFIG, io.NopCloser(bytes.NewReader(rawBody)))
	require.NoError(t, err, "deserializing configuration")

	unwrappedConfig, err := io.ReadAll(unwrappedConfigRd)
	require.NoError(t, err, "reading deserializing configuration")

	configInstance, err := storage.NewConfigurationFromBytes(version, unwrappedConfig)
	require.NoError(t, err, "reading configuration")

	// we dont test all the field from configuration
	require.Equal(t, versioning.FromString(storage.VERSION), configInstance.Version)
}

func TestServerParseToken(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"omitted keeps today's behaviour", []string{}, false},
		{"set", []string{"-token", "secret"}, false},
		{"explicitly empty", []string{"-token", ""}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Server{}).Parse(ctx, tt.args)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestServerParseCertKeyPair(t *testing.T) {
	_, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	defer ctx.Close()

	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{"neither", []string{}, false},
		{"both", []string{"-cert", "c.pem", "-key", "k.pem"}, false},
		{"cert only", []string{"-cert", "c.pem"}, true},
		{"key only", []string{"-key", "k.pem"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := (&Server{}).Parse(ctx, tt.args)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

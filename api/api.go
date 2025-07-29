package api

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/PlakarKorp/kloset/repository"
	"github.com/PlakarKorp/kloset/snapshot"
	"github.com/PlakarKorp/kloset/storage"
	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

type uiserver struct {
	store      storage.Store
	config     storage.Configuration
	repository *repository.Repository

	// XXX: Adding this for transition, it needs to go away. Some
	// places we only have Repository and out of AppContext we
	// only get a KContext, except sometimes you truly need an
	// AppContext.
	ctx *appcontext.AppContext
}

type Item[T any] struct {
	Item T `json:"item"`
}

type Items[T any] struct {
	Total int `json:"total"`
	Items []T `json:"items"`
}

type ItemsPage[T any] struct {
	HasNext bool `json:"has_next"`
	Items   []T  `json:"items"`
}

type ApiErrorRes struct {
	Error *ApiError `json:"error"`
}

func handleError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, repository.ErrNotReadable):
		err = &ApiError{
			HttpCode: 400,
			ErrCode:  "bad-request",
			Message:  err.Error(),
		}
	case errors.Is(err, repository.ErrBlobNotFound):
		fallthrough
	case errors.Is(err, repository.ErrPackfileNotFound):
		fallthrough
	case errors.Is(err, fs.ErrNotExist):
		fallthrough
	case errors.Is(err, snapshot.ErrNotFound):
		err = &ApiError{
			HttpCode: 404,
			ErrCode:  "not-found",
			Message:  err.Error(),
		}
	}

	apierr, ok := err.(*ApiError)
	if !ok {
		log.Printf("Unknown error encountered while serving %s: %v", r.URL, err)
		apierr = &ApiError{
			HttpCode: 500,
			ErrCode:  "internal-error",
			Message:  "Internal server error. Check server logs for more information.",
		}
	}

	w.WriteHeader(apierr.HttpCode)
	_ = json.NewEncoder(w).Encode(&ApiErrorRes{apierr})
}

type JSONAPIView func(w http.ResponseWriter, r *http.Request) error

func (view JSONAPIView) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := view(w, r); err != nil {
		handleError(w, r, err)
	}
}

type APIView func(w http.ResponseWriter, r *http.Request) error

func (view APIView) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := view(w, r); err != nil {
		w.Header().Set("Content-Type", "application/json")
		handleError(w, r, err)
	}
}

// TokenAuthMiddleware is a middleware that checks for the token in the request. If the token is empty, the middleware is a no-op.
func TokenAuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token != "" {
				key := r.Header.Get("Authorization")
				if key == "" {
					handleError(w, r, authError("missing Authorization header"))
					return
				}

				if subtle.ConstantTimeCompare([]byte(key), []byte("Bearer "+token)) == 0 {
					handleError(w, r, authError("invalid token"))
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
}

func (ui *uiserver) apiInfo(w http.ResponseWriter, r *http.Request) error {
	authenticated := false
	configuration := ui.config
	if authToken, err := ui.ctx.GetCookies().GetAuthToken(); err == nil && authToken != "" {
		authenticated = true
	}

	isDemoMode, _ := strconv.ParseBool(os.Getenv("PLAKAR_DEMO_MODE"))

	mode, err := ui.store.Mode(r.Context())
	if err != nil {
		return err
	}

	res := &struct {
		RepositoryId  string `json:"repository_id"`
		Authenticated bool   `json:"authenticated"`
		Version       string `json:"version"`
		Browsable     bool   `json:"browsable"`
		DemoMode      bool   `json:"demo_mode"`
	}{
		RepositoryId:  configuration.RepositoryID.String(),
		Authenticated: authenticated,
		Version:       utils.GetVersion(),
		Browsable:     mode&storage.ModeRead != 0,
		DemoMode:      isDemoMode,
	}
	return json.NewEncoder(w).Encode(res)
}

func SetupRoutes(server *http.ServeMux, repo *repository.Repository, ctx *appcontext.AppContext, token string) {
	ui := uiserver{
		store:      repo.Store(),
		config:     repo.Configuration(),
		repository: repo,
		ctx:        ctx,
	}

	authToken := TokenAuthMiddleware(token)
	urlSigner := NewSnapshotReaderURLSigner(&ui, token)

	// Catch all API endpoint, called if no more specific API endpoint is found
	server.Handle("/api/", JSONAPIView(func(w http.ResponseWriter, r *http.Request) error {
		return &ApiError{
			HttpCode: 404,
			ErrCode:  "not-found",
			Message:  "API endpoint not found",
		}
	}))

	isDemoMode, _ := strconv.ParseBool(os.Getenv("PLAKAR_DEMO_MODE"))

	server.Handle("GET /api/info", authToken(JSONAPIView(ui.apiInfo)))

	// The demo mode is the read-only mode of the API available at demo.plakar.io. Disable the write operations.
	if !isDemoMode {
		server.Handle("POST /api/authentication/login/github", authToken(JSONAPIView(ui.servicesLoginGithub)))
		server.Handle("POST /api/authentication/login/email", authToken(JSONAPIView(ui.servicesLoginEmail)))
		server.Handle("POST /api/authentication/logout", authToken(JSONAPIView(ui.servicesLogout)))

		server.Handle("POST /api/proxy/v1/account/notifications/set-status", authToken(JSONAPIView(ui.servicesProxy)))
		server.Handle("PUT /api/proxy/v1/account/services/alerting", authToken(JSONAPIView(ui.servicesSetAlertingServiceConfiguration)))
	}

	server.Handle("GET /api/proxy/v1/account/me", authToken(JSONAPIView(ui.servicesProxy)))
	server.Handle("GET /api/proxy/v1/account/notifications", authToken(JSONAPIView(ui.servicesProxy)))
	server.Handle("GET /api/proxy/v1/account/services/alerting", authToken(JSONAPIView(ui.servicesGetAlertingServiceConfiguration)))
	server.Handle("GET /api/proxy/v1/reporting/reports", authToken(JSONAPIView(ui.servicesProxy)))

	server.Handle("GET /api/repository/info", authToken(JSONAPIView(ui.repositoryInfo)))
	server.Handle("GET /api/repository/snapshots", authToken(JSONAPIView(ui.repositorySnapshots)))
	server.Handle("GET /api/repository/locate-pathname", authToken(JSONAPIView(ui.repositoryLocatePathname)))
	server.Handle("GET /api/repository/importer-types", authToken(JSONAPIView(ui.repositoryImporterTypes)))
	server.Handle("GET /api/repository/states", authToken(JSONAPIView(ui.repositoryStates)))
	server.Handle("GET /api/repository/state/{state}", authToken(JSONAPIView(ui.repositoryState)))

	server.Handle("GET /api/snapshot/{snapshot}", authToken(JSONAPIView(ui.snapshotHeader)))
	server.Handle("GET /api/snapshot/reader/{snapshot_path...}", urlSigner.VerifyMiddleware(APIView(ui.snapshotReader)))
	server.Handle("POST /api/snapshot/reader-sign-url/{snapshot_path...}", authToken(JSONAPIView(urlSigner.Sign)))

	server.Handle("GET /api/snapshot/vfs/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSBrowse)))
	server.Handle("GET /api/snapshot/vfs/children/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSChildren)))
	server.Handle("GET /api/snapshot/vfs/chunks/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSChunks)))
	server.Handle("GET /api/snapshot/vfs/search/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSSearch)))
	server.Handle("GET /api/snapshot/vfs/errors/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSErrors)))

	server.Handle("POST /api/snapshot/vfs/downloader/{snapshot_path...}", authToken(JSONAPIView(ui.snapshotVFSDownloader)))
	server.Handle("GET /api/snapshot/vfs/downloader-sign-url/{id}", JSONAPIView(ui.snapshotVFSDownloaderSigned))
}

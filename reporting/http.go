package reporting

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"

	errorspkg "github.com/PlakarKorp/plakar/internal/errors"
	"github.com/PlakarKorp/plakar/utils"
)

type HttpEmitter struct {
	url    string
	token  string
	client http.Client
}

func (emitter *HttpEmitter) Emit(ctx context.Context, report *Report) error {
	data, err := json.Marshal(report)
	if err != nil {
		return errorspkg.Wrap(ErrEncodeReport, err, "failed to encode report",
			errorspkg.WithContext("report", report.Task))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", emitter.url, bytes.NewReader(data))
	if err != nil {
		return errorspkg.Wrap(ErrBuildRequest, err, "failed to create request",
			errorspkg.WithContext("url", emitter.url))
	}
	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", utils.VERSION, runtime.GOOS, runtime.GOARCH))
	if emitter.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", emitter.token))
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := emitter.client.Do(req)
	if err != nil {
		return errorspkg.Wrap(ErrDoRequest, err, "failed to send report",
			errorspkg.WithContext("url", emitter.url))
	}
	res.Body.Close()
	if 200 <= res.StatusCode && res.StatusCode < 300 {
		return nil
	}
	return errorspkg.New(ErrBadStatus, "request failed", errorspkg.WithContext("status", res.Status))
}

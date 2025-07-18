package reporting

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/utils"
)

type HttpEmitter struct {
	url   string
	token string
	retry uint8
}

func (emitter *HttpEmitter) Emit(report *Report, logger *logging.Logger) {
	data, err := json.Marshal(report)
	if err != nil {
		logger.Error("failed to encode report: %s", err)
		return
	}

	backoffUnit := time.Minute
	for i := range emitter.retry {
		err := emitter.tryEmit(data)
		if err == nil {
			return
		}
		logger.Warn("failed to emit report: %s", err)
		time.Sleep(backoffUnit << i)
	}
	logger.Error("failed to emit report after %d attempts", emitter.retry)
}

func (reporter *HttpEmitter) tryEmit(data []byte) error {
	req, err := http.NewRequest("POST", reporter.url, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", utils.VERSION, runtime.GOOS, runtime.GOARCH))
	if reporter.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", reporter.token))
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if 200 <= res.StatusCode && res.StatusCode < 300 {
		return nil
	}

	return fmt.Errorf("request failed with status %s", res.Status)
}

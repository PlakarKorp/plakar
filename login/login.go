/*
 * Copyright (c) 2021 Gilles Chehade <gilles@poolp.org>
 *
 * Permission to use, copy, modify, and distribute this software for any
 * purpose with or without fee is hereby granted, provided that the above
 * copyright notice and this permission notice appear in all copies.
 *
 * THE SOFTWARE IS PROVIDED "AS IS" AND THE AUTHOR DISCLAIMS ALL WARRANTIES
 * WITH REGARD TO THIS SOFTWARE INCLUDING ALL IMPLIED WARRANTIES OF
 * MERCHANTABILITY AND FITNESS. IN NO EVENT SHALL THE AUTHOR BE LIABLE FOR
 * ANY SPECIAL, DIRECT, INDIRECT, OR CONSEQUENTIAL DAMAGES OR ANY DAMAGES
 * WHATSOEVER RESULTING FROM LOSS OF USE, DATA OR PROFITS, WHETHER IN AN
 * ACTION OF CONTRACT, NEGLIGENCE OR OTHER TORTIOUS ACTION, ARISING OUT OF
 * OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
 */

package login

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

type TokenResponse struct {
	Token string `json:"token"`
}

// defaultBaseURL is the plakar.io auth API root. It is overridable per-flow
// (via the baseURL field) so tests can point the login flow at a local server.
const defaultBaseURL = "https://api.plakar.io"

// ErrRateLimited marks a login failure caused by the auth API rate limiting the
// caller. It is wrapped into the returned error so callers can react to it with
// errors.Is without depending on a concrete error type or HTTP status code.
var ErrRateLimited = errors.New("rate limited")

// ErrUnknownPollID marks a poll for a sign-in the auth api does not know,
// or no longer knows: it expired or its token was already released.
var ErrUnknownPollID = errors.New("unknown ID")

type loginFlow struct {
	appCtx  *appcontext.AppContext
	noSpawn bool
	baseURL string
}

func NewLoginFlow(appCtx *appcontext.AppContext, noSpawn bool) (*loginFlow, error) {
	flow := &loginFlow{
		appCtx:  appCtx,
		noSpawn: noSpawn,
		baseURL: defaultBaseURL,
	}
	return flow, nil
}

func (flow *loginFlow) Poll(pollID string, iterations int, delay time.Duration, progressCb func()) (string, error) {
	return flow.poll(pollID, iterations, delay, progressCb, nil)
}

func (flow *loginFlow) poll(pollID string, iterations int, delay time.Duration, progressCb func(), promptCode func(retry bool) (string, error)) (string, error) {
	var completionCode string
	prompted := false

	tick := time.After(0)
	for range iterations {
		select {
		case <-flow.appCtx.Done():
			return "", flow.appCtx.Err()
		case <-tick:
			token, status, err := flow.PollOnce(pollID, completionCode)
			if err != nil {
				return "", err
			}
			switch status {
			case PollDone:
				return token, nil
			case PollPending:
				progressCb()
			case PollCodeRequired:
				if promptCode == nil {
					return "", fmt.Errorf("unexpected status code: %d", http.StatusUnauthorized)
				}
				completionCode, err = promptCode(prompted)
				if err != nil {
					return "", err
				}
				prompted = true
			}
		}
		tick = time.After(delay)
	}
	return "", fmt.Errorf("could not obtain token after %d iterations", iterations)
}

// PollStatus is where a sign-in stands after one poll of the auth api.
type PollStatus int

const (
	// PollPending: the user has not completed the sign-in yet.
	PollPending PollStatus = iota
	// PollCodeRequired: the sign-in is complete but the token is only
	// released against the completion code, and none or a wrong one was sent.
	PollCodeRequired
	// PollDone: the token was released.
	PollDone
)

// PollOnce asks the auth api once whether the sign-in identified by pollID
// has completed, presenting completionCode when it is not empty. The token is
// only returned with PollDone.
func (flow *loginFlow) PollOnce(pollID, completionCode string) (string, PollStatus, error) {
	reqUrl := flow.baseURL + "/v1/auth/poll/" + url.PathEscape(pollID)
	req, err := http.NewRequestWithContext(flow.appCtx, "POST", reqUrl, nil)
	if err != nil {
		return "", PollPending, fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
	}
	if completionCode != "" {
		req.Header.Set("X-Completion-Code", completionCode)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", PollPending, fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		var tokenResponse TokenResponse
		if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
			return "", PollPending, fmt.Errorf("failed to decode response JSON: %v", err)
		}
		return tokenResponse.Token, PollDone, nil
	case http.StatusAccepted:
		return "", PollPending, nil
	case http.StatusNotFound:
		return "", PollPending, ErrUnknownPollID
	case http.StatusTooManyRequests:
		return "", PollPending, ErrRateLimited
	case http.StatusUnauthorized:
		if isCompletionCodeRequired(resp.Body) {
			return "", PollCodeRequired, nil
		}
		return "", PollPending, fmt.Errorf("unexpected status code: %d", http.StatusUnauthorized)
	default:
		return "", PollPending, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

// maxErrorBodySize bounds how much of an error response body is read;
// the expected content is a small JSON error envelope.
const maxErrorBodySize = 4096

func isCompletionCodeRequired(body io.Reader) bool {
	data, _ := io.ReadAll(io.LimitReader(body, maxErrorBodySize))
	var env struct{ Code string }
	if json.Unmarshal(data, &env) == nil && env.Code != "" {
		return env.Code == "completion_code_required"
	}
	return bytes.Contains(data, []byte("invalid completion code"))
}

func (flow *loginFlow) Run(provider string, parameters map[string]string) (string, error) {
	var url string
	var body io.Reader

	switch provider {
	case "github":
		url = flow.baseURL + "/v1/auth/login/github"
	case "email":
		url = flow.baseURL + "/v1/auth/login/email"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	// parameters is map[string]string; completion_code is a JSON bool, so the
	// request body is an any-valued copy with the flag added.
	payload := map[string]any{"completion_code": true}
	for k, v := range parameters {
		payload[k] = v
	}
	if bodyBytes, err := json.Marshal(payload); err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %v", err)
	} else {
		body = bytes.NewBuffer(bodyBytes)
	}

	resp, err := http.Post(url, "application/json", body)
	if err != nil {
		return "", fmt.Errorf("unable to get the login URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return "", ErrRateLimited
		}
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	switch provider {
	case "github":
		return flow.handleGithubResponse(resp)
	case "email":
		return flow.handleEmailResponse(resp)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (flow *loginFlow) handleGithubResponse(resp *http.Response) (string, error) {
	var respData struct {
		URL            string `json:"URL"`
		PollID         string `json:"poll_id"`
		CompletionCode bool   `json:"completion_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	if flow.noSpawn {
		fmt.Printf("\nPlease open the following URL in your browser:\n\n")
		fmt.Printf("  %s\n\n", respData.URL)
	} else if err := utils.BrowserTrySpawn(respData.URL); err != nil {
		fmt.Printf("\nFailed to launch browser: %s\n\n", err)
		fmt.Printf("\nPlease open the following URL in your browser:\n\n")
		fmt.Printf("  %s\n\n", respData.URL)
	}

	// Wait for the token to be received
	fmt.Printf("Waiting for your browser to complete the login")
	defer fmt.Printf("\n")
	return flow.poll(respData.PollID, 300, time.Second, func() { fmt.Printf(".") },
		flow.completionCodePrompt(respData.CompletionCode))
}

func (flow *loginFlow) handleEmailResponse(resp *http.Response) (string, error) {
	var respData struct {
		PollID         string `json:"poll_id"`
		CompletionCode bool   `json:"completion_code"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	fmt.Printf("\nCheck your email for the login link. Do not close this window until you have logged in.\n")
	defer fmt.Printf("\n")
	return flow.poll(respData.PollID, 300, time.Second, func() { fmt.Printf(".") },
		flow.completionCodePrompt(respData.CompletionCode))
}

func (flow *loginFlow) completionCodePrompt(enabled bool) func(retry bool) (string, error) {
	if !enabled {
		return nil
	}
	reader := bufio.NewReader(flow.appCtx.Stdin)
	return func(retry bool) (string, error) {
		if retry {
			fmt.Printf("That code didn't match, try again: ")
		} else {
			fmt.Printf("\nEnter the code shown in your browser to finish signing in: ")
		}
		line, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read completion code: %w", err)
		}
		return strings.TrimSpace(line), nil
	}
}

// UILogin is a sign-in started on behalf of the local ui. The ui opens URL
// (github only; the email provider sends a link instead) and then drives
// PollOnce with PollID, passing the completion code the user reads off the
// sign-in page.
type UILogin struct {
	URL    string `json:"URL"`
	PollID string `json:"poll_id"`
}

// RunUI starts a sign-in for the local ui. The auth api releases the token
// only against the completion code shown to whoever completed the sign-in,
// and refuses a redirect alongside it: the user comes back to the ui tab by
// hand to type the code.
func (flow *loginFlow) RunUI(provider string, parameters map[string]string) (*UILogin, error) {
	var url string

	switch provider {
	case "github":
		url = flow.baseURL + "/v1/auth/login/github"
	case "email":
		url = flow.baseURL + "/v1/auth/login/email"
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	payload := map[string]any{"completion_code": true}
	for k, v := range parameters {
		payload[k] = v
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %v", err)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("unable to get the login URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusTooManyRequests {
			return nil, ErrRateLimited
		}
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var respData UILogin
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return nil, fmt.Errorf("failed to decode response JSON: %v", err)
	}
	if respData.PollID == "" {
		return nil, fmt.Errorf("the login API returned no poll ID")
	}
	return &respData, nil
}

func (flow *loginFlow) Close() error {
	return nil
}

func DeriveToken(ctx *appcontext.AppContext) (string, error) {
	token, err := ctx.GetCookies().GetAuthToken()
	if err != nil {
		return "", err
	}

	base := os.Getenv("PLAKAR_API_URL")
	if base == "" {
		base = defaultBaseURL
	}
	url := base + "/v1/account/derive-token"
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", fmt.Sprintf("plakar/%s (%s/%s)", utils.VERSION, runtime.GOOS, runtime.GOARCH))
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("request failed with status %s", res.Status)
	}

	var tokenResponse TokenResponse
	if err := json.NewDecoder(res.Body).Decode(&tokenResponse); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	return tokenResponse.Token, nil
}

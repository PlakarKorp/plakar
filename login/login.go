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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"time"

	"github.com/PlakarKorp/plakar/appcontext"
	"github.com/PlakarKorp/plakar/utils"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type loginFlow struct {
	appCtx  *appcontext.AppContext
	noSpawn bool
}

func NewLoginFlow(appCtx *appcontext.AppContext, noSpawn bool) (*loginFlow, error) {
	flow := &loginFlow{
		appCtx:  appCtx,
		noSpawn: noSpawn,
	}
	return flow, nil
}

func (flow *loginFlow) Poll(pollID string, iterations int, delay time.Duration, progressCb func()) (string, error) {
	tick := time.After(0)
	for range iterations {
		select {
		case <-flow.appCtx.Done():
			return "", flow.appCtx.Err()
		case <-tick:
			reqUrl := "https://api.plakar.io/v1/auth/poll/" + pollID
			req, err := http.NewRequestWithContext(flow.appCtx, "POST", reqUrl, nil)
			if err != nil {
				return "", fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
			}

			client := http.DefaultClient
			resp, err := client.Do(req)
			if err != nil {
				return "", fmt.Errorf("the /auth/login/github/poll API endpoint failed: %w", err)
			}
			// leaking resp for now
			if resp.StatusCode == http.StatusOK {
				var tokenResponse TokenResponse
				if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
					return "", fmt.Errorf("failed to decode response JSON: %v", err)
				}
				return tokenResponse.Token, nil
			} else if resp.StatusCode == http.StatusNotFound {
				return "", fmt.Errorf("unknown ID")
			} else if resp.StatusCode == http.StatusAccepted {
				progressCb()
			} else {
				return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
			}
		}
		tick = time.After(delay)
	}
	return "", fmt.Errorf("could not obtain token after %d iterations", iterations)
}

func (flow *loginFlow) Run(provider string, parameters map[string]string) (string, error) {
	var url string
	var body io.Reader

	switch provider {
	case "github":
		url = "https://api.plakar.io/v1/auth/login/github"
	case "email":
		url = "https://api.plakar.io/v1/auth/login/email"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if bodyBytes, err := json.Marshal(parameters); err != nil {
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
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d : %s", resp.StatusCode, data)
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
		URL    string `json:"URL"`
		PollID string `json:"poll_id"`
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
	return flow.Poll(respData.PollID, 300, time.Second, func() { fmt.Printf(".") })
}

func (flow *loginFlow) handleEmailResponse(resp *http.Response) (string, error) {
	var respData struct {
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	fmt.Printf("\nCheck your email for the login link. Do not close this window until you have logged in.\n")
	defer fmt.Printf("\n")
	return flow.Poll(respData.PollID, 300, time.Second, func() { fmt.Printf(".") })
}

func (flow *loginFlow) RunUI(provider string, parameters map[string]string) (string, error) {
	var url string
	var body io.Reader

	switch provider {
	case "github":
		url = "https://api.plakar.io/v1/auth/login/github"
	case "email":
		url = "https://api.plakar.io/v1/auth/login/email"
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}

	if bodyBytes, err := json.Marshal(parameters); err != nil {
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
		data, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("unexpected status code: %d : %s", resp.StatusCode, data)
	}

	switch provider {
	case "github":
		return flow.handleGithubResponseUI(resp)
	case "email":
		return flow.handleEmailResponseUI(resp)
	default:
		return "", fmt.Errorf("unsupported provider: %s", provider)
	}
}

func (flow *loginFlow) handleGithubResponseUI(resp *http.Response) (string, error) {
	var respData struct {
		URL    string `json:"URL"`
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	go func() {
		token, _ := flow.Poll(respData.PollID, 10, time.Second*5, func() {})
		if token != "" {
			if err := flow.appCtx.GetCookies().PutAuthToken(token); err != nil {
			}
		}

	}()

	return respData.URL, nil
}

func (flow *loginFlow) handleEmailResponseUI(resp *http.Response) (string, error) {
	var respData struct {
		PollID string `json:"poll_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return "", fmt.Errorf("failed to decode response JSON: %v", err)
	}

	go func() {
		token, _ := flow.Poll(respData.PollID, 10, time.Second*5, func() {})
		if token != "" {
			if err := flow.appCtx.GetCookies().PutAuthToken(token); err != nil {
			}
		}
	}()

	return "", nil
}

func (flow *loginFlow) Close() error {
	return nil
}

func DeriveToken(ctx *appcontext.AppContext) (string, error) {
	token, err := ctx.GetCookies().GetAuthToken()
	if err != nil {
		return "", err
	}

	url := "https://api.plakar.io/v1/account/derive-token"
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

package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// launchCodeTTL bounds how long the browser has to redeem the launch code
// once plakar ui started it.
const launchCodeTTL = 2 * time.Minute

// launchCode is the one-time code plakar ui hands to the browser it spawns,
// to be redeemed for the token: the token itself never goes through the URL,
// where it would end up in the browser history and in the arguments of the
// process that opens it.
type launchCode struct {
	mu      sync.Mutex
	code    string
	expires time.Time
}

func newLaunchCode(code string) *launchCode {
	return &launchCode{code: code, expires: time.Now().Add(launchCodeTTL)}
}

// redeem reports whether code is the launch code, still valid, and spends it.
func (l *launchCode) redeem(code string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.code == "" || time.Now().After(l.expires) {
		l.code = ""
		return false
	}
	if subtle.ConstantTimeCompare([]byte(l.code), []byte(code)) == 0 {
		return false
	}
	l.code = ""
	return true
}

// LaunchExchangeRequest is the body of the exchange route.
type LaunchExchangeRequest struct {
	Code string `json:"code"`
}

// authenticationExchange trades the one-time launch code for the token. It
// is the only route that does not require the token.
func (ui *uiserver) authenticationExchange(w http.ResponseWriter, r *http.Request) error {
	var req LaunchExchangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096)).Decode(&req); err != nil {
		return badRequestBody(err)
	}
	if req.Code == "" {
		return parameterError("code", MissingArgument, ErrMissingField)
	}
	if !ui.launchCode.redeem(req.Code) {
		return authError("invalid or expired code")
	}
	return json.NewEncoder(w).Encode(&TokenResponse{Token: ui.token})
}

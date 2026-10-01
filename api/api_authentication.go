package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/PlakarKorp/plakar/cookies"
	"github.com/PlakarKorp/plakar/login"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
)

type TokenResponse struct {
	Token string `json:"token"`
}

type LoginRequestEmail struct {
	Email string `json:"email"`
}

type LoginPollRequest struct {
	PollID         string `json:"poll_id"`
	CompletionCode string `json:"completion_code"`
}

// LoginPollResponse never carries the token: it stays in the cookies of the
// plakar process serving the ui.
type LoginPollResponse struct {
	Status string `json:"status"`
}

func badRequestBody(err error) *ApiError {
	return &ApiError{
		HttpCode: http.StatusBadRequest,
		ErrCode:  "bad-request",
		Message:  "failed to decode request body: " + err.Error(),
	}
}

func (ui *uiserver) servicesLoginGithub(w http.ResponseWriter, r *http.Request) error {
	lf, err := login.NewLoginFlow(ui.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	res, err := lf.RunUI("github", map[string]string{})
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	return json.NewEncoder(w).Encode(res)
}

func (ui *uiserver) servicesLoginEmail(w http.ResponseWriter, r *http.Request) error {
	var req LoginRequestEmail

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return badRequestBody(err)
	}
	if req.Email == "" {
		return parameterError("email", MissingArgument, ErrMissingField)
	}
	if _, err := utils.ValidateEmail(req.Email); err != nil {
		return parameterError("email", InvalidArgument, err)
	}

	lf, err := login.NewLoginFlow(ui.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	res, err := lf.RunUI("email", map[string]string{"email": req.Email})
	if err != nil {
		return fmt.Errorf("failed to run login flow: %w", err)
	}

	return json.NewEncoder(w).Encode(res)
}

// servicesLoginPoll polls the auth api once for a sign-in started by
// servicesLoginGithub or servicesLoginEmail, and stores the token when it is
// released. The ui calls it repeatedly: "pending" until the user completes the
// sign-in, then "code-required" until it sends the code shown on the sign-in
// page ("invalid-code" if it sent a wrong one), then "ok".
func (ui *uiserver) servicesLoginPoll(w http.ResponseWriter, r *http.Request) error {
	var req LoginPollRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return badRequestBody(err)
	}
	if req.PollID == "" {
		return parameterError("poll_id", MissingArgument, ErrMissingField)
	}
	if uuid.Validate(req.PollID) != nil {
		return parameterError("poll_id", InvalidArgument, ErrInvalidID)
	}

	lf, err := login.NewLoginFlow(ui.ctx, true)
	if err != nil {
		return fmt.Errorf("failed to create login flow: %w", err)
	}

	token, status, err := lf.PollOnce(req.PollID, req.CompletionCode)
	if errors.Is(err, login.ErrUnknownPollID) {
		return &ApiError{
			HttpCode: http.StatusNotFound,
			ErrCode:  "not-found",
			Message:  "Unknown or expired sign-in, please start over.",
		}
	}
	if err != nil {
		return fmt.Errorf("failed to poll login: %w", err)
	}

	res := LoginPollResponse{}
	switch status {
	case login.PollPending:
		res.Status = "pending"
	case login.PollCodeRequired:
		res.Status = "code-required"
		if req.CompletionCode != "" {
			res.Status = "invalid-code"
		}
	case login.PollDone:
		if err := ui.ctx.GetCookies().PutAuthToken(token); err != nil {
			return fmt.Errorf("failed to store auth token: %w", err)
		}
		res.Status = "ok"
	}

	return json.NewEncoder(w).Encode(res)
}

func (ui *uiserver) servicesLogout(w http.ResponseWriter, r *http.Request) error {
	err := ui.ctx.GetCookies().DeleteAuthToken()
	if errors.Is(err, cookies.ErrNotLoggedIn) {
		return nil
	}
	return err
}

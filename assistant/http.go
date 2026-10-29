/*
 * Copyright (c) 2026 Gilles Chehade <gilles@poolp.org>
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

package assistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/PlakarKorp/plakar/api"
)

// maxMessageBytes bounds a user message; the request body a little more.
const maxMessageBytes = 32 << 10

// maxBodyBytes allows the message, the page and the JSON around them.
const maxBodyBytes = maxMessageBytes + 6*maxPageField + 1024

// Status is what GET /api/assistant reports. Local says whether messages
// stay on this machine; Provider is the name to show for where they go.
type Status struct {
	Enabled  bool     `json:"enabled"`
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Local    bool     `json:"local,omitempty"`
	Tools    []string `json:"tools,omitempty"`
}

// SetupRoutes registers the assistant endpoints on mux behind auth. With a
// nil assistant the status endpoint says so and the others are not served.
func SetupRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler, a *Assistant) {
	mux.Handle("GET /api/assistant", auth(api.JSONAPIView(func(w http.ResponseWriter, r *http.Request) error {
		status := Status{}
		if a != nil {
			status = Status{
				Enabled:  true,
				Provider: a.opts.label(),
				Model:    a.opts.Model,
				Local:    a.opts.local(),
				Tools:    a.names,
			}
		}
		return json.NewEncoder(w).Encode(api.Item[Status]{Item: status})
	})))

	if a == nil {
		return
	}

	mux.Handle("POST /api/assistant/messages", auth(api.APIView(a.postMessage)))
	mux.Handle("DELETE /api/assistant/conversations/{id}", auth(api.JSONAPIView(a.deleteConversation)))
}

type messageRequest struct {
	ConversationID string `json:"conversation_id"`
	Message        string `json:"message"`
	// Page is what the UI shows as the message is sent, if anything.
	Page *Page `json:"page,omitempty"`
}

// ConversationEvent opens a response with the conversation the answer
// belongs to, so a first message learns the ID to continue with.
type ConversationEvent struct {
	ID string `json:"id"`
}

// ErrorEvent ends a response that failed after it started streaming.
type ErrorEvent struct {
	Message string `json:"message"`
}

// postMessage answers one message as a stream of server-sent events: a
// conversation event first, then text, tool_use and tool_result events as
// the turn unfolds, and done or error last.
func (a *Assistant) postMessage(w http.ResponseWriter, r *http.Request) error {
	var req messageRequest
	body := http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(body).Decode(&req); err != nil {
		return &api.ApiError{HttpCode: http.StatusBadRequest, ErrCode: "bad-request", Message: fmt.Sprintf("invalid request body: %s", err)}
	}

	req.Message = strings.TrimSpace(req.Message)
	if req.Message == "" {
		return parameterError("message", api.MissingArgument, "message is required")
	}
	if len(req.Message) > maxMessageBytes {
		return parameterError("message", api.InvalidArgument, fmt.Sprintf("message exceeds %d bytes", maxMessageBytes))
	}
	if req.Page != nil {
		if err := req.Page.validate(); err != nil {
			return parameterError("page", api.InvalidArgument, err.Error())
		}
	}

	conv, err := a.acquire(req.ConversationID)
	switch {
	case errors.Is(err, ErrUnknownConversation):
		return &api.ApiError{HttpCode: http.StatusNotFound, ErrCode: "not-found", Message: err.Error()}
	case errors.Is(err, ErrBusy):
		return &api.ApiError{HttpCode: http.StatusConflict, ErrCode: "busy", Message: err.Error()}
	case errors.Is(err, ErrTooMany):
		return &api.ApiError{HttpCode: http.StatusServiceUnavailable, ErrCode: "busy", Message: err.Error()}
	case err != nil:
		return err
	}
	defer a.release(conv)

	if len(conv.messages) >= maxTurns {
		return &api.ApiError{HttpCode: http.StatusConflict, ErrCode: "too-long", Message: ErrTooLong.Error()}
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		return fmt.Errorf("streaming unsupported by this server")
	}

	// From here on the status line is sent, and failures travel as events.
	h := w.Header()
	h.Set("Content-Type", "text/event-stream; charset=utf-8")
	h.Set("Cache-Control", "no-store")
	h.Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	emit := func(name string, payload any) error {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data); err != nil {
			return err
		}
		flusher.Flush()
		return nil
	}

	if err := emit("conversation", ConversationEvent{ID: conv.id}); err != nil {
		return nil
	}
	// The page rides along in the user turn, after the message, so that it
	// dates with it: a later message from another page tells the new one.
	text := req.Message
	if note := req.Page.note(); note != "" {
		text += "\n\n" + note
	}
	if err := a.run(r.Context(), conv, text, emit); err != nil {
		if r.Context().Err() != nil {
			// The browser left: nobody to tell.
			return nil
		}
		a.logf("assistant: conversation %s: %s", conv.id, err)
		_ = emit("error", ErrorEvent{Message: err.Error()})
	}
	return nil
}

func (a *Assistant) deleteConversation(w http.ResponseWriter, r *http.Request) error {
	if !a.Forget(r.PathValue("id")) {
		return &api.ApiError{HttpCode: http.StatusNotFound, ErrCode: "not-found", Message: ErrUnknownConversation.Error()}
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func parameterError(field string, code api.ParamErrorType, message string) *api.ApiError {
	return &api.ApiError{
		HttpCode: http.StatusBadRequest,
		ErrCode:  "invalid_params",
		Message:  "Invalid parameter",
		Params:   map[string]api.ParameterError{field: {Code: code, Message: message}},
	}
}

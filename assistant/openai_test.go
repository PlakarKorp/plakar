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
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/PlakarKorp/plakar/api"
	plakarmcp "github.com/PlakarKorp/plakar/subcommands/mcp"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// fakeChat stands in for a chat completions endpoint, the way LM Studio,
// Ollama, OpenAI or Gemini serve one.
type fakeChat struct {
	mu       sync.Mutex
	requests []map[string]any
	headers  []http.Header
	scripts  []func(w http.ResponseWriter)
}

func (f *fakeChat) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/v1/chat/completions" {
		http.Error(w, "unexpected path "+r.URL.Path, http.StatusNotFound)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, req)
	f.headers = append(f.headers, r.Header.Clone())
	n := len(f.requests)
	f.mu.Unlock()

	if n > len(f.scripts) {
		http.Error(w, "no script for this request", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	f.scripts[n-1](w)
}

func chunk(w http.ResponseWriter, delta, finish string) {
	reason := "null"
	if finish != "" {
		reason = fmt.Sprintf("%q", finish)
	}
	fmt.Fprintf(w, "data: {\"id\":\"c1\",\"object\":\"chat.completion.chunk\",\"choices\":[{\"index\":0,\"delta\":%s,\"finish_reason\":%s}]}\n\n", delta, reason)
}

// chatToolScript streams a tool call the way OpenAI fragments it: the id and
// name first, then the arguments in pieces, with no finish reason until the
// end.
func chatToolScript(name string, argumentPieces ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		chunk(w, `{"role":"assistant","content":null}`, "")
		chunk(w, fmt.Sprintf(`{"tool_calls":[{"index":0,"id":"call_abc","type":"function","function":{"name":%q,"arguments":""}}]}`, name), "")
		for _, piece := range argumentPieces {
			chunk(w, fmt.Sprintf(`{"tool_calls":[{"index":0,"function":{"arguments":%q}}]}`, piece), "")
		}
		chunk(w, `{}`, "tool_calls")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}
}

func chatTextScript(parts ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		chunk(w, `{"role":"assistant","content":""}`, "")
		for _, part := range parts {
			chunk(w, fmt.Sprintf(`{"content":%q}`, part), "")
		}
		chunk(w, `{}`, "stop")
		fmt.Fprint(w, "data: [DONE]\n\n")
	}
}

func newChatServer(t *testing.T, fake *fakeChat, opts Options) (*http.ServeMux, string) {
	t.Helper()
	repo, ctx := ptesting.GenerateRepository(t, bytes.NewBuffer(nil), bytes.NewBuffer(nil), nil)
	t.Cleanup(func() { ctx.Close() })

	snap := ptesting.GenerateSnapshot(t, repo, []ptesting.MockFile{
		ptesting.NewMockDir("subdir"),
		ptesting.NewMockFile("subdir/dummy.txt", 0644, "hello dummy"),
	})
	snapID := hex.EncodeToString(snap.Header.GetIndexShortID())
	snap.Close()

	upstream := httptest.NewServer(fake)
	t.Cleanup(upstream.Close)
	opts.BaseURL = upstream.URL + "/v1"

	server := (&plakarmcp.Mcp{MaxFileSize: MaxFileSize}).NewServer(ctx, repo)
	a, err := New(ctx, server, opts)
	require.NoError(t, err)
	t.Cleanup(func() { a.Close() })

	mux := http.NewServeMux()
	SetupRoutes(mux, api.TokenAuthMiddleware("secret"), a)
	return mux, snapID
}

func TestChatToolRoundTrip(t *testing.T) {
	fake := &fakeChat{scripts: []func(http.ResponseWriter){
		chatToolScript("list_files", `{"snap`, `shot":"`, "SNAP", `","path":"/subdir"}`),
		chatTextScript("One file: ", "dummy.txt."),
	}}
	mux, snapID := newChatServer(t, fake, Options{Provider: ProviderLMStudio, Model: "qwen"})

	// The script does not know the snapshot ID; patch it into the pieces.
	fake.scripts[0] = chatToolScript("list_files", `{"snapshot":"`+snapID+`","path":"/subdir"}`)

	_, events := ask(t, mux, "", "what is in subdir?")
	require.Equal(t, []string{"conversation", "tool_use", "tool_result", "text", "text", "done"}, names(events))
	require.Equal(t, "list_files", events[1].data["name"])
	require.Equal(t, "call_abc", events[1].data["id"])
	require.Contains(t, events[2].data["output"], "dummy.txt")
	require.Equal(t, false, events[2].data["is_error"])
	require.Equal(t, "end_turn", events[5].data["stop_reason"])

	// Local servers get no credentials.
	require.Empty(t, fake.headers[0].Get("Authorization"))

	// The tools went out as functions, the schema untouched.
	tools := fake.requests[0]["tools"].([]any)
	var found bool
	for _, tool := range tools {
		function := tool.(map[string]any)["function"].(map[string]any)
		parameters := function["parameters"].(map[string]any)
		require.Equal(t, "object", parameters["type"])
		// Every tool carries properties, empty for the ones taking no input.
		require.Contains(t, parameters, "properties")
		if function["name"] == "list_files" {
			found = true
			require.Contains(t, parameters["properties"], "snapshot")
		}
	}
	require.True(t, found)

	// The second request replays the call and its result in the chat form.
	messages := fake.requests[1]["messages"].([]any)
	require.Len(t, messages, 4)
	require.Equal(t, "system", messages[0].(map[string]any)["role"])
	assistant := messages[2].(map[string]any)
	require.Equal(t, "assistant", assistant["role"])
	calls := assistant["tool_calls"].([]any)
	require.Len(t, calls, 1)
	call := calls[0].(map[string]any)
	require.Equal(t, "call_abc", call["id"])
	arguments := call["function"].(map[string]any)["arguments"].(string)
	require.Contains(t, arguments, snapID)
	result := messages[3].(map[string]any)
	require.Equal(t, "tool", result["role"])
	require.Equal(t, "call_abc", result["tool_call_id"])
	require.Contains(t, result["content"], "dummy.txt")
}

func TestChatCredentialsAndModel(t *testing.T) {
	fake := &fakeChat{scripts: []func(http.ResponseWriter){chatTextScript("hi")}}
	mux, _ := newChatServer(t, fake, Options{Provider: ProviderOpenAI, Model: "gpt-test", APIKey: "sk-test"})

	_, events := ask(t, mux, "", "hello")
	require.Equal(t, "done", events[len(events)-1].name)
	require.Equal(t, "Bearer sk-test", fake.headers[0].Get("Authorization"))
	require.Equal(t, "gpt-test", fake.requests[0]["model"])
	require.Equal(t, true, fake.requests[0]["stream"])
}

func TestChatFailureIsReported(t *testing.T) {
	fake := &fakeChat{scripts: []func(http.ResponseWriter){
		func(w http.ResponseWriter) {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"error":{"message":"Incorrect API key provided","type":"invalid_request_error"}}`)
		},
		chatTextScript("ok"),
	}}
	mux, _ := newChatServer(t, fake, Options{Provider: ProviderOpenAI, Model: "gpt-test", APIKey: "bad"})

	id, events := ask(t, mux, "", "hello")
	last := events[len(events)-1]
	require.Equal(t, "error", last.name)
	require.Contains(t, last.data["message"], "refused the credentials")
	require.Contains(t, last.data["message"], "Incorrect API key")

	// The failed turn left nothing behind.
	_, events = ask(t, mux, id, "again")
	require.Equal(t, "done", events[len(events)-1].name)
	require.Len(t, fake.requests[1]["messages"].([]any), 2)
}

func TestChatStatus(t *testing.T) {
	mux, _ := newChatServer(t, &fakeChat{}, Options{Provider: ProviderOllama, Model: "llama"})

	req, _ := http.NewRequest("GET", "/api/assistant", nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var res api.Item[Status]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.True(t, res.Item.Enabled)
	require.Equal(t, "Ollama", res.Item.Provider)
	require.Equal(t, "llama", res.Item.Model)
	require.True(t, res.Item.Local)
}

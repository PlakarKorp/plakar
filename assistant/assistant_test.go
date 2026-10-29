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
	"bufio"
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/PlakarKorp/plakar/api"
	plakarmcp "github.com/PlakarKorp/plakar/subcommands/mcp"
	ptesting "github.com/PlakarKorp/plakar/testing"
	"github.com/stretchr/testify/require"
)

// fakeClaude stands in for the Messages API: each request is answered with
// the next scripted stream, and every request body is kept for inspection.
type fakeClaude struct {
	mu       sync.Mutex
	requests []map[string]any
	scripts  []func(w http.ResponseWriter)
}

func (f *fakeClaude) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var req map[string]any
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	f.mu.Lock()
	f.requests = append(f.requests, req)
	n := len(f.requests)
	f.mu.Unlock()

	if n > len(f.scripts) {
		http.Error(w, "no script for this request", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	f.scripts[n-1](w)
}

// messagesOf returns the messages the n-th request (from 1) carried.
func (f *fakeClaude) messagesOf(t *testing.T, n int) []any {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	require.GreaterOrEqual(t, len(f.requests), n)
	messages, ok := f.requests[n-1]["messages"].([]any)
	require.True(t, ok)
	return messages
}

func sse(w http.ResponseWriter, name, data string) {
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", name, data)
}

func streamStart(w http.ResponseWriter) {
	sse(w, "message_start", `{"type":"message_start","message":{"id":"msg_1","type":"message","role":"assistant","model":"claude-test","content":[],"stop_reason":null,"stop_sequence":null,"usage":{"input_tokens":1,"output_tokens":1}}}`)
}

func streamEnd(w http.ResponseWriter, stopReason string) {
	sse(w, "message_delta", fmt.Sprintf(`{"type":"message_delta","delta":{"stop_reason":%q,"stop_sequence":null},"usage":{"output_tokens":1}}`, stopReason))
	sse(w, "message_stop", `{"type":"message_stop"}`)
}

// toolCallScript streams one tool_use block.
func toolCallScript(name, input string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		streamStart(w)
		sse(w, "content_block_start", fmt.Sprintf(`{"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_1","name":%q,"input":{}}}`, name))
		sse(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":%q}}`, input))
		sse(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		streamEnd(w, "tool_use")
	}
}

// textScript streams an answer in as many pieces as given.
func textScript(parts ...string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		streamStart(w)
		sse(w, "content_block_start", `{"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}`)
		for _, part := range parts {
			sse(w, "content_block_delta", fmt.Sprintf(`{"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":%q}}`, part))
		}
		sse(w, "content_block_stop", `{"type":"content_block_stop","index":0}`)
		streamEnd(w, "end_turn")
	}
}

// failScript answers with an API error.
func failScript(status int, body string) func(http.ResponseWriter) {
	return func(w http.ResponseWriter) {
		w.WriteHeader(status)
		fmt.Fprint(w, body)
	}
}

type event struct {
	name string
	data map[string]any
}

func readEvents(t *testing.T, body io.Reader) []event {
	t.Helper()
	var events []event
	var current event
	scanner := bufio.NewScanner(body)
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case strings.HasPrefix(line, "event: "):
			current.name = strings.TrimPrefix(line, "event: ")
		case strings.HasPrefix(line, "data: "):
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &current.data))
		case line == "":
			if current.name != "" {
				events = append(events, current)
			}
			current = event{}
		}
	}
	require.NoError(t, scanner.Err())
	return events
}

func names(events []event) []string {
	out := make([]string, 0, len(events))
	for _, e := range events {
		out = append(out, e.name)
	}
	return out
}

func newTestServer(t *testing.T, fake *fakeClaude) (*http.ServeMux, string) {
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

	server := (&plakarmcp.Mcp{MaxFileSize: MaxFileSize}).NewServer(ctx, repo)
	a, err := New(ctx, server, Options{Provider: ProviderAnthropic, APIKey: "test", BaseURL: upstream.URL})
	require.NoError(t, err)
	t.Cleanup(func() { a.Close() })

	mux := http.NewServeMux()
	SetupRoutes(mux, api.TokenAuthMiddleware("secret"), a)
	return mux, snapID
}

func post(t *testing.T, mux *http.ServeMux, token string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req, err := http.NewRequest("POST", "/api/assistant/messages", strings.NewReader(body))
	require.NoError(t, err)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// ask sends message in the conversation (a new one when empty), from the
// page if given, and returns the conversation ID and the events of the
// answer.
func ask(t *testing.T, mux *http.ServeMux, conversation, message string, page ...Page) (string, []event) {
	t.Helper()
	req := map[string]any{"conversation_id": conversation, "message": message}
	if len(page) > 0 {
		req["page"] = page[0]
	}
	body, err := json.Marshal(req)
	require.NoError(t, err)
	w := post(t, mux, "secret", string(body))
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	events := readEvents(t, w.Body)
	require.NotEmpty(t, events)
	require.Equal(t, "conversation", events[0].name)
	id, ok := events[0].data["id"].(string)
	require.True(t, ok)
	return id, events
}

func TestStatus(t *testing.T) {
	mux, _ := newTestServer(t, &fakeClaude{})

	req, _ := http.NewRequest("GET", "/api/assistant", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)

	req.Header.Set("Authorization", "Bearer secret")
	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())

	var res api.Item[Status]
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &res))
	require.True(t, res.Item.Enabled)
	require.Equal(t, DefaultModel, res.Item.Model)
	require.Contains(t, res.Item.Tools, "list_snapshots")
	require.NotContains(t, res.Item.Tools, "backup")
}

func TestStatusDisabled(t *testing.T) {
	mux := http.NewServeMux()
	SetupRoutes(mux, api.TokenAuthMiddleware(""), nil)

	req, _ := http.NewRequest("GET", "/api/assistant", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
	require.JSONEq(t, `{"item":{"enabled":false}}`, w.Body.String())

	w = post(t, mux, "", `{"message":"hi"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestToolRoundTrip(t *testing.T) {
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){
		toolCallScript("list_snapshots", `{}`),
		textScript("There is ", "one snapshot."),
		textScript("Still one."),
	}}
	mux, snapID := newTestServer(t, fake)

	w := post(t, mux, "secret", `{"message":"how many snapshots?"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
	require.Equal(t, "text/event-stream; charset=utf-8", w.Header().Get("Content-Type"))

	events := readEvents(t, w.Body)
	require.Equal(t, []string{"conversation", "tool_use", "tool_result", "text", "text", "done"}, names(events))

	require.Equal(t, "list_snapshots", events[1].data["name"])
	require.Equal(t, "list_snapshots", events[2].data["name"])
	require.Contains(t, events[2].data["output"], snapID)
	require.Equal(t, false, events[2].data["is_error"])
	require.Equal(t, "There is ", events[3].data["text"])
	require.Equal(t, "end_turn", events[5].data["stop_reason"])

	// The tool result went back to the model, in a user turn after the
	// assistant's tool call.
	second := fake.messagesOf(t, 2)
	require.Len(t, second, 3)
	last := second[2].(map[string]any)
	require.Equal(t, "user", last["role"])
	result := last["content"].([]any)[0].(map[string]any)
	require.Equal(t, "tool_result", result["type"])
	require.Equal(t, "toolu_1", result["tool_use_id"])
	require.Contains(t, result["content"].([]any)[0].(map[string]any)["text"], snapID)

	// The tool definitions came from the MCP server.
	tools := fake.requests[0]["tools"].([]any)
	var found bool
	for _, tool := range tools {
		if tool.(map[string]any)["name"] == "list_files" {
			found = true
			schema := tool.(map[string]any)["input_schema"].(map[string]any)
			require.Equal(t, "object", schema["type"])
			require.Contains(t, schema["properties"], "snapshot")
			require.Contains(t, schema["required"], "snapshot")
		}
	}
	require.True(t, found, "list_files tool not sent")

	// The next message in the conversation replays the whole turn.
	id := events[0].data["id"].(string)
	_, events = ask(t, mux, id, "still?")
	require.Equal(t, "done", events[len(events)-1].name)
	require.Len(t, fake.messagesOf(t, 3), 5)
}

func TestToolErrorIsFedBack(t *testing.T) {
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){
		toolCallScript("read_file", `{"snapshot":"nope","path":"/x"}`),
		textScript("That snapshot does not exist."),
	}}
	mux, _ := newTestServer(t, fake)

	_, events := ask(t, mux, "", "read /x from nope")
	require.Equal(t, "tool_result", events[2].name)
	require.Equal(t, true, events[2].data["is_error"])
	require.Equal(t, "done", events[len(events)-1].name)

	second := fake.messagesOf(t, 2)
	result := second[2].(map[string]any)["content"].([]any)[0].(map[string]any)
	require.Equal(t, true, result["is_error"])
}

func TestFailedTurnIsNotKept(t *testing.T) {
	unauthorized := `{"type":"error","error":{"type":"authentication_error","message":"invalid x-api-key"}}`
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){
		failScript(http.StatusUnauthorized, unauthorized),
		toolCallScript("list_snapshots", `{}`),
		// A 400 rather than a 500: the SDK retries the latter, and would
		// consume the next script doing so.
		failScript(http.StatusBadRequest, `{"type":"error","error":{"type":"invalid_request_error","message":"boom"}}`),
		textScript("hello"),
	}}
	mux, _ := newTestServer(t, fake)

	// A failure before any answer.
	id, events := ask(t, mux, "", "hello")
	last := events[len(events)-1]
	require.Equal(t, "error", last.name)
	require.Contains(t, last.data["message"], "ANTHROPIC_API_KEY")

	// A failure after a tool round, which has already produced a tool_use
	// and its result.
	_, events = ask(t, mux, id, "hello again")
	require.Equal(t, []string{"conversation", "tool_use", "tool_result", "error"}, names(events))

	// Neither turn is part of the history the next message replays: it
	// carries that message alone.
	_, events = ask(t, mux, id, "third time")
	require.Equal(t, "done", events[len(events)-1].name)
	require.Len(t, fake.messagesOf(t, 4), 1)
}

func TestRequestValidation(t *testing.T) {
	mux, _ := newTestServer(t, &fakeClaude{})

	w := post(t, mux, "secret", `{"message":"  "}`)
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "invalid_params")

	w = post(t, mux, "secret", `{"conversation_id":"missing","message":"hi"}`)
	require.Equal(t, http.StatusNotFound, w.Code)

	w = post(t, mux, "secret", `not json`)
	require.Equal(t, http.StatusBadRequest, w.Code)

	w = post(t, mux, "", `{"message":"hi"}`)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestForget(t *testing.T) {
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){textScript("hi")}}
	mux, _ := newTestServer(t, fake)

	id, _ := ask(t, mux, "", "hello")

	req, _ := http.NewRequest("DELETE", "/api/assistant/conversations/"+id, nil)
	req.Header.Set("Authorization", "Bearer secret")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusNoContent, w.Code)

	w = httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	require.Equal(t, http.StatusNotFound, w.Code)

	w = post(t, mux, "secret", `{"conversation_id":"`+id+`","message":"hi"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestEvictsIdleConversations(t *testing.T) {
	scripts := make([]func(http.ResponseWriter), 0, maxConversations+1)
	for i := 0; i < maxConversations+1; i++ {
		scripts = append(scripts, textScript("hi"))
	}
	mux, _ := newTestServer(t, &fakeClaude{scripts: scripts})

	var first string
	for i := 0; i < maxConversations+1; i++ {
		id, _ := ask(t, mux, "", "hello")
		if i == 0 {
			first = id
		}
	}

	w := post(t, mux, "secret", `{"conversation_id":"`+first+`","message":"still there?"}`)
	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestBusyConversation(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){
		textScript("hi"),
		func(w http.ResponseWriter) {
			close(started)
			<-release
			textScript("done waiting")(w)
		},
	}}
	mux, _ := newTestServer(t, fake)

	id, _ := ask(t, mux, "", "hello")

	answered := make(chan []event, 1)
	go func() {
		_, events := ask(t, mux, id, "slow one")
		answered <- events
	}()

	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("the slow answer never reached the model")
	}

	w := post(t, mux, "secret", `{"conversation_id":"`+id+`","message":"me too"}`)
	require.Equal(t, http.StatusConflict, w.Code, w.Body.String())
	require.Contains(t, w.Body.String(), "still answering")

	close(release)
	select {
	case events := <-answered:
		require.Equal(t, "done", events[len(events)-1].name)
	case <-time.After(5 * time.Second):
		t.Fatal("the slow answer never completed")
	}

	// Released, the conversation takes messages again; the fake has no
	// script left, so the answer is an error event, not a refusal.
	w = post(t, mux, "secret", `{"conversation_id":"`+id+`","message":"x"}`)
	require.Equal(t, http.StatusOK, w.Code, w.Body.String())
}

func TestPageNote(t *testing.T) {
	fake := &fakeClaude{scripts: []func(http.ResponseWriter){
		textScript("A directory."),
		textScript("Nothing to add."),
	}}
	mux, _ := newTestServer(t, fake)

	// The page rides along in the user turn, after the message.
	_, events := ask(t, mux, "", `what is here?`, Page{Name: "browse", Snapshot: "1a2b3c4d", Path: "/home"})
	require.Equal(t, "done", events[len(events)-1].name)
	first, err := json.Marshal(fake.messagesOf(t, 1)[0])
	require.NoError(t, err)
	require.Contains(t, string(first), `what is here?\n\n[The web UI is showing snapshot 1a2b3c4d, browsing the directory /home.]`)

	// A page without a subject adds nothing.
	_, events = ask(t, mux, "", `and now?`, Page{Name: "settings"})
	require.Equal(t, "done", events[len(events)-1].name)
	second, err := json.Marshal(fake.messagesOf(t, 2)[0])
	require.NoError(t, err)
	require.NotContains(t, string(second), "web UI")

	// An oversized page is refused before anything is sent.
	w := post(t, mux, "secret", fmt.Sprintf(`{"message":"hi","page":{"name":"browse","snapshot":"x","path":%q}}`, strings.Repeat("p", maxPageField+1)))
	require.Equal(t, http.StatusBadRequest, w.Code)
	require.Contains(t, w.Body.String(), "page")
	require.Len(t, fake.requests, 2)
}

func TestPageNotes(t *testing.T) {
	for _, c := range []struct {
		page Page
		want string
	}{
		{Page{Name: "browse", Snapshot: "ab", Path: "/", File: "/etc/hosts"}, "[The web UI is showing snapshot ab, previewing the file /etc/hosts.]"},
		{Page{Name: "browse"}, ""},
		{Page{Name: "errors", Snapshot: "ab", Path: "/"}, "[The web UI is showing the errors of snapshot ab.]"},
		{Page{Name: "errors", Snapshot: "ab", Path: "/var"}, "[The web UI is showing the errors of snapshot ab under /var.]"},
		{Page{Name: "compare", Older: "ab", Newer: "cd", Path: "/etc"}, "[The web UI is showing the comparison of snapshot ab (older) with snapshot cd (newer), at /etc.]"},
		{Page{Name: "compare"}, "[The web UI is showing the comparison of the newest snapshot with the one before it from the same source.]"},
		{Page{Name: "snapshots"}, "[The web UI is showing the list of snapshots.]"},
		{Page{Name: "login"}, ""},
	} {
		require.Equal(t, c.want, c.page.note(), "%+v", c.page)
	}
	var none *Page
	require.Equal(t, "", none.note())
}

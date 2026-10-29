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

// Package assistant answers questions about a repository from the web UI.
//
// It drives a model through a tool-use loop whose tools are the ones the
// plakar MCP server exposes: the server runs in-process, reached over an
// in-memory transport, so the assistant sees exactly what any MCP client
// would. Only the read-only tools are registered; the assistant cannot
// change the repository.
package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/PlakarKorp/kloset/logging"
	"github.com/PlakarKorp/plakar/utils"
	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// DefaultModel is the model the assistant talks to with the Anthropic
// provider unless Options.Model names another one.
const DefaultModel = "claude-opus-5-5"

// MaxFileSize bounds what the read_file tool hands the model. It sits under
// maxToolOutput so the tool itself reports the cut, with the size and offset
// the model needs to page, rather than the backstop below slicing its JSON.
const MaxFileSize = 48 << 10

const (
	// maxToolRounds bounds how many times one user message may send the
	// model back to the tools before the turn is cut short.
	maxToolRounds = 25

	// maxToolOutput is the backstop on the bytes of a single tool result fed
	// back to the model; the tools page their own output, so it is not meant
	// to be hit.
	maxToolOutput = 64 << 10

	// maxConversations bounds the conversations held in memory; past it the
	// least recently used idle one is dropped.
	maxConversations = 32

	// maxTurns bounds the messages a conversation accumulates before it has
	// to be started over: every request replays the whole history.
	maxTurns = 200

	// maxTokens bounds one assistant turn. Answers are short; a long task
	// spends its length across tool rounds, not in one message.
	maxTokens = 16000
)

// ErrBusy reports a conversation that is still answering a previous message.
var ErrBusy = errors.New("the assistant is still answering in this conversation")

// ErrTooLong reports a conversation that reached maxTurns.
var ErrTooLong = errors.New("this conversation is too long to continue, start a new one")

// ErrTooMany reports that every conversation slot is answering a message.
var ErrTooMany = errors.New("too many conversations are being answered at once, try again shortly")

// ErrUnknownConversation reports a conversation ID the assistant does not
// hold, forgotten or never created.
var ErrUnknownConversation = errors.New("unknown conversation")

// Options configures an Assistant. FromEnv fills one from the environment.
type Options struct {
	Provider Provider
	// Model is required except with ProviderAnthropic, which has a default.
	Model string
	// APIKey is the provider's credential; the local servers take none.
	APIKey string
	// BaseURL is the API root, defaulted per provider.
	BaseURL string
	Logger  *logging.Logger
}

// Assistant holds the model backend, the MCP session the tools run over and
// the conversations in progress.
type Assistant struct {
	opts    Options
	backend backend
	logger  *logging.Logger

	session *mcp.ClientSession
	serving *mcp.ServerSession
	names   []string

	mu            sync.Mutex
	conversations map[string]*conversation
}

// New connects a client to server over an in-memory transport, lists its
// tools and builds the assistant around them. The sessions last until Close
// or until ctx ends.
func New(ctx context.Context, server *mcp.Server, opts Options) (*Assistant, error) {
	if err := opts.validate(); err != nil {
		return nil, err
	}

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serving, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, fmt.Errorf("mcp server: %w", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "plakar-ui", Version: utils.VERSION}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		_ = serving.Close()
		return nil, fmt.Errorf("mcp client: %w", err)
	}

	listed, err := listTools(ctx, session)
	if err != nil {
		_ = session.Close()
		_ = serving.Close()
		return nil, fmt.Errorf("mcp tools: %w", err)
	}

	specs := make([]toolSpec, 0, len(listed))
	names := make([]string, 0, len(listed))
	for _, tool := range listed {
		spec, err := specOf(tool)
		if err != nil {
			_ = session.Close()
			_ = serving.Close()
			return nil, fmt.Errorf("mcp tool %s: %w", tool.Name, err)
		}
		specs = append(specs, spec)
		names = append(names, tool.Name)
	}

	var back backend
	switch opts.Provider {
	case ProviderAnthropic:
		back, err = newAnthropicBackend(opts.Model, opts.APIKey, opts.BaseURL, specs)
	default:
		back = newOpenAIBackend(opts.Model, opts.APIKey, opts.BaseURL, specs)
	}
	if err != nil {
		_ = session.Close()
		_ = serving.Close()
		return nil, err
	}

	return &Assistant{
		opts:          opts,
		backend:       back,
		logger:        opts.Logger,
		session:       session,
		serving:       serving,
		names:         names,
		conversations: make(map[string]*conversation),
	}, nil
}

// Close ends the MCP sessions. Conversations in progress fail their tool
// calls from then on.
func (a *Assistant) Close() error {
	err := a.session.Close()
	if serr := a.serving.Close(); err == nil {
		err = serr
	}
	return err
}

// Model reports the model the assistant talks to.
func (a *Assistant) Model() string { return a.opts.Model }

// Tools reports the names of the tools the assistant may call.
func (a *Assistant) Tools() []string { return a.names }

func listTools(ctx context.Context, session *mcp.ClientSession) ([]*mcp.Tool, error) {
	var tools []*mcp.Tool
	params := &mcp.ListToolsParams{}
	for {
		res, err := session.ListTools(ctx, params)
		if err != nil {
			return nil, err
		}
		tools = append(tools, res.Tools...)
		if res.NextCursor == "" {
			return tools, nil
		}
		params.Cursor = res.NextCursor
	}
}

// specOf reads an MCP tool description. The input schema crosses as JSON,
// whatever Go value the server built it from.
func specOf(tool *mcp.Tool) (toolSpec, error) {
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		return toolSpec{}, err
	}
	var schema map[string]any
	if err := json.Unmarshal(raw, &schema); err != nil {
		return toolSpec{}, err
	}
	if schema == nil {
		schema = map[string]any{}
	}
	if _, ok := schema["type"]; !ok {
		schema["type"] = "object"
	}
	return toolSpec{Name: tool.Name, Description: tool.Description, Schema: schema}, nil
}

// conversation is one chat thread: the messages replayed on every request.
// A conversation answers one message at a time; busy marks the one in
// progress and is only touched under the assistant lock.
type conversation struct {
	id       string
	busy     bool
	messages []message
	lastUsed time.Time
}

// acquire claims the conversation id names, or a new one when id is empty,
// for one answer; release gives it back. A new conversation may push the
// least recently used idle one out, and when none is idle there is no room.
func (a *Assistant) acquire(id string) (*conversation, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if id != "" {
		conv, ok := a.conversations[id]
		if !ok {
			return nil, ErrUnknownConversation
		}
		if conv.busy {
			return nil, ErrBusy
		}
		conv.busy = true
		conv.lastUsed = time.Now()
		return conv, nil
	}

	if len(a.conversations) >= maxConversations {
		a.evictLocked()
	}
	if len(a.conversations) >= maxConversations {
		return nil, ErrTooMany
	}

	conv := &conversation{id: uuid.NewString(), busy: true, lastUsed: time.Now()}
	a.conversations[conv.id] = conv
	return conv, nil
}

func (a *Assistant) release(conv *conversation) {
	a.mu.Lock()
	defer a.mu.Unlock()
	conv.busy = false
	conv.lastUsed = time.Now()
}

// evictLocked drops the least recently used conversation that is not
// answering; one in progress is never pulled from under its run.
func (a *Assistant) evictLocked() {
	var oldest *conversation
	for _, conv := range a.conversations {
		if conv.busy {
			continue
		}
		if oldest == nil || conv.lastUsed.Before(oldest.lastUsed) {
			oldest = conv
		}
	}
	if oldest != nil {
		delete(a.conversations, oldest.id)
	}
}

// Forget drops a conversation. It reports whether there was one to drop. A
// run in progress on it finishes into a conversation nobody can reach.
func (a *Assistant) Forget(id string) bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	_, ok := a.conversations[id]
	delete(a.conversations, id)
	return ok
}

func (a *Assistant) logf(format string, args ...any) {
	if a.logger != nil {
		a.logger.Warn(format, args...)
	}
}

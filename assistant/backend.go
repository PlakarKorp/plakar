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
	"context"
	"encoding/json"
)

// The model is reached through a backend, one per provider family. The
// loop in run.go only ever sees the neutral types below; what a provider
// needs replayed verbatim, such as the thinking blocks the Messages API
// checks against the history that produced them, travels in native.

// toolSpec is a tool as the model sees it: the MCP description, with its
// input schema as the JSON object the server published.
type toolSpec struct {
	Name        string
	Description string
	Schema      map[string]any
}

// toolCall is a call the model asked for.
type toolCall struct {
	ID    string
	Name  string
	Input json.RawMessage
}

// toolResult is what a call returned, for the model to read.
type toolResult struct {
	ID      string
	Output  string
	IsError bool
}

type role int

const (
	roleUser role = iota
	roleAssistant
	roleTool
)

// message is one entry of a conversation: the user's text, the assistant's
// turn, or the results of the tool calls it asked for.
type message struct {
	role    role
	text    string
	calls   []toolCall
	results []toolResult
	// native holds the backend's own form of an assistant turn, when it
	// has one it wants replayed as is.
	native any
}

type stopReason string

const (
	stopEndTurn   stopReason = "end_turn"
	stopToolUse   stopReason = "tool_use"
	stopMaxTokens stopReason = "max_tokens"
	stopRefusal   stopReason = "refusal"
)

// turn is one assistant message as the model ended it.
type turn struct {
	text        string
	calls       []toolCall
	stop        stopReason
	explanation string
	native      any
}

// backend sends one request to the model and relays the text as it
// arrives, through emit, then returns the whole turn.
type backend interface {
	stream(ctx context.Context, history []message, emit EventFunc) (*turn, error)
}

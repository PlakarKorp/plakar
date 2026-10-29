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
	"fmt"
	"slices"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// EventFunc receives the steps of a run as they happen, named after the
// payload types below: text, tool_use, tool_result, done.
type EventFunc func(name string, payload any) error

// TextEvent carries a piece of the answer as the model writes it.
type TextEvent struct {
	Text string `json:"text"`
}

// ToolUseEvent announces a tool call the model started; its input is still
// streaming and comes with the ToolResultEvent.
type ToolUseEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// ToolResultEvent reports what a tool call was given and what it returned.
type ToolResultEvent struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Output    string          `json:"output"`
	IsError   bool            `json:"is_error"`
	Truncated bool            `json:"truncated,omitempty"`
}

// DoneEvent closes a turn. Explanation is set when the model declined.
type DoneEvent struct {
	StopReason  string `json:"stop_reason"`
	Explanation string `json:"explanation,omitempty"`
}

// run answers text in conv, emitting events on the way. The caller holds
// the conversation through acquire, so one message is answered at a time.
//
// The conversation only learns of the turn once it completed: a turn that
// failed, was interrupted or was declined leaves it as it was, so the next
// message starts from a history the API accepts. Keeping a partial turn
// would strand tool calls without results, and every later request would
// be refused for it.
func (a *Assistant) run(ctx context.Context, conv *conversation, text string, emit EventFunc) error {
	messages := append(slices.Clone(conv.messages), message{role: roleUser, text: text})

	for round := 0; ; round++ {
		t, err := a.backend.stream(ctx, messages, emit)
		if err != nil {
			return err
		}

		if t.stop == stopRefusal {
			return emit("done", DoneEvent{StopReason: string(t.stop), Explanation: t.explanation})
		}

		messages = append(messages, message{role: roleAssistant, text: t.text, calls: t.calls, native: t.native})

		if t.stop != stopToolUse {
			conv.messages = messages
			return emit("done", DoneEvent{StopReason: string(t.stop)})
		}

		results, err := a.callTools(ctx, t.calls, round < maxToolRounds, emit)
		if err != nil {
			return err
		}
		messages = append(messages, message{role: roleTool, results: results})

		if round >= maxToolRounds {
			return fmt.Errorf("stopped after %d rounds of tool calls without an answer", maxToolRounds)
		}
	}
}

// callTools runs every call through the MCP session and returns their
// results, in the order the model asked for them. With allowed false the
// calls are refused instead, so the history stays well-formed when the
// round budget is spent.
func (a *Assistant) callTools(ctx context.Context, calls []toolCall, allowed bool, emit EventFunc) ([]toolResult, error) {
	results := make([]toolResult, 0, len(calls))
	for _, call := range calls {
		output, isError := a.callTool(ctx, call, allowed)
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		truncated := false
		if len(output) > maxToolOutput {
			output = strings.ToValidUTF8(output[:maxToolOutput], "") + "\n[output truncated]"
			truncated = true
		}
		if output == "" {
			output = "(no output)"
		}

		if err := emit("tool_result", ToolResultEvent{
			ID:        call.ID,
			Name:      call.Name,
			Input:     call.Input,
			Output:    output,
			IsError:   isError,
			Truncated: truncated,
		}); err != nil {
			return nil, err
		}
		results = append(results, toolResult{ID: call.ID, Output: output, IsError: isError})
	}
	return results, nil
}

func (a *Assistant) callTool(ctx context.Context, call toolCall, allowed bool) (string, bool) {
	if !allowed {
		return fmt.Sprintf("not executed: this turn already made %d rounds of tool calls, answer with what you have", maxToolRounds), true
	}

	var args map[string]any
	if err := json.Unmarshal(call.Input, &args); err != nil {
		return fmt.Sprintf("invalid tool input: %s", err), true
	}

	// A failed call is the model's to see and recover from, so it goes
	// back as an error result rather than ending the turn.
	res, err := a.session.CallTool(ctx, &mcp.CallToolParams{Name: call.Name, Arguments: args})
	if err != nil {
		return err.Error(), true
	}
	return toolText(res), res.IsError
}

// toolText flattens a tool result to the text the model reads: the text
// blocks, or the structured output when the server sent only that.
func toolText(res *mcp.CallToolResult) string {
	var parts []string
	for _, content := range res.Content {
		if text, ok := content.(*mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	if len(parts) == 0 && res.StructuredContent != nil {
		if raw, err := json.Marshal(res.StructuredContent); err == nil {
			return string(raw)
		}
	}
	return strings.Join(parts, "\n")
}

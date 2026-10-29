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
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// openaiBackend talks to a chat completions endpoint: OpenAI's own, and the
// ones Gemini, LM Studio, Ollama and most other servers expose in the same
// shape. It is hand-written rather than an SDK because the local servers
// follow the format loosely, and the subset used here is small.
type openaiBackend struct {
	client  *http.Client
	baseURL string
	apiKey  string
	model   string
	tools   []chatTool
}

func newOpenAIBackend(model, apiKey, baseURL string, tools []toolSpec) *openaiBackend {
	chatTools := make([]chatTool, 0, len(tools))
	for _, tool := range tools {
		// A tool without parameters still gets an empty properties map:
		// LM Studio rejects a function schema without one as having no
		// parameters at all.
		parameters := make(map[string]any, len(tool.Schema)+1)
		for key, value := range tool.Schema {
			parameters[key] = value
		}
		if _, ok := parameters["properties"]; !ok {
			parameters["properties"] = map[string]any{}
		}
		chatTools = append(chatTools, chatTool{
			Type: "function",
			Function: chatFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  parameters,
			},
		})
	}
	return &openaiBackend{
		// No timeout on the client: an answer streams for as long as it
		// takes, and the request context ends it when the browser leaves.
		client:  &http.Client{},
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		model:   model,
		tools:   chatTools,
	}
}

type chatTool struct {
	Type     string       `json:"type"`
	Function chatFunction `json:"function"`
}

type chatFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

type chatToolCall struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function chatFunctionCall `json:"function"`
}

type chatFunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type chatMessage struct {
	Role       string         `json:"role"`
	Content    *string        `json:"content"`
	ToolCalls  []chatToolCall `json:"tool_calls,omitempty"`
	ToolCallID string         `json:"tool_call_id,omitempty"`
}

type chatRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Tools    []chatTool    `json:"tools,omitempty"`
	Stream   bool          `json:"stream"`
}

// chatChunk is one streamed piece of a completion. Tool calls arrive in
// fragments keyed by index: the first carries the id and name, the rest
// append to the arguments.
type chatChunk struct {
	Choices []struct {
		Delta struct {
			Content   string `json:"content"`
			ToolCalls []struct {
				Index    int    `json:"index"`
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Error *chatError `json:"error"`
}

type chatError struct {
	Message string `json:"message"`
}

// messages converts the history to the chat completions form.
func (b *openaiBackend) messages(history []message) []chatMessage {
	out := make([]chatMessage, 0, 1+len(history))
	system := systemPrompt
	out = append(out, chatMessage{Role: "system", Content: &system})
	for _, m := range history {
		switch m.role {
		case roleUser:
			text := m.text
			out = append(out, chatMessage{Role: "user", Content: &text})
		case roleAssistant:
			msg := chatMessage{Role: "assistant"}
			if m.text != "" {
				text := m.text
				msg.Content = &text
			}
			for _, call := range m.calls {
				msg.ToolCalls = append(msg.ToolCalls, chatToolCall{
					ID:   call.ID,
					Type: "function",
					Function: chatFunctionCall{
						Name:      call.Name,
						Arguments: string(call.Input),
					},
				})
			}
			out = append(out, msg)
		case roleTool:
			for _, result := range m.results {
				output := result.Output
				out = append(out, chatMessage{Role: "tool", ToolCallID: result.ID, Content: &output})
			}
		}
	}
	return out
}

// maxChunkLine bounds one line of the stream; a chunk is a few hundred
// bytes, and a server sending megabytes in one is not one to keep reading.
const maxChunkLine = 1 << 20

func (b *openaiBackend) stream(ctx context.Context, history []message, emit EventFunc) (*turn, error) {
	body, err := json.Marshal(chatRequest{
		Model:    b.model,
		Messages: b.messages(history),
		Tools:    b.tools,
		Stream:   true,
	})
	if err != nil {
		return nil, fmt.Errorf("model request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, b.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("model request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	if b.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+b.apiKey)
	}

	res, err := b.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("the model endpoint %s could not be reached: %w", b.baseURL, err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, describeChatFailure(res)
	}

	t := &turn{stop: stopEndTurn}
	// Calls in the order the model opened them, found again by index as
	// their fragments arrive.
	var order []int
	pending := make(map[int]*pendingCall)

	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 64<<10), maxChunkLine)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			break
		}

		var chunk chatChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return nil, fmt.Errorf("model stream: %w", err)
		}
		if chunk.Error != nil {
			return nil, fmt.Errorf("the model endpoint failed: %s", chunk.Error.Message)
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		choice := chunk.Choices[0]

		if choice.Delta.Content != "" {
			t.text += choice.Delta.Content
			if err := emit("text", TextEvent{Text: choice.Delta.Content}); err != nil {
				return nil, err
			}
		}

		for _, fragment := range choice.Delta.ToolCalls {
			call, ok := pending[fragment.Index]
			if !ok {
				call = &pendingCall{id: fragment.ID, name: fragment.Function.Name}
				if call.id == "" {
					// Some local servers leave the id out; the model only
					// needs it to be unique within the turn.
					call.id = fmt.Sprintf("call_%d", len(order)+1)
				}
				pending[fragment.Index] = call
				order = append(order, fragment.Index)
				if err := emit("tool_use", ToolUseEvent{ID: call.id, Name: call.name}); err != nil {
					return nil, err
				}
			} else if call.name == "" && fragment.Function.Name != "" {
				call.name = fragment.Function.Name
			}
			call.args.WriteString(fragment.Function.Arguments)
		}

		switch choice.FinishReason {
		case "tool_calls":
			t.stop = stopToolUse
		case "length":
			t.stop = stopMaxTokens
		case "content_filter":
			t.stop = stopRefusal
			t.explanation = "the model's content filter stopped the answer"
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("model stream: %w", err)
	}

	for _, index := range order {
		call := pending[index]
		input := strings.TrimSpace(call.args.String())
		if input == "" {
			input = "{}"
		}
		t.calls = append(t.calls, toolCall{ID: call.id, Name: call.name, Input: json.RawMessage(input)})
	}
	// A server that forgets to say so still asked for its tools.
	if len(t.calls) > 0 && t.stop == stopEndTurn {
		t.stop = stopToolUse
	}
	return t, nil
}

// pendingCall is a tool call whose arguments are still streaming in.
type pendingCall struct {
	id   string
	name string
	args strings.Builder
}

// describeChatFailure turns an error response into a line the UI can show.
func describeChatFailure(res *http.Response) error {
	body, _ := io.ReadAll(io.LimitReader(res.Body, 64<<10))
	var payload struct {
		Error *chatError `json:"error"`
	}
	detail := strings.TrimSpace(string(body))
	if json.Unmarshal(body, &payload) == nil && payload.Error != nil && payload.Error.Message != "" {
		detail = payload.Error.Message
	}
	switch res.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("the model endpoint refused the credentials (HTTP %d): %s", res.StatusCode, detail)
	case http.StatusTooManyRequests:
		return fmt.Errorf("the model endpoint is rate limiting requests, try again shortly: %s", detail)
	case http.StatusNotFound:
		return fmt.Errorf("the model endpoint has no such model or route (HTTP 404): %s", detail)
	}
	return fmt.Errorf("the model endpoint failed (HTTP %d): %s", res.StatusCode, detail)
}

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
	"errors"
	"fmt"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"
)

// anthropicBackend talks to the Messages API through the Anthropic SDK.
type anthropicBackend struct {
	client anthropic.Client
	model  string
	tools  []anthropic.BetaToolUnionParam
	system []anthropic.BetaTextBlockParam
}

func newAnthropicBackend(model, apiKey, baseURL string, tools []toolSpec) (*anthropicBackend, error) {
	params := make([]anthropic.BetaToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		param, err := anthropicTool(tool)
		if err != nil {
			return nil, fmt.Errorf("tool %s: %w", tool.Name, err)
		}
		params = append(params, param)
	}

	var opts []option.RequestOption
	if apiKey != "" {
		opts = append(opts, option.WithAPIKey(apiKey))
	}
	if baseURL != "" {
		opts = append(opts, option.WithBaseURL(baseURL))
	}

	return &anthropicBackend{
		client: anthropic.NewClient(opts...),
		model:  model,
		tools:  params,
		// The system prompt closes the cached prefix: tools, then system,
		// then the conversation that varies.
		system: []anthropic.BetaTextBlockParam{{
			Text:         systemPrompt,
			CacheControl: anthropic.NewBetaCacheControlEphemeralParam(),
		}},
	}, nil
}

// anthropicTool turns a tool description into the definition the model
// sees: properties and required go in their own fields, anything else the
// schema says rides along untouched.
func anthropicTool(tool toolSpec) (anthropic.BetaToolUnionParam, error) {
	param := anthropic.BetaToolParam{
		Name:        tool.Name,
		Description: anthropic.String(tool.Description),
	}
	extra := make(map[string]any)
	for key, value := range tool.Schema {
		switch key {
		case "type":
		case "properties":
			param.InputSchema.Properties = value
		case "required":
			list, ok := value.([]any)
			if !ok {
				return anthropic.BetaToolUnionParam{}, fmt.Errorf("required is not a list")
			}
			for _, item := range list {
				name, ok := item.(string)
				if !ok {
					return anthropic.BetaToolUnionParam{}, fmt.Errorf("required holds a non-string")
				}
				param.InputSchema.Required = append(param.InputSchema.Required, name)
			}
		default:
			extra[key] = value
		}
	}
	if len(extra) > 0 {
		param.InputSchema.ExtraFields = extra
	}
	return anthropic.BetaToolUnionParam{OfTool: &param}, nil
}

// messages converts the history to the API's form. An assistant turn is
// replayed as the API returned it, thinking blocks included, since the API
// checks those against the history that produced them.
func (b *anthropicBackend) messages(history []message) []anthropic.BetaMessageParam {
	out := make([]anthropic.BetaMessageParam, 0, len(history))
	for _, m := range history {
		switch m.role {
		case roleUser:
			out = append(out, anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(m.text)))
		case roleAssistant:
			if native, ok := m.native.(anthropic.BetaMessageParam); ok {
				out = append(out, native)
				continue
			}
			blocks := make([]anthropic.BetaContentBlockParamUnion, 0, 1+len(m.calls))
			if m.text != "" {
				blocks = append(blocks, anthropic.NewBetaTextBlock(m.text))
			}
			for _, call := range m.calls {
				blocks = append(blocks, anthropic.BetaContentBlockParamUnion{
					OfToolUse: &anthropic.BetaToolUseBlockParam{ID: call.ID, Name: call.Name, Input: call.Input},
				})
			}
			out = append(out, anthropic.BetaMessageParam{Role: anthropic.BetaMessageParamRoleAssistant, Content: blocks})
		case roleTool:
			blocks := make([]anthropic.BetaContentBlockParamUnion, 0, len(m.results))
			for _, result := range m.results {
				blocks = append(blocks, anthropic.NewBetaToolResultBlock(result.ID, result.Output, result.IsError))
			}
			out = append(out, anthropic.NewBetaUserMessage(blocks...))
		}
	}
	return out
}

func (b *anthropicBackend) stream(ctx context.Context, history []message, emit EventFunc) (*turn, error) {
	params := anthropic.BetaMessageNewParams{
		Model:     anthropic.Model(b.model),
		MaxTokens: maxTokens,
		System:    b.system,
		Messages:  b.messages(history),
		Tools:     b.tools,
		OutputConfig: anthropic.BetaOutputConfigParam{
			Effort: anthropic.BetaOutputConfigEffortMedium,
		},
		// A turn the model's safety classifiers decline is re-served by a
		// fallback model inside the same request rather than lost.
		Fallbacks: anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()},
		Betas:     []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01},
	}

	stream := b.client.Beta.Messages.NewStreaming(ctx, params)
	defer stream.Close()

	var msg anthropic.BetaMessage
	for stream.Next() {
		event := stream.Current()
		if err := msg.Accumulate(event); err != nil {
			return nil, fmt.Errorf("model stream: %w", err)
		}

		switch variant := event.AsAny().(type) {
		case anthropic.BetaRawContentBlockStartEvent:
			if use, ok := variant.ContentBlock.AsAny().(anthropic.BetaToolUseBlock); ok {
				if err := emit("tool_use", ToolUseEvent{ID: use.ID, Name: use.Name}); err != nil {
					return nil, err
				}
			}
		case anthropic.BetaRawContentBlockDeltaEvent:
			if delta, ok := variant.Delta.AsAny().(anthropic.BetaTextDelta); ok {
				if err := emit("text", TextEvent{Text: delta.Text}); err != nil {
					return nil, err
				}
			}
		}
	}
	if err := stream.Err(); err != nil {
		return nil, describeAnthropic(err)
	}

	t := &turn{native: msg.ToParam(), explanation: msg.StopDetails.Explanation}
	switch msg.StopReason {
	case anthropic.BetaStopReasonToolUse:
		t.stop = stopToolUse
	case anthropic.BetaStopReasonMaxTokens:
		t.stop = stopMaxTokens
	case anthropic.BetaStopReasonRefusal:
		t.stop = stopRefusal
	default:
		t.stop = stopEndTurn
	}
	for _, block := range msg.Content {
		switch variant := block.AsAny().(type) {
		case anthropic.BetaTextBlock:
			t.text += variant.Text
		case anthropic.BetaToolUseBlock:
			t.calls = append(t.calls, toolCall{
				ID:    variant.ID,
				Name:  variant.Name,
				Input: json.RawMessage(variant.JSON.Input.Raw()),
			})
		}
	}
	return t, nil
}

// describeAnthropic turns an SDK error into a line the UI can show.
func describeAnthropic(err error) error {
	var apierr *anthropic.Error
	if !errors.As(err, &apierr) {
		return fmt.Errorf("model: %w", err)
	}
	switch apierr.StatusCode {
	case 401, 403:
		return fmt.Errorf("the Claude API refused the credentials, check ANTHROPIC_API_KEY where plakar ui runs: %w", err)
	case 429:
		return fmt.Errorf("the Claude API is rate limiting requests, try again shortly: %w", err)
	}
	return fmt.Errorf("the Claude API failed: %w", err)
}

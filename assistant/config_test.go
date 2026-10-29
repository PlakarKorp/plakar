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
	"testing"

	"github.com/stretchr/testify/require"
)

func env(values map[string]string) func(string) string {
	return func(key string) string { return values[key] }
}

func TestFromEnvDisabled(t *testing.T) {
	_, enabled, err := FromEnv(env(nil))
	require.NoError(t, err)
	require.False(t, enabled)
}

func TestFromEnvDetectsProviderFromKey(t *testing.T) {
	opts, enabled, err := FromEnv(env(map[string]string{"ANTHROPIC_API_KEY": "a"}))
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, ProviderAnthropic, opts.Provider)
	require.Equal(t, DefaultModel, opts.Model)
	require.Equal(t, "a", opts.APIKey)
	require.False(t, opts.local())

	opts, enabled, err = FromEnv(env(map[string]string{"OPENAI_API_KEY": "o", "PLAKAR_ASSISTANT_MODEL": "gpt"}))
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, ProviderOpenAI, opts.Provider)
	require.Equal(t, "https://api.openai.com/v1", opts.BaseURL)
	require.Equal(t, "o", opts.APIKey)
	require.Equal(t, "OpenAI", opts.label())

	opts, _, err = FromEnv(env(map[string]string{"GEMINI_API_KEY": "g", "PLAKAR_ASSISTANT_MODEL": "gemini"}))
	require.NoError(t, err)
	require.Equal(t, ProviderGemini, opts.Provider)
	require.Equal(t, "Google", opts.label())
}

func TestFromEnvRequiresModelOutsideAnthropic(t *testing.T) {
	_, _, err := FromEnv(env(map[string]string{"OPENAI_API_KEY": "o"}))
	require.ErrorContains(t, err, "PLAKAR_ASSISTANT_MODEL")

	_, _, err = FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "lmstudio"}))
	require.ErrorContains(t, err, "PLAKAR_ASSISTANT_MODEL")
}

func TestFromEnvLocalProviders(t *testing.T) {
	opts, enabled, err := FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "LMStudio", "PLAKAR_ASSISTANT_MODEL": "qwen"}))
	require.NoError(t, err)
	require.True(t, enabled)
	require.Equal(t, ProviderLMStudio, opts.Provider)
	require.Equal(t, "http://localhost:1234/v1", opts.BaseURL)
	require.Empty(t, opts.APIKey)
	require.True(t, opts.local())

	opts, _, err = FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "ollama", "PLAKAR_ASSISTANT_MODEL": "llama", "PLAKAR_ASSISTANT_URL": "http://nas.local:11434/v1"}))
	require.NoError(t, err)
	require.Equal(t, "http://nas.local:11434/v1", opts.BaseURL)
	require.True(t, opts.local(), "a named local server stays local wherever it is bound")
}

func TestFromEnvCompatible(t *testing.T) {
	_, _, err := FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "openai-compatible", "PLAKAR_ASSISTANT_MODEL": "m"}))
	require.ErrorContains(t, err, "PLAKAR_ASSISTANT_URL")

	opts, _, err := FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "openai-compatible", "PLAKAR_ASSISTANT_MODEL": "m", "PLAKAR_ASSISTANT_URL": "http://127.0.0.1:8000/v1"}))
	require.NoError(t, err)
	require.True(t, opts.local())

	opts, _, err = FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "openai-compatible", "PLAKAR_ASSISTANT_MODEL": "m", "PLAKAR_ASSISTANT_URL": "https://inference.example.com/v1", "PLAKAR_ASSISTANT_API_KEY": "k"}))
	require.NoError(t, err)
	require.False(t, opts.local())
	require.Equal(t, "k", opts.APIKey)
}

func TestFromEnvRejectsUnknownProvider(t *testing.T) {
	_, _, err := FromEnv(env(map[string]string{"PLAKAR_ASSISTANT_PROVIDER": "bard"}))
	require.ErrorContains(t, err, "unknown provider")
}

func TestFromEnvExplicitProviderWinsOverKeys(t *testing.T) {
	opts, _, err := FromEnv(env(map[string]string{
		"ANTHROPIC_API_KEY":         "a",
		"PLAKAR_ASSISTANT_PROVIDER": "openai",
		"OPENAI_API_KEY":            "o",
		"PLAKAR_ASSISTANT_MODEL":    "gpt",
	}))
	require.NoError(t, err)
	require.Equal(t, ProviderOpenAI, opts.Provider)
	require.Equal(t, "o", opts.APIKey)
}

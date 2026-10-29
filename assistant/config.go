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
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Provider names the family of API the assistant talks to.
type Provider string

const (
	// ProviderAnthropic is the Claude Messages API.
	ProviderAnthropic Provider = "anthropic"
	// ProviderOpenAI is OpenAI's chat completions API.
	ProviderOpenAI Provider = "openai"
	// ProviderGemini is Google's chat completions endpoint for Gemini.
	ProviderGemini Provider = "gemini"
	// ProviderLMStudio is the LM Studio local server.
	ProviderLMStudio Provider = "lmstudio"
	// ProviderOllama is the Ollama local server.
	ProviderOllama Provider = "ollama"
	// ProviderCompatible is any other chat completions endpoint, named by
	// its URL.
	ProviderCompatible Provider = "openai-compatible"
)

// providerInfo is what the code knows about each provider: where its API
// lives, which variable holds its key, how to call it in front of people.
type providerInfo struct {
	label   string
	keyVar  string
	baseURL string
	local   bool
}

var providers = map[Provider]providerInfo{
	ProviderAnthropic:  {label: "Anthropic", keyVar: "ANTHROPIC_API_KEY"},
	ProviderOpenAI:     {label: "OpenAI", keyVar: "OPENAI_API_KEY", baseURL: "https://api.openai.com/v1"},
	ProviderGemini:     {label: "Google", keyVar: "GEMINI_API_KEY", baseURL: "https://generativelanguage.googleapis.com/v1beta/openai"},
	ProviderLMStudio:   {label: "LM Studio", baseURL: "http://localhost:1234/v1", local: true},
	ProviderOllama:     {label: "Ollama", baseURL: "http://localhost:11434/v1", local: true},
	ProviderCompatible: {label: "the configured endpoint"},
}

// detectionOrder is which provider an unset PLAKAR_ASSISTANT_PROVIDER
// resolves to: the first whose key is in the environment. Local servers
// have no key and so are only ever named explicitly.
var detectionOrder = []Provider{ProviderAnthropic, ProviderOpenAI, ProviderGemini}

// FromEnv reads the assistant's configuration from the environment:
//
//	PLAKAR_ASSISTANT_PROVIDER  anthropic, openai, gemini, lmstudio, ollama or
//	                           openai-compatible; or else the first provider
//	                           whose key variable is set
//	PLAKAR_ASSISTANT_MODEL     the model, required except with anthropic
//	PLAKAR_ASSISTANT_URL       the API base URL, required with
//	                           openai-compatible, optional otherwise
//	PLAKAR_ASSISTANT_API_KEY   the key, in place of the provider's own
//	                           variable (ANTHROPIC_API_KEY, OPENAI_API_KEY,
//	                           GEMINI_API_KEY)
//
// It reports enabled false, and no error, when nothing in the environment
// asks for the assistant.
func FromEnv(getenv func(string) string) (opts Options, enabled bool, err error) {
	provider := Provider(strings.ToLower(strings.TrimSpace(getenv("PLAKAR_ASSISTANT_PROVIDER"))))
	if provider == "" {
		for _, candidate := range detectionOrder {
			if getenv(providers[candidate].keyVar) != "" {
				provider = candidate
				break
			}
		}
		if provider == "" {
			return Options{}, false, nil
		}
	}

	info, ok := providers[provider]
	if !ok {
		return Options{}, false, fmt.Errorf("PLAKAR_ASSISTANT_PROVIDER: unknown provider %q", provider)
	}

	opts = Options{
		Provider: provider,
		Model:    strings.TrimSpace(getenv("PLAKAR_ASSISTANT_MODEL")),
		BaseURL:  strings.TrimSpace(getenv("PLAKAR_ASSISTANT_URL")),
		APIKey:   getenv("PLAKAR_ASSISTANT_API_KEY"),
	}
	if opts.APIKey == "" && info.keyVar != "" {
		opts.APIKey = getenv(info.keyVar)
	}
	if err := opts.validate(); err != nil {
		return Options{}, false, err
	}
	return opts, true, nil
}

// validate checks the options against what the provider needs, naming the
// variable to set rather than failing later in a request.
func (o *Options) validate() error {
	info, ok := providers[o.Provider]
	if !ok {
		return fmt.Errorf("unknown provider %q", o.Provider)
	}
	if o.Model == "" {
		if o.Provider != ProviderAnthropic {
			return fmt.Errorf("PLAKAR_ASSISTANT_MODEL is required with provider %s", o.Provider)
		}
		o.Model = DefaultModel
	}
	if o.BaseURL == "" {
		if o.Provider == ProviderCompatible {
			return fmt.Errorf("PLAKAR_ASSISTANT_URL is required with provider %s", o.Provider)
		}
		o.BaseURL = info.baseURL
	}
	if o.APIKey == "" && info.keyVar != "" && o.Provider != ProviderAnthropic {
		return fmt.Errorf("%s is required with provider %s", info.keyVar, o.Provider)
	}
	return nil
}

// label is the provider's name as shown to people.
func (o *Options) label() string {
	return providers[o.Provider].label
}

// local reports whether messages stay on this machine: the local servers,
// and any other endpoint bound to a loopback address.
func (o *Options) local() bool {
	if providers[o.Provider].local {
		return true
	}
	if o.Provider != ProviderCompatible {
		return false
	}
	u, err := url.Parse(o.BaseURL)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

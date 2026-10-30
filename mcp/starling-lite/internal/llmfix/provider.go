// Package llmfix is the LLM fallback for remediation proposals
// (propose_remediation_llm). Deterministic rules always run first; LLM output
// is treated as untrusted and must pass strict parsing, a denylist, a ShrikeOps
// re-scan and (when a cluster is available) a server-side dry-run.
package llmfix

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Provider completes a single system+user prompt and returns raw text.
type Provider interface {
	Name() string
	Complete(ctx context.Context, system, user string) (string, error)
}

// ErrDisabled is returned by FromEnv when LLM_PROVIDER is unset.
var ErrDisabled = errors.New("LLM fallback disabled: set LLM_PROVIDER=vertex|openai|ollama|avirka")

const (
	defaultTimeout   = 30 * time.Second
	maxTimeout       = 120 * time.Second
	maxHTTPBody      = 1 << 20  // 1 MiB raw HTTP response
	maxCompletionLen = 32 << 10 // 32 KiB completion text
)

// Timeout returns LLM_TIMEOUT_SECONDS (default 30s, capped at 120s).
func Timeout(getenv func(string) string) time.Duration {
	if v := strings.TrimSpace(getenv("LLM_TIMEOUT_SECONDS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			d := time.Duration(n) * time.Second
			if d > maxTimeout {
				return maxTimeout
			}
			return d
		}
	}
	return defaultTimeout
}

// FromEnv builds the configured provider. API keys are read only from the
// environment; a missing key fails closed (error, no provider).
func FromEnv(getenv func(string) string) (Provider, error) {
	name := strings.ToLower(strings.TrimSpace(getenv("LLM_PROVIDER")))
	model := strings.TrimSpace(getenv("LLM_MODEL"))
	client := &http.Client{Timeout: Timeout(getenv)}
	need := func(k string) (string, error) {
		v := strings.TrimSpace(getenv(k))
		if v == "" {
			return "", fmt.Errorf("LLM_PROVIDER=%s requires env %s (refusing to run without it)", name, k)
		}
		return v, nil
	}
	switch name {
	case "":
		return nil, ErrDisabled
	case "openai":
		key, err := need("OPENAI_API_KEY")
		if err != nil {
			return nil, err
		}
		base := strings.TrimSpace(getenv("OPENAI_BASE_URL"))
		if base == "" {
			base = "https://api.openai.com/v1"
		}
		if model == "" {
			model = "gpt-4o-mini"
		}
		return &openAICompat{name: "openai", baseURL: base, apiKey: key, model: model, client: client}, nil
	case "avirka":
		key, err := need("AVIRKA_API_KEY")
		if err != nil {
			return nil, err
		}
		base, err := need("AVIRKA_BASE_URL")
		if err != nil {
			return nil, err
		}
		if model == "" {
			return nil, fmt.Errorf("LLM_PROVIDER=avirka requires env LLM_MODEL")
		}
		return &openAICompat{name: "avirka", baseURL: base, apiKey: key, model: model, client: client}, nil
	case "vertex":
		tok, err := need("VERTEX_ACCESS_TOKEN")
		if err != nil {
			return nil, err
		}
		project, err := need("VERTEX_PROJECT")
		if err != nil {
			return nil, err
		}
		loc := strings.TrimSpace(getenv("VERTEX_LOCATION"))
		if loc == "" {
			loc = "us-central1"
		}
		if model == "" {
			model = "gemini-2.5-flash"
		}
		return &vertex{token: tok, project: project, location: loc, model: model, client: client}, nil
	case "ollama":
		host := strings.TrimSpace(getenv("OLLAMA_HOST"))
		if host == "" {
			host = "http://127.0.0.1:11434"
		}
		if !strings.HasPrefix(host, "http://") && !strings.HasPrefix(host, "https://") {
			host = "http://" + host
		}
		if model == "" {
			model = "llama3.1"
		}
		return &ollama{host: strings.TrimRight(host, "/"), model: model, client: client}, nil
	}
	return nil, fmt.Errorf("unknown LLM_PROVIDER %q (want vertex|openai|ollama|avirka)", name)
}

func postJSON(ctx context.Context, c *http.Client, url string, headers map[string]string, in, out any) error {
	body, err := json.Marshal(in)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("llm request: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, maxHTTPBody))
	if err != nil {
		return fmt.Errorf("llm read: %w", err)
	}
	if resp.StatusCode/100 != 2 {
		// Never echo request headers (keys); the body is truncated.
		snippet := string(raw)
		if len(snippet) > 300 {
			snippet = snippet[:300]
		}
		return fmt.Errorf("llm http %d: %s", resp.StatusCode, snippet)
	}
	return json.Unmarshal(raw, out)
}

func capText(s string) (string, error) {
	if len(s) > maxCompletionLen {
		return "", fmt.Errorf("llm completion too large (%d bytes > %d)", len(s), maxCompletionLen)
	}
	return s, nil
}

// openAICompat speaks the OpenAI chat-completions API (also used for avirka).
type openAICompat struct {
	name, baseURL, apiKey, model string
	client                       *http.Client
}

func (p *openAICompat) Name() string { return p.name + ":" + p.model }

func (p *openAICompat) Complete(ctx context.Context, system, user string) (string, error) {
	in := map[string]any{
		"model":           p.model,
		"temperature":     0,
		"max_tokens":      2048,
		"response_format": map[string]string{"type": "json_object"},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	var out struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	url := strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	if err := postJSON(ctx, p.client, url, map[string]string{"Authorization": "Bearer " + p.apiKey}, in, &out); err != nil {
		return "", err
	}
	if len(out.Choices) == 0 {
		return "", errors.New("llm returned no choices")
	}
	return capText(out.Choices[0].Message.Content)
}

// vertex calls Gemini on Vertex AI generateContent with a caller-supplied
// OAuth access token (VERTEX_ACCESS_TOKEN).
type vertex struct {
	token, project, location, model string
	client                          *http.Client
}

func (p *vertex) Name() string { return "vertex:" + p.model }

func (p *vertex) Complete(ctx context.Context, system, user string) (string, error) {
	in := map[string]any{
		"systemInstruction": map[string]any{"parts": []map[string]string{{"text": system}}},
		"contents":          []map[string]any{{"role": "user", "parts": []map[string]string{{"text": user}}}},
		"generationConfig":  map[string]any{"temperature": 0, "maxOutputTokens": 2048, "responseMimeType": "application/json"},
	}
	var out struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	url := fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:generateContent",
		p.location, p.project, p.location, p.model)
	if err := postJSON(ctx, p.client, url, map[string]string{"Authorization": "Bearer " + p.token}, in, &out); err != nil {
		return "", err
	}
	if len(out.Candidates) == 0 || len(out.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("llm returned no candidates")
	}
	return capText(out.Candidates[0].Content.Parts[0].Text)
}

// ollama calls a local Ollama /api/chat (no key).
type ollama struct {
	host, model string
	client      *http.Client
}

func (p *ollama) Name() string { return "ollama:" + p.model }

func (p *ollama) Complete(ctx context.Context, system, user string) (string, error) {
	in := map[string]any{
		"model":   p.model,
		"stream":  false,
		"format":  "json",
		"options": map[string]any{"temperature": 0},
		"messages": []map[string]string{
			{"role": "system", "content": system},
			{"role": "user", "content": user},
		},
	}
	var out struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	}
	if err := postJSON(ctx, p.client, p.host+"/api/chat", nil, in, &out); err != nil {
		return "", err
	}
	return capText(out.Message.Content)
}

// Fake is an in-process provider for tests and offline demos. It never makes
// network calls. Respond receives the prompts and returns the completion.
type Fake struct {
	Respond func(system, user string) (string, error)
	Calls   int
}

func (f *Fake) Name() string { return "fake" }

func (f *Fake) Complete(ctx context.Context, system, user string) (string, error) {
	f.Calls++
	if err := ctx.Err(); err != nil {
		return "", err
	}
	return f.Respond(system, user)
}

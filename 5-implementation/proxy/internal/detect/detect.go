// Package detect recognizes LLM API traffic by its shape, regardless of the
// destination host. It is deterministic: paths, headers and body structure.
package detect

import (
	"encoding/json"
	"net/http"
	"strings"
)

type Match struct {
	Format string // anthropic | openai | ollama | gemini | generic
	Signal string // what gave it away, for the audit log
}

var pathFormats = []struct{ suffix, format string }{
	{"/v1/messages", "anthropic"},
	{"/chat/completions", "openai"},
	{"/v1/completions", "openai"},
	{"/v1/responses", "openai"},
	{"/v1/embeddings", "openai"},
	{"/api/chat", "ollama"},
	{"/api/generate", "ollama"},
	{":generateContent", "gemini"},
	{":streamGenerateContent", "gemini"},
}

// LLM reports whether the request looks like a call to a language model API.
func LLM(r *http.Request, body []byte) (Match, bool) {
	if r.Header.Get("Anthropic-Version") != "" {
		return Match{"anthropic", "anthropic-version header"}, true
	}
	if r.Header.Get("Openai-Organization") != "" || r.Header.Get("Openai-Project") != "" {
		return Match{"openai", "openai-organization header"}, true
	}
	for _, p := range pathFormats {
		if strings.HasSuffix(r.URL.Path, p.suffix) && r.Method == http.MethodPost {
			return Match{p.format, "path " + p.suffix}, true
		}
	}
	if m, ok := bodyShape(body); ok {
		return m, true
	}
	return Match{}, false
}

// bodyShape matches the request schemas shared by nearly every LLM API:
// a model name plus a chat transcript, a prompt, or Gemini-style contents.
func bodyShape(body []byte) (Match, bool) {
	var b struct {
		Model    any               `json:"model"`
		Messages []json.RawMessage `json:"messages"`
		Prompt   any               `json:"prompt"`
		Contents []json.RawMessage `json:"contents"`
	}
	if len(body) == 0 || body[0] != '{' || json.Unmarshal(body, &b) != nil {
		return Match{}, false
	}
	if len(b.Messages) > 0 {
		var m struct {
			Role    string `json:"role"`
			Content any    `json:"content"`
		}
		if json.Unmarshal(b.Messages[0], &m) == nil && m.Role != "" && m.Content != nil {
			return Match{"generic", "chat transcript in body (messages[].role/content)"}, true
		}
	}
	if b.Model != nil && b.Prompt != nil {
		return Match{"generic", "model + prompt in body"}, true
	}
	if len(b.Contents) > 0 && strings.Contains(string(b.Contents[0]), `"parts"`) {
		return Match{"gemini", "contents[].parts in body"}, true
	}
	return Match{}, false
}

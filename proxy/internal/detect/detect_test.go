package detect

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLLM(t *testing.T) {
	cases := []struct {
		name, method, path, body string
		headers                  map[string]string
		want                     bool
	}{
		{"anthropic path", "POST", "/v1/messages", `{}`, nil, true},
		{"openai on a vps", "POST", "/v1/chat/completions", `{}`, nil, true},
		{"vllm custom prefix", "POST", "/llm/openai/v1/chat/completions", `{}`, nil, true},
		{"ollama", "POST", "/api/chat", `{}`, nil, true},
		{"anthropic header", "POST", "/proxy", `{}`, map[string]string{"anthropic-version": "2023-06-01"}, true},
		{"renamed path, chat body", "POST", "/q", `{"model":"x","messages":[{"role":"user","content":"hi"}]}`, nil, true},
		{"prompt body", "POST", "/gen", `{"model":"llama3","prompt":"hi"}`, nil, true},
		{"gemini body", "POST", "/x", `{"contents":[{"parts":[{"text":"hi"}]}]}`, nil, true},
		{"normal page", "GET", "/", ``, nil, false},
		{"normal json api", "POST", "/api/orders", `{"items":[1,2],"messages":["shipped"]}`, nil, false},
		{"form post", "POST", "/login", `user=a&pass=b`, nil, false},
		{"get on llm path", "GET", "/v1/messages", ``, nil, false},
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
		for k, v := range c.headers {
			r.Header.Set(k, v)
		}
		if _, got := LLM(r, []byte(c.body)); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}

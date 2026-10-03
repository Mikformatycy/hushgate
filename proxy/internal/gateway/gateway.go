// Package gateway is the reverse proxy between agents and the LLM provider.
package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

const maxBody = 64 << 20

type Gateway struct {
	Upstream    *url.URL
	UpstreamKey string // if set, replaces the agent's x-api-key so agents never hold the real key
	TokenLimit  int64  // per agent; 0 = unlimited
	Client      *http.Client
	Vault       *vault.Vault
	Policy      *policy.Policy
	Budget      budget.Store
	Audit       audit.Logger
}

var hopHeaders = map[string]bool{
	"Connection": true, "Keep-Alive": true, "Proxy-Connection": true, "Te": true, "Trailer": true,
	"Transfer-Encoding": true, "Upgrade": true, "Host": true, "Content-Length": true,
	"Accept-Encoding": true, "X-Agent-Id": true, "Proxy-Authorization": true,
}

func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.Write([]byte("ok"))
		return
	}
	ctx := r.Context()
	agent := r.Header.Get("X-Agent-Id")
	if agent == "" {
		agent = "default"
	}

	st, err := g.Budget.Status(ctx, agent)
	if err != nil {
		log.Printf("budget status: %v", err)
		apiError(w, http.StatusServiceUnavailable, "api_error", "provenance gate: budget store unavailable")
		return
	}
	if st.Killed {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "denied", Reason: "killed: " + st.KillReason})
		apiError(w, http.StatusForbidden, "permission_error", "provenance gate: agent halted: "+st.KillReason)
		return
	}
	if g.TokenLimit > 0 && st.Used >= g.TokenLimit {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "denied", Reason: "token budget exhausted", Usage: st.Used})
		apiError(w, http.StatusForbidden, "permission_error",
			fmt.Sprintf("provenance gate: token budget exhausted (%d/%d)", st.Used, g.TokenLimit))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid_request_error", "provenance gate: cannot read body")
		return
	}
	body, refs := g.Vault.Mask(body)
	if len(refs) > 0 {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "mask", Tokens: tokenNames(refs)})
	}

	resp, err := g.forward(ctx, r, body)
	if err != nil {
		log.Printf("upstream: %v", err)
		apiError(w, http.StatusBadGateway, "api_error", "provenance gate: upstream unreachable")
		return
	}
	defer resp.Body.Close()

	for k, vs := range resp.Header {
		if !hopHeaders[k] && k != "Content-Encoding" {
			w.Header()[k] = vs
		}
	}

	isMessages := r.Method == http.MethodPost && r.URL.Path == "/v1/messages"
	ct := resp.Header.Get("Content-Type")
	switch {
	case isMessages && resp.StatusCode == http.StatusOK && strings.HasPrefix(ct, "text/event-stream"):
		w.WriteHeader(resp.StatusCode)
		s := &streamFilter{g: g, ctx: ctx, agent: agent, w: w, tools: map[int]*toolBuf{}}
		if f, ok := w.(http.Flusher); ok {
			s.flush = f.Flush
		}
		if err := s.run(resp.Body); err != nil {
			log.Printf("stream: %v", err)
		}
		g.recordUsage(ctx, agent, s.usage)
	case isMessages && resp.StatusCode == http.StatusOK && strings.HasPrefix(ct, "application/json"):
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			apiError(w, http.StatusBadGateway, "api_error", "provenance gate: upstream read failed")
			return
		}
		out, usage := g.filterMessage(ctx, agent, raw)
		w.WriteHeader(resp.StatusCode)
		w.Write(out)
		g.recordUsage(ctx, agent, usage)
	default:
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
	}
}

func (g *Gateway) forward(ctx context.Context, r *http.Request, body []byte) (*http.Response, error) {
	u := *g.Upstream
	u.Path = strings.TrimSuffix(u.Path, "/") + r.URL.Path
	u.RawQuery = r.URL.RawQuery
	req, err := http.NewRequestWithContext(ctx, r.Method, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	for k, vs := range r.Header {
		if !hopHeaders[k] {
			req.Header[k] = vs
		}
	}
	// identity keeps SSE readable chunk-by-chunk
	req.Header.Set("Accept-Encoding", "identity")
	if g.UpstreamKey != "" {
		req.Header.Set("X-Api-Key", g.UpstreamKey)
		req.Header.Del("Authorization")
	}
	return g.Client.Do(req)
}

// decideTool applies policy to one complete tool call. It returns the action
// and the (possibly rehydrated) JSON input to hand to the agent.
func (g *Gateway) decideTool(ctx context.Context, agent, id, name, input string) (policy.Action, string, string) {
	refs := g.Vault.Tokens(input)
	act, rehydrate, reason := g.Policy.Decide(name, refs)
	if act == policy.Kill {
		if err := g.Budget.Kill(ctx, agent, fmt.Sprintf("%s: %s", name, reason)); err != nil {
			log.Printf("kill %s: %v", agent, err)
		}
	}
	g.Audit.Record(audit.Event{Agent: agent, Kind: "tool_call", Tool: name, ToolID: id,
		Action: string(act), Reason: reason, Tokens: tokenNames(refs)})
	if act == policy.Allow && rehydrate {
		input = g.Vault.RehydrateJSON(input)
	}
	return act, input, reason
}

// filterMessage handles non-streaming /v1/messages responses.
func (g *Gateway) filterMessage(ctx context.Context, agent string, raw []byte) ([]byte, int64) {
	var msg map[string]any
	if err := json.Unmarshal(raw, &msg); err != nil {
		return raw, 0
	}
	content, _ := msg["content"].([]any)
	kept, removed := 0, 0
	for i, c := range content {
		block, _ := c.(map[string]any)
		if block["type"] != "tool_use" {
			continue
		}
		name, _ := block["name"].(string)
		id, _ := block["id"].(string)
		in, _ := json.Marshal(block["input"])
		act, out, reason := g.decideTool(ctx, agent, id, name, string(in))
		if act != policy.Allow {
			content[i] = map[string]any{"type": "text", "text": blockedText(name, act, reason)}
			removed++
			continue
		}
		var input any
		if json.Unmarshal([]byte(out), &input) == nil {
			block["input"] = input
		}
		kept++
	}
	if removed > 0 && kept == 0 && msg["stop_reason"] == "tool_use" {
		msg["stop_reason"] = "end_turn"
	}
	var usage int64
	if u, ok := msg["usage"].(map[string]any); ok {
		usage = sumUsage(u)
	}
	out, err := json.Marshal(msg)
	if err != nil {
		return raw, usage
	}
	return out, usage
}

func (g *Gateway) recordUsage(ctx context.Context, agent string, tokens int64) {
	if tokens == 0 {
		return
	}
	total, err := g.Budget.AddUsage(ctx, agent, tokens)
	if err != nil {
		log.Printf("add usage: %v", err)
		return
	}
	g.Audit.Record(audit.Event{Agent: agent, Kind: "usage", Usage: total})
}

func sumUsage(u map[string]any) int64 {
	var n int64
	for _, k := range []string{"input_tokens", "output_tokens", "cache_creation_input_tokens", "cache_read_input_tokens"} {
		if f, ok := u[k].(float64); ok {
			n += int64(f)
		}
	}
	return n
}

func blockedText(tool string, act policy.Action, reason string) string {
	return fmt.Sprintf("[Provenance Gate] %s call to tool %q: %s", strings.ToUpper(string(act)), tool, reason)
}

func tokenNames(refs []vault.Ref) []string {
	out := make([]string, 0, len(refs))
	for _, r := range refs {
		out = append(out, r.Token)
	}
	return out
}

func apiError(w http.ResponseWriter, status int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"type":  "error",
		"error": map[string]string{"type": typ, "message": msg},
	})
}

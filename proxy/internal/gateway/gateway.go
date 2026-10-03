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
	"time"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
	"github.com/Mikformatycy/hushgate/proxy/internal/budget"
	"github.com/Mikformatycy/hushgate/proxy/internal/metrics"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/review"
	"github.com/Mikformatycy/hushgate/proxy/internal/scan"
	"github.com/Mikformatycy/hushgate/proxy/internal/shell"
	"github.com/Mikformatycy/hushgate/proxy/internal/signature"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

const maxBody = 64 << 20

type Gateway struct {
	Upstream    *url.URL
	UpstreamKey string                   // if set, replaces the agent's x-api-key so agents never hold the real key
	TokenLimit  int64                    // per agent; 0 = unlimited (used when Limits is nil)
	Limits      func(agent string) int64 // optional per-agent budgets from the live policy
	ModelOK     func(model string) bool  // optional allowed-models check from the live policy
	Client      *http.Client
	Vault       *vault.Vault
	Policy      *policy.Policy
	Budget      budget.Store
	Audit       audit.Logger
	Reviews     *review.Store // optional: queue tools that have no policy rule
	Scans       *scan.Store   // optional: queue tool results for injection scanning
	Guard       *shell.Guard  // optional: shell calls that reach the network count as network tools
	Signatures  *signature.Set
	Metrics     *metrics.Metrics // optional; nil-safe
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
	start := time.Now()
	sw := &statusWriter{ResponseWriter: w, code: http.StatusOK}
	w = sw
	defer func() { g.Metrics.Request(route(r.URL.Path), sw.code, time.Since(start)) }()

	ctx := r.Context()
	agent := r.Header.Get("X-Agent-Id")
	if agent == "" {
		agent = "default"
	}

	st, err := g.Budget.Status(ctx, agent)
	if err != nil {
		log.Printf("budget status: %v", err)
		apiError(w, http.StatusServiceUnavailable, "api_error", "hushgate: budget store unavailable")
		return
	}
	if st.Killed {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "denied", Reason: "killed: " + st.KillReason})
		apiError(w, http.StatusForbidden, "permission_error", "hushgate: agent halted: "+st.KillReason)
		return
	}
	limit := g.TokenLimit
	if g.Limits != nil {
		limit = g.Limits(agent)
	}
	if limit > 0 && st.Used >= limit {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "denied", Reason: "token budget exhausted", Usage: st.Used})
		apiError(w, http.StatusForbidden, "permission_error",
			fmt.Sprintf("hushgate: token budget exhausted (%d/%d)", st.Used, limit))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, maxBody))
	if err != nil {
		apiError(w, http.StatusBadRequest, "invalid_request_error", "hushgate: cannot read body")
		return
	}
	if g.ModelOK != nil && strings.HasPrefix(r.URL.Path, "/v1/messages") {
		var req struct {
			Model string `json:"model"`
		}
		if json.Unmarshal(body, &req) == nil && req.Model != "" && !g.ModelOK(req.Model) {
			g.Audit.Record(audit.Event{Agent: agent, Kind: "model_blocked", Tool: req.Model,
				Reason: "model " + req.Model + " is not in models.allow"})
			apiError(w, http.StatusForbidden, "permission_error",
				"hushgate: model "+req.Model+" is not allowed by policy")
			return
		}
	}
	body, refs := g.Vault.Mask(body)
	if len(refs) > 0 {
		g.Audit.Record(audit.Event{Agent: agent, Kind: "mask", Tokens: tokenNames(refs)})
		for _, ref := range refs {
			g.Metrics.Masked(ref.Tier.String(), 1)
		}
	}
	if r.URL.Path == "/v1/messages" {
		g.discoverTools(agent, body)
		g.queueToolResults(agent, body)
	}

	g.Metrics.Observe(metrics.StagePreprocess, time.Since(start))
	upstreamStart := time.Now()
	resp, err := g.forward(ctx, r, body)
	g.Metrics.Observe(metrics.StageUpstreamTTFB, time.Since(upstreamStart))
	if err != nil {
		log.Printf("upstream: %v", err)
		apiError(w, http.StatusBadGateway, "api_error", "hushgate: upstream unreachable")
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
			apiError(w, http.StatusBadGateway, "api_error", "hushgate: upstream read failed")
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

// discoverTools queues every tool definition the policy has no rule for. The
// body is already masked, so descriptions never carry vaulted values.
func (g *Gateway) discoverTools(agent string, body []byte) {
	if g.Reviews == nil {
		return
	}
	var req struct {
		Tools []struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			InputSchema any    `json:"input_schema"`
		} `json:"tools"`
	}
	if json.Unmarshal(body, &req) != nil {
		return
	}
	for _, t := range req.Tools {
		if t.Name != "" && !g.Policy.Has(t.Name) {
			g.Reviews.ObserveTool(t.Name, t.Description, t.InputSchema,
				string(g.Policy.SinkFor(t.Name))+" (policy default)", agent)
		}
	}
}

// queueToolResults sends what the agent just received from its tools (the
// last user message) to the injection scan queue, labelled with the call that
// produced it. The body is already masked.
func (g *Gateway) queueToolResults(agent string, body []byte) {
	if g.Scans == nil {
		return
	}
	var req struct {
		Messages []struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &req) != nil || len(req.Messages) == 0 {
		return
	}
	type block struct {
		Type      string          `json:"type"`
		ID        string          `json:"id"`
		Name      string          `json:"name"`
		Input     json.RawMessage `json:"input"`
		ToolUseID string          `json:"tool_use_id"`
		Content   json.RawMessage `json:"content"`
		Text      string          `json:"text"`
	}
	calls := map[string]string{}
	for _, m := range req.Messages {
		var blocks []block
		if json.Unmarshal(m.Content, &blocks) != nil {
			continue
		}
		for _, b := range blocks {
			if b.Type == "tool_use" {
				src := b.Name + " " + string(b.Input)
				if len(src) > 120 {
					src = src[:120] + "…"
				}
				calls[b.ID] = src
			}
		}
	}
	last := req.Messages[len(req.Messages)-1]
	var blocks []block
	if last.Role != "user" || json.Unmarshal(last.Content, &blocks) != nil {
		return
	}
	for _, b := range blocks {
		if b.Type != "tool_result" {
			continue
		}
		text := blockText(b.Content)
		if len(strings.TrimSpace(text)) < 20 {
			continue
		}
		src, ok := calls[b.ToolUseID]
		if !ok {
			src = "tool result"
		}
		g.Scans.Observe(agent, src, text)
	}
}

// blockText flattens tool_result content: a string or a list of text blocks.
func blockText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var parts []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	json.Unmarshal(raw, &parts)
	var b strings.Builder
	for _, p := range parts {
		if p.Type == "text" {
			b.WriteString(p.Text)
			b.WriteString("\n")
		}
	}
	return b.String()
}

// decideTool applies policy to one complete tool call. It returns the action
// and the (possibly rehydrated) JSON input to hand to the agent.
func (g *Gateway) decideTool(ctx context.Context, agent, id, name, input string) (policy.Action, string, string) {
	defer func(start time.Time) { g.Metrics.Observe(metrics.StageToolDecision, time.Since(start)) }(time.Now())
	refs := g.Vault.Tokens(input)
	sink := g.Policy.SinkFor(name)
	var guardNote, shellText string
	if g.Guard != nil && g.Guard.IsShellTool(name) {
		cmd := commandOf(input)
		shellText = shell.Normalize(cmd)
		if prog, ok := g.Guard.NetworkCommand(cmd); ok && sink == policy.Local {
			sink = policy.Network
			guardNote = "Bash guard: command runs " + prog + ", treated as a network tool; "
		}
	}
	act, rehydrate, reason := g.Policy.DecideAs(sink, refs)
	reason = guardNote + reason

	// Known attack signatures can only make the decision stricter.
	if g.Signatures != nil {
		for _, h := range g.Signatures.Match(name, signature.Flatten(input), shellText) {
			sig := h.Signature
			g.Audit.Record(audit.Event{Agent: agent, Kind: "signature", Tool: name, ToolID: id, Action: sig.Action,
				Reason: fmt.Sprintf("%s %s (%s, %s): matched %q", sig.ID, sig.Name, sig.Category, sig.Severity, h.Match)})
			if sig.Action != "alert" && signature.Rank(sig.Action) > signature.Rank(string(act)) {
				act, rehydrate = policy.Action(sig.Action), false
				reason = fmt.Sprintf("signature %s: %s", sig.ID, sig.Name)
			}
		}
	}
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

// statusWriter records the status code for metrics and keeps Flush working,
// which the SSE stream filter needs.
type statusWriter struct {
	http.ResponseWriter
	code int
}

func (s *statusWriter) WriteHeader(code int) {
	s.code = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func route(path string) string {
	switch path {
	case "/v1/messages":
		return "messages"
	case "/v1/messages/count_tokens":
		return "count_tokens"
	}
	return "other"
}

// commandOf returns the shell command of a shell tool call ("command" field,
// as Claude Code's Bash tool sends it), or all its text otherwise.
func commandOf(input string) string {
	var in struct {
		Command string `json:"command"`
	}
	if json.Unmarshal([]byte(input), &in) == nil && in.Command != "" {
		return in.Command
	}
	return signature.Flatten(input)
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
	g.Metrics.Tokens(agent, tokens)
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
	return fmt.Sprintf("[HushGate] %s call to tool %q: %s", strings.ToUpper(string(act)), tool, reason)
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

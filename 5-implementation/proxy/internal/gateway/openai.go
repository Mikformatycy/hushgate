package gateway

// Chat Completions support: the OpenAI-compatible API spoken by OpenAI,
// OpenRouter, most hosted and self-hosted models and many agent harnesses.
// Requests get the same masking, tool policy, Bash guard, signatures and
// budgets as Anthropic Messages requests; only the wire format differs.

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
)

func isChat(path string) bool { return strings.HasSuffix(path, "/chat/completions") }

// discoverChatTools queues every tool definition the policy has no rule for.
func (g *Gateway) discoverChatTools(agent string, body []byte) {
	if g.Reviews == nil {
		return
	}
	var req struct {
		Tools []struct {
			Function struct {
				Name        string `json:"name"`
				Description string `json:"description"`
				Parameters  any    `json:"parameters"`
			} `json:"function"`
		} `json:"tools"`
	}
	if json.Unmarshal(body, &req) != nil {
		return
	}
	for _, t := range req.Tools {
		if n := t.Function.Name; n != "" && !g.Policy.Has(n) {
			g.Reviews.ObserveTool(n, t.Function.Description, t.Function.Parameters,
				string(g.Policy.SinkFor(n))+" (policy default)", agent)
		}
	}
}

// queueChatToolResults sends the tool results that arrived since the model's
// last turn (role "tool" messages) to the injection scan queue.
func (g *Gateway) queueChatToolResults(agent string, body []byte) {
	if g.Scans == nil {
		return
	}
	var req struct {
		Messages []struct {
			Role       string          `json:"role"`
			Content    json.RawMessage `json:"content"`
			ToolCallID string          `json:"tool_call_id"`
			ToolCalls  []struct {
				ID       string `json:"id"`
				Function struct {
					Name      string `json:"name"`
					Arguments string `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &req) != nil {
		return
	}
	calls := map[string]string{}
	start := 0
	for i, m := range req.Messages {
		if m.Role != "assistant" {
			continue
		}
		start = i + 1
		for _, tc := range m.ToolCalls {
			src := tc.Function.Name + " " + tc.Function.Arguments
			if len(src) > 120 {
				src = src[:120] + "…"
			}
			calls[tc.ID] = src
		}
	}
	for _, m := range req.Messages[start:] {
		if m.Role != "tool" {
			continue
		}
		text := blockText(m.Content)
		if len(strings.TrimSpace(text)) < 20 {
			continue
		}
		src, ok := calls[m.ToolCallID]
		if !ok {
			src = "tool result"
		}
		g.Scans.Observe(agent, src, text)
	}
}

// withUsage makes streamed answers report token usage, so budgets count them
// whatever the agent asked for.
func withUsage(body []byte) []byte {
	var req map[string]json.RawMessage
	if json.Unmarshal(body, &req) != nil {
		return body
	}
	var stream bool
	if json.Unmarshal(req["stream"], &stream) != nil || !stream {
		return body
	}
	opts := map[string]any{}
	json.Unmarshal(req["stream_options"], &opts)
	if opts["include_usage"] == true {
		return body
	}
	opts["include_usage"] = true
	req["stream_options"], _ = json.Marshal(opts)
	out, err := json.Marshal(req)
	if err != nil {
		return body
	}
	return out
}

func chatUsage(u map[string]any) int64 {
	if t, ok := u["total_tokens"].(float64); ok && t > 0 {
		return int64(t)
	}
	var n int64
	for _, k := range []string{"prompt_tokens", "completion_tokens"} {
		if f, ok := u[k].(float64); ok {
			n += int64(f)
		}
	}
	return n
}

// chatStream rewrites a Chat Completions SSE stream. Text passes through as it
// arrives. Tool-call fragments are held per choice until the chunk that
// finishes the choice; then each complete call is checked, and allowed calls
// are sent whole while blocked ones become a text note.
type chatStream struct {
	g     *Gateway
	ctx   context.Context
	agent string
	w     io.Writer
	flush func()

	calls    map[int][]*chatCall // held calls per choice, in arrival order
	envelope map[string]any      // id, model, created... copied into synthesized chunks
	usage    int64
}

type chatCall struct {
	index    int
	hasIndex bool
	id, name string
	args     strings.Builder
}

func (s *chatStream) run(body io.Reader) error {
	br := bufio.NewReader(body)
	var data []string
	for {
		line, err := br.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		switch {
		case trimmed == "" && len(data) > 0:
			if werr := s.handle(strings.Join(data, "\n")); werr != nil {
				return werr
			}
			data = nil
		case strings.HasPrefix(trimmed, ":"): // keep-alive comments, e.g. ": OPENROUTER PROCESSING"
			if werr := s.write(trimmed + "\n\n"); werr != nil {
				return werr
			}
		case strings.HasPrefix(trimmed, "data:"):
			data = append(data, strings.TrimPrefix(trimmed[len("data:"):], " "))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(data) > 0 {
					if werr := s.handle(strings.Join(data, "\n")); werr != nil {
						return werr
					}
				}
				return s.releaseAll() // stream ended without [DONE]
			}
			return err
		}
	}
}

func (s *chatStream) handle(data string) error {
	if strings.TrimSpace(data) == "[DONE]" {
		if err := s.releaseAll(); err != nil {
			return err
		}
		return s.emit(data)
	}
	var chunk map[string]any
	if json.Unmarshal([]byte(data), &chunk) != nil {
		return s.emit(data)
	}
	s.remember(chunk)
	if u, ok := chunk["usage"].(map[string]any); ok {
		s.usage = max(s.usage, chatUsage(u))
	}
	choices, _ := chunk["choices"].([]any)
	var keep []any
	var finishing []int
	for _, c := range choices {
		ch, ok := c.(map[string]any)
		if !ok {
			keep = append(keep, c)
			continue
		}
		idx := intOf(ch["index"])
		delta, _ := ch["delta"].(map[string]any)
		if tcs, ok := delta["tool_calls"].([]any); ok {
			for _, t := range tcs {
				s.collect(idx, t)
			}
			delete(delta, "tool_calls")
		}
		reason, _ := ch["finish_reason"].(string)
		if reason != "" && len(s.calls[idx]) > 0 {
			finishing = append(finishing, idx)
		}
		if reason == "" && emptyDelta(delta) {
			continue // only tool-call fragments: held until complete
		}
		keep = append(keep, ch)
	}
	// Release a choice's calls before the chunk that finishes it.
	for _, idx := range finishing {
		removedAll, err := s.release(idx)
		if err != nil {
			return err
		}
		if removedAll {
			for _, c := range keep {
				if ch, ok := c.(map[string]any); ok && intOf(ch["index"]) == idx && ch["finish_reason"] == "tool_calls" {
					ch["finish_reason"] = "stop"
				}
			}
		}
	}
	if len(choices) > 0 && len(keep) == 0 && chunk["usage"] == nil {
		return nil
	}
	if keep == nil {
		keep = []any{}
	}
	chunk["choices"] = keep
	out, err := json.Marshal(chunk)
	if err != nil {
		return s.emit(data)
	}
	return s.emit(string(out))
}

// collect adds one tool-call fragment. Fragments carry an index; providers
// that omit it send each call whole, told apart by its id.
func (s *chatStream) collect(choice int, raw any) {
	tc, _ := raw.(map[string]any)
	if tc == nil {
		return
	}
	id, _ := tc["id"].(string)
	_, hasIndex := tc["index"]
	index := intOf(tc["index"])
	var c *chatCall
	for _, x := range s.calls[choice] {
		if (hasIndex && x.hasIndex && x.index == index) || (!hasIndex && id != "" && x.id == id) {
			c = x
		}
	}
	if c == nil && !hasIndex && id == "" && len(s.calls[choice]) > 0 {
		c = s.calls[choice][len(s.calls[choice])-1]
	}
	if c == nil {
		c = &chatCall{index: index, hasIndex: hasIndex}
		s.calls[choice] = append(s.calls[choice], c)
	}
	if id != "" {
		c.id = id
	}
	if fn, ok := tc["function"].(map[string]any); ok {
		if n, _ := fn["name"].(string); n != "" && c.name == "" {
			c.name = n
		}
		if a, _ := fn["arguments"].(string); a != "" {
			c.args.WriteString(a)
		}
	}
}

// release decides each held call of one choice and sends the result. It
// reports whether every call was removed.
func (s *chatStream) release(choice int) (bool, error) {
	calls := s.calls[choice]
	delete(s.calls, choice)
	kept, notes := s.g.decideChatCalls(s.ctx, s.agent, calls)
	if len(notes) > 0 {
		if err := s.synth(choice, map[string]any{"content": strings.Join(notes, "\n")}); err != nil {
			return false, err
		}
	}
	if len(kept) > 0 {
		if err := s.synth(choice, map[string]any{"tool_calls": kept}); err != nil {
			return false, err
		}
	}
	return len(kept) == 0 && len(calls) > 0, nil
}

func (s *chatStream) releaseAll() error {
	var idx []int
	for i := range s.calls {
		idx = append(idx, i)
	}
	sort.Ints(idx)
	for _, i := range idx {
		if _, err := s.release(i); err != nil {
			return err
		}
	}
	return nil
}

// decideChatCalls applies the policy to complete calls: allowed ones come
// back in wire form (with real values restored where the policy allows),
// blocked ones as notes for the model and the user.
func (g *Gateway) decideChatCalls(ctx context.Context, agent string, calls []*chatCall) ([]any, []string) {
	var kept []any
	var notes []string
	for _, c := range calls {
		args := c.args.String()
		if strings.TrimSpace(args) == "" {
			args = "{}"
		}
		act, out, reason := g.decideTool(ctx, agent, c.id, c.name, args)
		if act != policy.Allow {
			notes = append(notes, blockedText(c.name, act, reason))
			continue
		}
		kept = append(kept, map[string]any{"index": len(kept), "id": c.id, "type": "function",
			"function": map[string]any{"name": c.name, "arguments": out}})
	}
	return kept, notes
}

func (s *chatStream) remember(chunk map[string]any) {
	if s.envelope == nil {
		s.envelope = map[string]any{}
	}
	for _, k := range []string{"id", "object", "created", "model", "system_fingerprint", "provider"} {
		if v, ok := chunk[k]; ok {
			s.envelope[k] = v
		}
	}
}

func (s *chatStream) synth(choice int, delta map[string]any) error {
	c := map[string]any{}
	for k, v := range s.envelope {
		c[k] = v
	}
	c["choices"] = []any{map[string]any{"index": choice, "delta": delta, "finish_reason": nil}}
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	return s.emit(string(b))
}

func (s *chatStream) emit(data string) error {
	var b strings.Builder
	for _, line := range strings.Split(data, "\n") {
		b.WriteString("data: " + line + "\n")
	}
	b.WriteString("\n")
	return s.write(b.String())
}

func (s *chatStream) write(text string) error {
	if _, err := io.WriteString(s.w, text); err != nil {
		return err
	}
	if s.flush != nil {
		s.flush()
	}
	return nil
}

// filterChat handles non-streaming Chat Completions responses.
func (g *Gateway) filterChat(ctx context.Context, agent string, raw []byte) ([]byte, int64) {
	var resp map[string]any
	if json.Unmarshal(raw, &resp) != nil {
		return raw, 0
	}
	var usage int64
	if u, ok := resp["usage"].(map[string]any); ok {
		usage = chatUsage(u)
	}
	choices, _ := resp["choices"].([]any)
	for _, c := range choices {
		ch, _ := c.(map[string]any)
		msg, _ := ch["message"].(map[string]any)
		tcs, _ := msg["tool_calls"].([]any)
		if len(tcs) == 0 {
			continue
		}
		var calls []*chatCall
		for i, t := range tcs {
			tc, _ := t.(map[string]any)
			fn, _ := tc["function"].(map[string]any)
			call := &chatCall{index: i, hasIndex: true}
			call.id, _ = tc["id"].(string)
			call.name, _ = fn["name"].(string)
			args, _ := fn["arguments"].(string)
			call.args.WriteString(args)
			calls = append(calls, call)
		}
		kept, notes := g.decideChatCalls(ctx, agent, calls)
		if len(notes) > 0 {
			text, _ := msg["content"].(string)
			msg["content"] = strings.TrimSpace(text + "\n" + strings.Join(notes, "\n"))
		}
		if len(kept) == 0 {
			delete(msg, "tool_calls")
			if ch["finish_reason"] == "tool_calls" {
				ch["finish_reason"] = "stop"
			}
		} else {
			for _, k := range kept { // non-streamed calls carry no index
				delete(k.(map[string]any), "index")
			}
			msg["tool_calls"] = kept
		}
	}
	out, err := json.Marshal(resp)
	if err != nil {
		return raw, usage
	}
	return out, usage
}

// chatError answers in the OpenAI error format, which OpenAI-compatible
// clients parse and show.
func chatError(w http.ResponseWriter, status int, typ, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]any{
		"error": map[string]any{"message": msg, "type": typ, "code": nil},
	})
}

func intOf(v any) int {
	if f, ok := v.(float64); ok {
		return int(f)
	}
	return 0
}

// emptyDelta reports whether a delta has nothing for the agent: no role and no
// non-empty value.
func emptyDelta(d map[string]any) bool {
	if _, ok := d["role"]; ok {
		return false
	}
	for _, v := range d {
		if v != nil && v != "" {
			return false
		}
	}
	return true
}

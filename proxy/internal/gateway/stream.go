package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"

	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
)

// streamFilter rewrites an Anthropic SSE stream. Text passes through as it
// arrives; each tool_use block is held until content_block_stop so the policy
// sees the complete arguments before the agent does.
type streamFilter struct {
	g     *Gateway
	ctx   context.Context
	agent string
	w     io.Writer
	flush func()

	tools         map[int]*toolBuf
	kept, removed int
	usage         int64
}

type toolBuf struct {
	id, name string
	start    string // raw content_block_start data
	input    strings.Builder
}

type sseEvent struct {
	Type         string          `json:"type"`
	Index        int             `json:"index"`
	ContentBlock json.RawMessage `json:"content_block"`
	Delta        json.RawMessage `json:"delta"`
	Message      *struct {
		Usage map[string]any `json:"usage"`
	} `json:"message"`
	Usage map[string]any `json:"usage"`
}

func (s *streamFilter) run(body io.Reader) error {
	br := bufio.NewReader(body)
	var name string
	var data []string
	for {
		line, err := br.ReadString('\n')
		trimmed := strings.TrimRight(line, "\r\n")
		switch {
		case trimmed == "" && (len(data) > 0 || name != ""):
			if werr := s.handle(name, strings.Join(data, "\n")); werr != nil {
				return werr
			}
			name, data = "", nil
		case strings.HasPrefix(trimmed, "event:"):
			name = strings.TrimSpace(trimmed[len("event:"):])
		case strings.HasPrefix(trimmed, "data:"):
			data = append(data, strings.TrimPrefix(trimmed[len("data:"):], " "))
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				if len(data) > 0 {
					return s.handle(name, strings.Join(data, "\n"))
				}
				return nil
			}
			return err
		}
	}
}

func (s *streamFilter) handle(name, data string) error {
	var ev sseEvent
	if err := json.Unmarshal([]byte(data), &ev); err != nil {
		return s.emit(name, data)
	}
	switch ev.Type {
	case "message_start":
		if ev.Message != nil {
			s.usage += sumUsage(ev.Message.Usage)
		}
	case "content_block_start":
		var cb struct{ Type, ID, Name string }
		json.Unmarshal(ev.ContentBlock, &cb)
		if cb.Type == "tool_use" {
			s.tools[ev.Index] = &toolBuf{id: cb.ID, name: cb.Name, start: data}
			return nil
		}
	case "content_block_delta":
		if tb, ok := s.tools[ev.Index]; ok {
			var d struct {
				PartialJSON string `json:"partial_json"`
			}
			json.Unmarshal(ev.Delta, &d)
			tb.input.WriteString(d.PartialJSON)
			return nil
		}
	case "content_block_stop":
		if tb, ok := s.tools[ev.Index]; ok {
			delete(s.tools, ev.Index)
			return s.releaseTool(ev.Index, tb, data)
		}
	case "message_delta":
		if ev.Usage != nil {
			// message_delta carries the cumulative output count
			if f, ok := ev.Usage["output_tokens"].(float64); ok {
				s.usage += int64(f)
			}
		}
		if s.removed > 0 && s.kept == 0 {
			data = rewriteStopReason(data)
		}
	}
	return s.emit(name, data)
}

func (s *streamFilter) releaseTool(index int, tb *toolBuf, stopData string) error {
	input := tb.input.String()
	if strings.TrimSpace(input) == "" {
		input = "{}"
	}
	act, out, reason := s.g.decideTool(s.ctx, s.agent, tb.id, tb.name, input)
	if act == policy.Allow {
		s.kept++
		delta, _ := json.Marshal(map[string]any{
			"type": "content_block_delta", "index": index,
			"delta": map[string]string{"type": "input_json_delta", "partial_json": out},
		})
		return s.emitAll(
			[2]string{"content_block_start", tb.start},
			[2]string{"content_block_delta", string(delta)},
			[2]string{"content_block_stop", stopData},
		)
	}

	// Replace the call with a text block at the same index so the stream stays well-formed.
	s.removed++
	start, _ := json.Marshal(map[string]any{
		"type": "content_block_start", "index": index,
		"content_block": map[string]string{"type": "text", "text": ""},
	})
	delta, _ := json.Marshal(map[string]any{
		"type": "content_block_delta", "index": index,
		"delta": map[string]string{"type": "text_delta", "text": blockedText(tb.name, act, reason)},
	})
	return s.emitAll(
		[2]string{"content_block_start", string(start)},
		[2]string{"content_block_delta", string(delta)},
		[2]string{"content_block_stop", stopData},
	)
}

func rewriteStopReason(data string) string {
	var m map[string]any
	if json.Unmarshal([]byte(data), &m) != nil {
		return data
	}
	if d, ok := m["delta"].(map[string]any); ok && d["stop_reason"] == "tool_use" {
		d["stop_reason"] = "end_turn"
		if b, err := json.Marshal(m); err == nil {
			return string(b)
		}
	}
	return data
}

func (s *streamFilter) emitAll(events ...[2]string) error {
	for _, e := range events {
		if err := s.emit(e[0], e[1]); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamFilter) emit(name, data string) error {
	var b strings.Builder
	if name != "" {
		b.WriteString("event: " + name + "\n")
	}
	for _, line := range strings.Split(data, "\n") {
		b.WriteString("data: " + line + "\n")
	}
	b.WriteString("\n")
	if _, err := io.WriteString(s.w, b.String()); err != nil {
		return err
	}
	if s.flush != nil {
		s.flush()
	}
	return nil
}

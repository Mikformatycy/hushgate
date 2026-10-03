// Package audit records everything the gate decides. Postgres sink comes later;
// for now events go to stdout as JSON lines.
package audit

import (
	"encoding/json"
	"io"
	"sync"
	"time"
)

type Event struct {
	Time   time.Time `json:"time"`
	Agent  string    `json:"agent"`
	Kind   string    `json:"kind"` // request | mask | tool_call | usage | denied
	Tool   string    `json:"tool,omitempty"`
	ToolID string    `json:"tool_id,omitempty"`
	Action string    `json:"action,omitempty"`
	Reason string    `json:"reason,omitempty"`
	Tokens []string  `json:"tokens,omitempty"` // vault placeholders involved, never values
	Usage  int64     `json:"usage,omitempty"`
}

type Logger interface {
	Record(Event)
}

type JSONLogger struct {
	mu sync.Mutex
	w  io.Writer
}

func NewJSONLogger(w io.Writer) *JSONLogger { return &JSONLogger{w: w} }

func (l *JSONLogger) Record(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	b, _ := json.Marshal(e)
	l.mu.Lock()
	defer l.mu.Unlock()
	l.w.Write(append(b, '\n'))
}

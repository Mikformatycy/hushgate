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
	ID     int64     `json:"id"`
	Time   time.Time `json:"time"`
	Agent  string    `json:"agent"`
	Kind   string    `json:"kind"` // mask | tool_call | usage | denied | reset | shadow_ai | node_blocked
	Host   string    `json:"host,omitempty"`
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

// Multi fans an event out to several loggers.
type Multi []Logger

func (m Multi) Record(e Event) {
	if e.Time.IsZero() {
		e.Time = time.Now()
	}
	for _, l := range m {
		l.Record(e)
	}
}

// Ring keeps the most recent events in memory for the dashboard.
type Ring struct {
	mu     sync.Mutex
	size   int
	nextID int64
	events []Event
}

func NewRing(size int) *Ring { return &Ring{size: size} }

func (r *Ring) Record(e Event) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.nextID++
	e.ID = r.nextID
	r.events = append(r.events, e)
	if len(r.events) > r.size {
		r.events = r.events[len(r.events)-r.size:]
	}
}

// After returns events with ID greater than after, oldest first.
func (r *Ring) After(after int64) []Event {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := []Event{}
	for _, e := range r.events {
		if e.ID > after {
			out = append(out, e)
		}
	}
	return out
}

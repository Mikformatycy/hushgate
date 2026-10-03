// Package scan queues content that agents read (tool results) for the AI
// advisor to check for prompt injection. It is an early-warning signal only:
// nothing here blocks a request; enforcement stays with the deterministic rules.
package scan

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// MaxText bounds what is sent for classification (Jev allows 32k tokens).
const MaxText = 60_000

type Item struct {
	ID         string    `json:"id"`
	Agent      string    `json:"agent"`
	Source     string    `json:"source"` // e.g. read_file {"path":"quarterly_report.md"}
	Text       string    `json:"text"`   // already masked by the vault
	ObservedAt time.Time `json:"observed_at"`
	Status     string    `json:"status"` // pending | done
	Injection  *float64  `json:"injection,omitempty"`
	Model      string    `json:"model,omitempty"`
}

type Store struct {
	mu    sync.Mutex
	items map[string]*Item
	order []string
	max   int
}

func NewStore(max int) *Store { return &Store{items: map[string]*Item{}, max: max} }

// Observe queues text an agent received. Conversations resend earlier tool
// results on every turn, so each (agent, text) pair is queued once.
func (s *Store) Observe(agent, source, text string) bool {
	if len(text) > MaxText {
		text = text[:MaxText]
	}
	sum := sha256.Sum256([]byte(agent + "\x00" + text))
	id := hex.EncodeToString(sum[:8])
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[id]; ok {
		return false
	}
	s.items[id] = &Item{ID: id, Agent: agent, Source: source, Text: text, ObservedAt: time.Now(), Status: "pending"}
	s.order = append(s.order, id)
	if len(s.order) > s.max {
		delete(s.items, s.order[0])
		s.order = s.order[1:]
	}
	return true
}

// Pending returns up to limit unscanned items, oldest first.
func (s *Store) Pending(limit int) []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := []Item{}
	for _, id := range s.order {
		if it := s.items[id]; it.Status == "pending" {
			out = append(out, *it)
			if len(out) == limit {
				break
			}
		}
	}
	return out
}

// Resolve records the classifier's probability that the text is an injection.
func (s *Store) Resolve(id string, injection float64, model string) (Item, error) {
	if injection < 0 || injection > 1 {
		return Item{}, fmt.Errorf("probability %v out of range", injection)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	switch {
	case !ok:
		return Item{}, fmt.Errorf("no scan %q", id)
	case it.Status != "pending":
		return Item{}, fmt.Errorf("scan %q already done", id)
	}
	it.Status, it.Injection, it.Model = "done", &injection, model
	return *it, nil
}

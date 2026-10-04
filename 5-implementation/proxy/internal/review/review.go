// Package review is the human-in-the-loop queue for decisions the rules
// could not make: tools with no policy entry and variables no rule
// recognized. An AI advisor may attach a suggestion; a person applies it,
// unless the policy file lets confident suggestions apply automatically.
package review

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
)

// Suggestion is advice from the AI advisor. Probabilities and Confidence are
// the classifier's calibrated outputs (0-1); Rationale is optional text.
type Suggestion struct {
	Value         string             `json:"value"`
	Probabilities map[string]float64 `json:"probabilities,omitempty"`
	Confidence    float64            `json:"confidence"`
	Rationale     string             `json:"rationale,omitempty"`
	Model         string             `json:"model"`
	At            time.Time          `json:"at"`
}

type Item struct {
	ID         string         `json:"id"`
	Kind       string         `json:"kind"` // tool | variable
	Subject    string         `json:"subject"`
	Context    map[string]any `json:"context"` // what the advisor may see; never secret values
	Current    string         `json:"current"` // what the rules do today
	Options    []string       `json:"options"`
	FirstSeen  time.Time      `json:"first_seen"`
	SeenBy     string         `json:"seen_by,omitempty"`
	Suggestion *Suggestion    `json:"suggestion,omitempty"`
	Status     string         `json:"status"` // pending | applied | dismissed
	Decision   string         `json:"decision,omitempty"`
	DecidedAt  *time.Time     `json:"decided_at,omitempty"`
	DecidedBy  string         `json:"decided_by,omitempty"` // reviewer | auto-accept | hushgate.yaml
	// Auto says whether the suggestion was applied without a person, and why or why not.
	Auto string `json:"auto,omitempty"`
}

var (
	ToolOptions     = []string{"local", "network", "deny"}
	VariableOptions = []string{"C0", "C1", "C2", "C3"}
)

type Store struct {
	mu    sync.Mutex
	items map[string]*Item
}

func NewStore() *Store { return &Store{items: map[string]*Item{}} }

// ObserveTool queues a tool the policy has no rule for. Idempotent.
func (s *Store) ObserveTool(name, description string, schema any, current, agent string) {
	s.observe(&Item{
		ID: "tool:" + name, Kind: "tool", Subject: name, Current: current, Options: ToolOptions, SeenBy: agent,
		Context: map[string]any{"description": description, "input_schema": schema},
	})
}

// ObserveVariable queues a variable that fell through to the default rule.
// Only the value's shape is recorded, never the value.
func (s *Store) ObserveVariable(name, value, current string) {
	s.observe(&Item{
		ID: "var:" + name, Kind: "variable", Subject: name, Current: current, Options: VariableOptions,
		Context: Shape(value),
	})
}

func (s *Store) observe(it *Item) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.items[it.ID]; ok {
		return
	}
	it.FirstSeen, it.Status = time.Now(), "pending"
	s.items[it.ID] = it
}

func (s *Store) List() []Item {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]Item, 0, len(s.items))
	for _, it := range s.items {
		out = append(out, *it)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].FirstSeen.Before(out[j].FirstSeen) })
	return out
}

func (s *Store) Get(id string) (Item, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok {
		return Item{}, false
	}
	return *it, true
}

// Suggest attaches advice to a pending item. It changes nothing else.
func (s *Store) Suggest(id string, sg Suggestion) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	switch {
	case !ok:
		return Item{}, fmt.Errorf("no review %q", id)
	case it.Status != "pending":
		return Item{}, fmt.Errorf("review %q is already %s", id, it.Status)
	case !contains(it.Options, sg.Value):
		return Item{}, fmt.Errorf("value %q not one of %v", sg.Value, it.Options)
	}
	sg.At = time.Now()
	it.Suggestion = &sg
	return *it, nil
}

// Resolve records a human decision; the caller applies it.
func (s *Store) Resolve(id, status, decision string) (Item, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	switch {
	case !ok:
		return Item{}, fmt.Errorf("no review %q", id)
	case it.Status != "pending":
		return Item{}, fmt.Errorf("review %q is already %s", id, it.Status)
	case status == "applied" && !contains(it.Options, decision):
		return Item{}, fmt.Errorf("value %q not one of %v", decision, it.Options)
	}
	now := time.Now()
	it.Status, it.Decision, it.DecidedAt, it.DecidedBy = status, decision, &now, "reviewer"
	return *it, nil
}

// Settle closes a pending item because the rule now exists in the policy
// file (written by a reviewer or edited by hand). Unknown or decided ids are ignored.
func (s *Store) Settle(id, decision string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	it, ok := s.items[id]
	if !ok || it.Status != "pending" {
		return false
	}
	now := time.Now()
	it.Status, it.Decision, it.DecidedAt, it.DecidedBy = "applied", decision, &now, "hushgate.yaml"
	return true
}

// SetDecidedBy records who made a decision that was written to the policy file.
func (s *Store) SetDecidedBy(id, by string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it, ok := s.items[id]; ok {
		it.DecidedBy = by
	}
}

// SetAuto records the auto-accept outcome for an item's suggestion.
func (s *Store) SetAuto(id, note string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if it, ok := s.items[id]; ok {
		it.Auto = note
	}
}

// Shape describes a value without revealing it: length, character classes,
// entropy and a run-length pattern ("payments-oncall" -> "a8-a6").
func Shape(v string) map[string]any {
	var classes []string
	var lower, upper, digit, other bool
	var pat strings.Builder
	prev, run := rune(0), 0
	flush := func() {
		if run == 0 {
			return
		}
		pat.WriteRune(prev)
		if prev == 'a' || prev == 'A' || prev == '9' {
			fmt.Fprintf(&pat, "%d", run)
		}
	}
	for _, r := range v {
		var c rune
		switch {
		case unicode.IsLower(r):
			c, lower = 'a', true
		case unicode.IsUpper(r):
			c, upper = 'A', true
		case unicode.IsDigit(r):
			c, digit = '9', true
		default:
			c, other = r, true
		}
		if c == prev && (c == 'a' || c == 'A' || c == '9') {
			run++
			continue
		}
		flush()
		prev, run = c, 1
	}
	flush()
	for _, c := range []struct {
		on   bool
		name string
	}{{lower, "lowercase"}, {upper, "uppercase"}, {digit, "digits"}, {other, "symbols"}} {
		if c.on {
			classes = append(classes, c.name)
		}
	}
	return map[string]any{
		"length":  len(v),
		"classes": classes,
		"entropy": math.Round(entropy(v)*10) / 10,
		"pattern": pat.String(),
	}
}

func entropy(s string) float64 {
	counts := map[rune]int{}
	for _, r := range s {
		counts[r]++
	}
	var h float64
	n := float64(len([]rune(s)))
	for _, c := range counts {
		p := float64(c) / n
		h -= p * math.Log2(p)
	}
	return h
}

func contains(xs []string, x string) bool {
	for _, y := range xs {
		if x == y {
			return true
		}
	}
	return false
}

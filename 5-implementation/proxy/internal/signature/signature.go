// Package signature matches tool calls against a feed of known attack
// patterns (remote code execution, unsafe deserialization, supply-chain
// packages, credential theft ...). Feeds are YAML documents loaded from files
// or URLs, so a security team can publish them centrally.
package signature

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Signature struct {
	ID          string   `yaml:"id" json:"id"`
	Name        string   `yaml:"name" json:"name"`
	Category    string   `yaml:"category" json:"category"`
	Severity    string   `yaml:"severity" json:"severity"` // low | medium | high | critical
	Action      string   `yaml:"action" json:"action"`     // alert | block | kill
	Tools       []string `yaml:"tools" json:"tools"`       // tool names, or "*" for every tool
	Pattern     string   `yaml:"pattern" json:"pattern"`   // RE2 regular expression
	Unless      string   `yaml:"unless,omitempty" json:"unless,omitempty"`
	Description string   `yaml:"description" json:"description"`
	Reference   string   `yaml:"reference,omitempty" json:"reference,omitempty"`

	re, unless *regexp.Regexp
}

type Feed struct {
	Name       string      `yaml:"feed"`
	Version    string      `yaml:"version"`
	Signatures []Signature `yaml:"signatures"`
}

var (
	actions    = map[string]int{"alert": 1, "block": 2, "kill": 3}
	severities = map[string]bool{"low": true, "medium": true, "high": true, "critical": true}
)

// Rank orders actions by strength so the strictest decision wins.
func Rank(action string) int { return actions[action] }

// ParseFeed decodes and validates a feed; one bad signature rejects the feed.
func ParseFeed(data []byte) (*Feed, error) {
	f := &Feed{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(f); err != nil {
		return nil, fmt.Errorf("invalid feed YAML: %w", err)
	}
	var errs []string
	seen := map[string]bool{}
	for i := range f.Signatures {
		s := &f.Signatures[i]
		where := s.ID
		if where == "" {
			where = fmt.Sprintf("signatures[%d]", i)
		}
		switch {
		case s.ID == "":
			errs = append(errs, where+": id is required")
		case seen[s.ID]:
			errs = append(errs, where+": duplicate id")
		}
		seen[s.ID] = true
		if Rank(s.Action) == 0 {
			errs = append(errs, fmt.Sprintf("%s: action %q is not alert, block or kill", where, s.Action))
		}
		if !severities[s.Severity] {
			errs = append(errs, fmt.Sprintf("%s: severity %q is not low, medium, high or critical", where, s.Severity))
		}
		if len(s.Tools) == 0 {
			s.Tools = []string{"*"}
		}
		var err error
		if s.re, err = regexp.Compile(s.Pattern); err != nil || s.Pattern == "" {
			errs = append(errs, fmt.Sprintf("%s: bad pattern: %v", where, err))
		}
		if s.Unless != "" {
			if s.unless, err = regexp.Compile(s.Unless); err != nil {
				errs = append(errs, fmt.Sprintf("%s: bad unless pattern: %v", where, err))
			}
		}
	}
	if len(errs) > 0 {
		return nil, errors.New(strings.Join(errs, "; "))
	}
	return f, nil
}

// Hit is a signature that matched a tool call.
type Hit struct {
	Signature *Signature
	Match     string // the matched text, trimmed for the audit log
}

// FeedStatus is what the dashboard shows per source.
type FeedStatus struct {
	Source  string `json:"source"`
	Name    string `json:"name"`
	Version string `json:"version"`
	Count   int    `json:"count"`
	Error   string `json:"error,omitempty"` // last load failure; the previous copy stays active
}

// Set is the active signatures from all feeds, minus disabled ids.
type Set struct {
	mu       sync.RWMutex
	sigs     []*Signature
	feeds    []FeedStatus
	disabled map[string]bool
}

func (s *Set) Replace(feeds []*Feed, status []FeedStatus, disabled []string) {
	off := map[string]bool{}
	for _, id := range disabled {
		off[id] = true
	}
	// A later feed overrides an earlier one per id, so a central feed can
	// update a local baseline without producing duplicate hits.
	var sigs []*Signature
	at := map[string]int{}
	for _, f := range feeds {
		for i := range f.Signatures {
			sig := &f.Signatures[i]
			if j, ok := at[sig.ID]; ok {
				sigs[j] = sig
				continue
			}
			at[sig.ID] = len(sigs)
			sigs = append(sigs, sig)
		}
	}
	s.mu.Lock()
	s.sigs, s.feeds, s.disabled = sigs, status, off
	s.mu.Unlock()
}

// Match checks a tool call. text is the call's input flattened to plain text;
// shellText, if set, is the normalized shell command for shell tools.
func (s *Set) Match(tool, text, shellText string) []Hit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var hits []Hit
	for _, sig := range s.sigs {
		if s.disabled[sig.ID] || !appliesTo(sig, tool) {
			continue
		}
		for _, t := range []string{text, shellText} {
			if t == "" {
				continue
			}
			m := sig.re.FindString(t)
			if m == "" || (sig.unless != nil && sig.unless.MatchString(t)) {
				continue
			}
			if len(m) > 80 {
				m = m[:80] + "…"
			}
			hits = append(hits, Hit{Signature: sig, Match: m})
			break
		}
	}
	sort.SliceStable(hits, func(i, j int) bool { return Rank(hits[i].Signature.Action) > Rank(hits[j].Signature.Action) })
	return hits
}

// SignatureView adds whether a signature is switched off in the policy.
type SignatureView struct {
	Signature
	Enabled bool `json:"enabled"`
}

func (s *Set) List() ([]SignatureView, []FeedStatus) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SignatureView, 0, len(s.sigs))
	for _, sig := range s.sigs {
		out = append(out, SignatureView{Signature: *sig, Enabled: !s.disabled[sig.ID]})
	}
	return out, append([]FeedStatus(nil), s.feeds...)
}

func appliesTo(sig *Signature, tool string) bool {
	for _, t := range sig.Tools {
		if t == "*" || t == tool {
			return true
		}
	}
	return false
}

// Flatten turns a tool call's JSON input into plain text (all string values,
// one per line) so patterns see real newlines and quotes, not JSON escapes.
func Flatten(inputJSON string) string {
	var v any
	if json.Unmarshal([]byte(inputJSON), &v) != nil {
		return inputJSON
	}
	var b strings.Builder
	var walk func(any)
	walk = func(x any) {
		switch t := x.(type) {
		case string:
			b.WriteString(t)
			b.WriteByte('\n')
		case []any:
			for _, e := range t {
				walk(e)
			}
		case map[string]any:
			keys := make([]string, 0, len(t))
			for k := range t {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				walk(t[k])
			}
		}
	}
	walk(v)
	return b.String()
}

// Package policy decides what happens to each tool call the model requests.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

// Sink says where a tool's arguments end up.
type Sink string

const (
	Local   Sink = "local"   // stays on the machine (file writes, local shell)
	Network Sink = "network" // leaves the machine (email, HTTP, chat)
	Deny    Sink = "deny"    // never allowed
)

type Action string

const (
	Allow Action = "allow"
	Block Action = "block" // drop the call, agent keeps running
	Kill  Action = "kill"  // drop the call and trip the agent's circuit breaker
)

type Policy struct {
	mu      sync.RWMutex
	Tools   map[string]Sink `json:"tools"`
	Default Sink            `json:"default"`
}

// Load reads a JSON policy file. An empty path yields a permissive default.
func Load(path string) (*Policy, error) {
	p := &Policy{Tools: map[string]Sink{}, Default: Local}
	if path == "" {
		return p, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(b, p); err != nil {
		return nil, fmt.Errorf("parse policy: %w", err)
	}
	if p.Default == "" {
		p.Default = Deny
	}
	return p, nil
}

func (p *Policy) SinkFor(tool string) Sink {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if s, ok := p.Tools[tool]; ok {
		return s
	}
	return p.Default
}

// Has reports whether the tool has an explicit rule.
func (p *Policy) Has(tool string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	_, ok := p.Tools[tool]
	return ok
}

// Set adds or changes a rule at runtime (after a human approves a review).
func (p *Policy) Set(tool string, sink Sink) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.Tools == nil {
		p.Tools = map[string]Sink{}
	}
	p.Tools[tool] = sink
}

// Snapshot is a copy that is safe to serialize.
func (p *Policy) Snapshot() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tools := make(map[string]Sink, len(p.Tools))
	for k, v := range p.Tools {
		tools[k] = v
	}
	return map[string]any{"tools": tools, "default": p.Default}
}

func ValidSink(s string) bool {
	return s == string(Local) || s == string(Network) || s == string(Deny)
}

// Decide returns the action for a tool call and whether vault placeholders in
// its arguments may be swapped back to real values. A credential headed off
// the machine means the agent is compromised (kill); personal data headed off
// the machine is a policy violation (block).
func (p *Policy) Decide(tool string, carried []vault.Ref) (act Action, rehydrate bool, reason string) {
	highest := vault.Tier(-1)
	for _, r := range carried {
		highest = max(highest, r.Tier)
	}
	switch p.SinkFor(tool) {
	case Deny:
		return Block, false, "tool not permitted by policy"
	case Network:
		switch {
		case highest >= vault.Secret:
			return Kill, false, "vaulted secret in arguments of network tool"
		case highest >= vault.Confidential:
			return Block, false, "personal or confidential data in arguments of network tool"
		}
		return Allow, false, "network tool, no secrets"
	default:
		return Allow, true, "local tool"
	}
}

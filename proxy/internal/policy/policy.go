// Package policy decides what happens to each tool call the model requests.
package policy

import (
	"encoding/json"
	"fmt"
	"os"
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
	if s, ok := p.Tools[tool]; ok {
		return s
	}
	return p.Default
}

// Decide returns the action for a tool call and whether vault placeholders in
// its arguments may be swapped back to real values.
func (p *Policy) Decide(tool string, carriesSecrets bool) (act Action, rehydrate bool, reason string) {
	switch p.SinkFor(tool) {
	case Deny:
		return Block, false, "tool not permitted by policy"
	case Network:
		if carriesSecrets {
			return Kill, false, "vaulted secret in arguments of network tool"
		}
		return Allow, false, "network tool, no secrets"
	default:
		return Allow, true, "local tool"
	}
}

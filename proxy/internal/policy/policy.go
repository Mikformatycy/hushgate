// Package policy decides what happens to each tool call the model requests.
package policy

import (
	"sync"

	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
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
	// OnNetwork is the action when vaulted data of a tier is in a network
	// tool call. Nil means the defaults: C2 block, C3 kill.
	OnNetwork map[vault.Tier]Action `json:"-"`
}

var defaultOnNetwork = map[vault.Tier]Action{vault.Confidential: Block, vault.Secret: Kill}

// Replace swaps in a new rule set atomically (live policy reload).
func (p *Policy) Replace(tools map[string]Sink, def Sink, onNetwork map[vault.Tier]Action) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.Tools, p.Default, p.OnNetwork = tools, def, onNetwork
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

// Snapshot is a copy that is safe to serialize.
func (p *Policy) Snapshot() map[string]any {
	p.mu.RLock()
	defer p.mu.RUnlock()
	tools := make(map[string]Sink, len(p.Tools))
	for k, v := range p.Tools {
		tools[k] = v
	}
	on := map[string]Action{}
	for t, a := range p.onNetwork() {
		on[t.String()] = a
	}
	return map[string]any{"tools": tools, "default": p.Default, "on_network_tool": on}
}

func (p *Policy) onNetwork() map[vault.Tier]Action {
	if p.OnNetwork == nil {
		return defaultOnNetwork
	}
	return p.OnNetwork
}

func ValidSink(s string) bool {
	return s == string(Local) || s == string(Network) || s == string(Deny)
}

// Decide returns the action for a tool call and whether vault placeholders in
// its arguments may be swapped back to real values. A credential headed off
// the machine means the agent is compromised (kill); personal data headed off
// the machine is a policy violation (block).
func (p *Policy) Decide(tool string, carried []vault.Ref) (act Action, rehydrate bool, reason string) {
	return p.DecideAs(p.SinkFor(tool), carried)
}

// DecideAs decides for an explicit sink, e.g. when the Bash guard finds that a
// "local" shell call actually sends data off the machine.
func (p *Policy) DecideAs(sink Sink, carried []vault.Ref) (act Action, rehydrate bool, reason string) {
	highest := vault.Tier(-1)
	for _, r := range carried {
		highest = max(highest, r.Tier)
	}
	switch sink {
	case Deny:
		return Block, false, "tool not permitted by policy"
	case Network:
		if highest < 0 {
			return Allow, false, "network tool, no secrets"
		}
		p.mu.RLock()
		act, ok := p.onNetwork()[highest]
		p.mu.RUnlock()
		if !ok {
			act = Allow
		}
		what := "personal or confidential data"
		if highest >= vault.Secret {
			what = "vaulted secret"
		}
		if act == Allow {
			return Allow, false, what + " in arguments of network tool, allowed by policy for " + highest.String()
		}
		return act, false, what + " in arguments of network tool"
	default:
		return Allow, true, "local tool"
	}
}

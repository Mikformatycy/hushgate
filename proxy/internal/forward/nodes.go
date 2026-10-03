package forward

import (
	"crypto/subtle"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
)

// Node is a device or workload allowed to talk to LLM providers. In a real
// deployment the identity comes from a device certificate or SSO session;
// here it is the proxy credentials that MDM bakes into the proxy URL.
type Node struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
	Token string `json:"token,omitempty"`
}

type Registry struct {
	mu    sync.RWMutex
	nodes map[string]Node
}

func NewRegistry(nodes []Node) *Registry {
	r := &Registry{}
	r.Replace(nodes)
	return r
}

// Replace swaps the allowlist (live policy reload).
func (r *Registry) Replace(nodes []Node) {
	m := make(map[string]Node, len(nodes))
	for _, n := range nodes {
		m[n.ID] = n
	}
	r.mu.Lock()
	r.nodes = m
	r.mu.Unlock()
}

// Authenticate resolves Proxy-Authorization: Basic id:token to a node.
func (r *Registry) Authenticate(req *http.Request) *Node {
	h := req.Header.Get("Proxy-Authorization")
	enc, ok := strings.CutPrefix(h, "Basic ")
	if !ok {
		return nil
	}
	raw, err := base64.StdEncoding.DecodeString(enc)
	if err != nil {
		return nil
	}
	id, token, _ := strings.Cut(string(raw), ":")
	r.mu.RLock()
	n, ok := r.nodes[id]
	r.mu.RUnlock()
	if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(n.Token)) != 1 {
		return nil
	}
	return &n
}

// List returns nodes without their tokens.
func (r *Registry) List() []Node {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, Node{ID: n.ID, Owner: n.Owner})
	}
	return out
}

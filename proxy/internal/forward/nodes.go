package forward

import (
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"os"
	"strings"
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
	nodes map[string]Node
}

func LoadNodes(path string) (*Registry, error) {
	reg := &Registry{nodes: map[string]Node{}}
	if path == "" {
		return reg, nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var f struct {
		Nodes []Node `json:"nodes"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, err
	}
	for _, n := range f.Nodes {
		reg.nodes[n.ID] = n
	}
	return reg, nil
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
	n, ok := r.nodes[id]
	if !ok || subtle.ConstantTimeCompare([]byte(token), []byte(n.Token)) != 1 {
		return nil
	}
	return &n
}

// List returns nodes without their tokens.
func (r *Registry) List() []Node {
	out := make([]Node, 0, len(r.nodes))
	for _, n := range r.nodes {
		out = append(out, Node{ID: n.ID, Owner: n.Owner})
	}
	return out
}

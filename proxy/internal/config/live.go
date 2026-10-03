package config

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"
)

// ApplyFunc pushes a validated config into the running components.
type ApplyFunc func(old, new *Config) error

// Status describes the last reload, for the dashboard.
type Status struct {
	Path     string    `json:"path"`
	Version  string    `json:"version"` // short hash of the file and its .env files
	LoadedAt time.Time `json:"loaded_at"`
	Error    string    `json:"error,omitempty"` // last rejected edit; the previous policy stays active
}

// Live holds the active config and reloads it when the file (or a vault
// .env file it lists) changes. Edits that fail validation are rejected and
// the last good config stays in force.
type Live struct {
	path    string
	apply   ApplyFunc
	onEvent func(action, detail string) // "reloaded" or "rejected"
	// Extra returns more content to watch for a config, e.g. signature feeds;
	// a change in it triggers a reload like an edit to the file itself.
	Extra func(*Config) [][]byte

	cur     atomic.Pointer[Config]
	mu      sync.Mutex // serializes reloads and write-backs
	hash    string
	errHash string
	status  atomic.Pointer[Status]
}

func NewLive(path string, apply ApplyFunc, onEvent func(action, detail string)) *Live {
	return &Live{path: path, apply: apply, onEvent: onEvent}
}

// Load performs the initial load; unlike later reloads, an error is fatal.
func (l *Live) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.reload(true)
	return err
}

func (l *Live) Get() *Config { return l.cur.Load() }

func (l *Live) Status() Status { return *l.status.Load() }

// Watch polls for changes until ctx ends. Polling (not inotify) because
// editors replace files on save and bind mounts don't forward those events.
func (l *Live) Watch(ctx context.Context, every time.Duration) {
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			l.mu.Lock()
			l.reload(false)
			l.mu.Unlock()
		}
	}
}

// settle is how long a changed file must stay unchanged before it is applied.
// Saves that write in place briefly leave a truncated file; reading one of
// those could apply half a policy.
const settle = 200 * time.Millisecond

// reload applies the file if it changed; caller holds l.mu.
func (l *Live) reload(initial bool) (changed bool, err error) {
	cfg, perr, hash, err := l.read()
	if err != nil {
		return false, l.reject(initial, "cannot read policy file: "+err.Error(), "read:"+err.Error())
	}
	if hash == l.hash {
		return false, nil
	}
	if !initial {
		time.Sleep(settle)
		_, _, again, err := l.read()
		if err != nil || again != hash {
			return false, nil // still being written: try again on the next tick
		}
	}
	if perr != nil {
		return false, l.reject(initial, perr.Error(), hash)
	}
	old := l.cur.Load()
	if err := l.apply(old, cfg); err != nil {
		return false, l.reject(initial, err.Error(), hash)
	}
	l.cur.Store(cfg)
	l.hash, l.errHash = hash, ""
	l.status.Store(&Status{Path: l.path, Version: hash[:8], LoadedAt: time.Now()})
	if !initial {
		changes := Diff(old, cfg)
		if len(changes) == 0 {
			changes = []string{"vault .env files or signature feeds changed"}
		}
		l.onEvent("reloaded", joinChanges(changes))
	}
	return true, nil
}

// read loads the file and fingerprints it with everything it points at.
func (l *Live) read() (cfg *Config, perr error, hash string, err error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, nil, "", err
	}
	cfg, perr = Parse(data)
	return cfg, perr, l.fingerprint(data, cfg), nil
}

// reject records a bad edit once per distinct file content.
func (l *Live) reject(initial bool, msg, hash string) error {
	if initial {
		return errors.New(msg)
	}
	if hash != l.errHash {
		l.errHash = hash
		st := l.Status()
		st.Error = msg
		l.status.Store(&st)
		l.onEvent("rejected", msg)
	}
	return errors.New(msg)
}

// fingerprint covers the policy file and the .env files it points at, so
// editing a vault file also triggers a reload.
func (l *Live) fingerprint(data []byte, cfg *Config) string {
	h := sha256.New()
	h.Write(data)
	if cfg != nil {
		for _, f := range cfg.Vault.EnvFiles {
			b, _ := os.ReadFile(f)
			h.Write([]byte(f))
			h.Write(b)
		}
		if l.Extra != nil {
			for _, b := range l.Extra(cfg) {
				h.Write(b)
			}
		}
	}
	return hex.EncodeToString(h.Sum(nil))
}

// SetToolRule writes tools.rules.<tool> into the file and reloads.
func (l *Live) SetToolRule(tool, sink string) error {
	return l.edit([]string{"tools", "rules"}, tool, sink)
}

// SetVariableOverride writes vault.overrides.<name> into the file and reloads.
func (l *Live) SetVariableOverride(name, tier string) error {
	return l.edit([]string{"vault", "overrides"}, name, tier)
}

// edit changes one key in the YAML document, keeping comments and layout,
// writes the file in place (bind mounts break on rename) and reloads.
func (l *Live) edit(parents []string, key, value string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	data, err := os.ReadFile(l.path)
	if err != nil {
		return err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return fmt.Errorf("policy file is not valid YAML: %w", err)
	}
	if len(doc.Content) == 0 {
		doc = yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{{Kind: yaml.MappingNode}}}
	}
	m := doc.Content[0]
	for _, p := range parents {
		m = child(m, p)
	}
	setScalar(m, key, value)

	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return err
	}
	if _, err := Parse(buf.Bytes()); err != nil {
		return fmt.Errorf("edit would make the policy invalid: %w", err)
	}
	if err := os.WriteFile(l.path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("cannot write policy file: %w", err)
	}
	_, err = l.reload(false)
	return err
}

// child returns the mapping under key, creating it if needed.
func child(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			v := m.Content[i+1]
			if v.Kind != yaml.MappingNode { // e.g. "rules:" with no entries
				*v = yaml.Node{Kind: yaml.MappingNode, LineComment: v.LineComment}
			}
			return v
		}
	}
	v := &yaml.Node{Kind: yaml.MappingNode}
	m.Content = append(m.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: key}, v)
	return v
}

func setScalar(m *yaml.Node, key, value string) {
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Kind, m.Content[i+1].Value, m.Content[i+1].Tag = yaml.ScalarNode, value, ""
			return
		}
	}
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value})
}

func joinChanges(c []string) string {
	var b bytes.Buffer
	for i, s := range c {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(s)
	}
	return b.String()
}

package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDefaults(t *testing.T) {
	c, err := Parse(nil)
	if err != nil {
		t.Fatal(err)
	}
	if c.Tools.Default != "deny" || c.Masking.MaskFrom != "C2" || *c.Injection.AlertThreshold != 0.8 {
		t.Fatalf("defaults = %+v", c)
	}
	if c.Masking.OnNetworkTool["C3"] != "kill" || c.Masking.OnNetworkTool["C2"] != "block" {
		t.Fatalf("tier actions = %v", c.Masking.OnNetworkTool)
	}
	if c.LimitFor("anyone") != 0 || !c.ModelAllowed("anything") {
		t.Fatal("empty config should be unlimited and allow any model")
	}
}

func TestParseRejects(t *testing.T) {
	cases := map[string]string{
		"unknown key":   "toolz:\n  default: deny\n",
		"bad sink":      "tools:\n  rules:\n    Bash: lokal\n",
		"bad tier":      "masking:\n  mask_from: C9\n",
		"bad action":    "masking:\n  on_network_tool:\n    C3: explode\n",
		"bad threshold": "injection:\n  alert_threshold: 2\n",
		"neg budget":    "budgets:\n  default_tokens: -1\n",
		"dup node":      "nodes:\n  - {id: a, token: x}\n  - {id: a, token: y}\n",
		"not yaml":      "tools: [\n",
	}
	for name, src := range cases {
		if _, err := Parse([]byte(src)); err == nil {
			t.Errorf("%s: accepted", name)
		}
	}
	_, err := Parse([]byte("tools:\n  rules:\n    Bash: lokal\n"))
	if !strings.Contains(err.Error(), "tools.rules.Bash") {
		t.Fatalf("error should name the setting: %v", err)
	}
}

func TestLookups(t *testing.T) {
	c, err := Parse([]byte(`
budgets:
  default_tokens: 1000
  agents: {big: 5000}
models:
  allow: ["claude-haiku-*"]
llm_hosts:
  approved: [api.anthropic.com]
`))
	if err != nil {
		t.Fatal(err)
	}
	if c.LimitFor("big") != 5000 || c.LimitFor("other") != 1000 {
		t.Fatal("budgets")
	}
	if !c.ModelAllowed("claude-haiku-4-5") || c.ModelAllowed("claude-opus-5-5") {
		t.Fatal("models")
	}
	if !c.HostApproved("api.anthropic.com") || c.HostApproved("evil.example") {
		t.Fatal("hosts")
	}
}

func TestDiffRedactsTokens(t *testing.T) {
	a, _ := Parse([]byte("tools:\n  rules: {Bash: local}\nnodes:\n  - {id: n1, token: secret1}\n"))
	b, _ := Parse([]byte("tools:\n  rules: {Bash: deny, Read: local}\nnodes:\n  - {id: n1, token: secret2}\n"))
	d := strings.Join(Diff(a, b), "\n")
	for _, want := range []string{"tools.rules.Bash: local → deny", "tools.rules.Read: added local", "nodes.n1.token: changed"} {
		if !strings.Contains(d, want) {
			t.Errorf("diff missing %q:\n%s", want, d)
		}
	}
	if strings.Contains(d, "secret") {
		t.Fatalf("diff leaked a token:\n%s", d)
	}
}

type events struct{ got []string }

func (e *events) add(action, detail string) { e.got = append(e.got, action+": "+detail) }

func writeFile(t *testing.T, path, s string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(s), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLiveReloadRejectAndRecover(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hushgate.yaml")
	env := filepath.Join(dir, ".env")
	writeFile(t, env, "A=1\n")
	writeFile(t, path, "tools:\n  rules: {Bash: local}\nvault:\n  env_files: ["+env+"]\n")

	applied := 0
	ev := &events{}
	l := NewLive(path, func(old, new *Config) error { applied++; return nil }, ev.add)
	if err := l.Load(); err != nil {
		t.Fatal(err)
	}
	if applied != 1 || len(ev.got) != 0 {
		t.Fatalf("initial load: applied=%d events=%v", applied, ev.got)
	}

	reload := func() { l.mu.Lock(); l.reload(false); l.mu.Unlock() }

	reload() // nothing changed
	if applied != 1 {
		t.Fatal("reloaded without a change")
	}

	writeFile(t, path, "tools:\n  rules: {Bash: deny}\nvault:\n  env_files: ["+env+"]\n")
	reload()
	if l.Get().Tools.Rules["Bash"] != "deny" || !strings.Contains(ev.got[0], "reloaded: tools.rules.Bash: local → deny") {
		t.Fatalf("change not applied: %v", ev.got)
	}

	writeFile(t, path, "tools:\n  rules: {Bash: nope}\n")
	reload()
	reload() // same bad content: reported once
	if l.Get().Tools.Rules["Bash"] != "deny" {
		t.Fatal("bad edit replaced the active policy")
	}
	if len(ev.got) != 2 || !strings.HasPrefix(ev.got[1], "rejected: tools.rules.Bash") || l.Status().Error == "" {
		t.Fatalf("rejection: %v / %+v", ev.got, l.Status())
	}

	writeFile(t, path, "tools:\n  rules: {Bash: local}\nvault:\n  env_files: ["+env+"]\n")
	reload()
	if l.Get().Tools.Rules["Bash"] != "local" || l.Status().Error != "" {
		t.Fatal("fixed file not picked up")
	}

	writeFile(t, env, "A=2\n") // vault file edit alone triggers a reload
	before := applied
	reload()
	if applied != before+1 || !strings.HasSuffix(ev.got[len(ev.got)-1], "vault .env files or signature feeds changed") {
		t.Fatalf("env change not reloaded: %v", ev.got)
	}
}

func TestEditKeepsComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hushgate.yaml")
	writeFile(t, path, "# HushGate policy\ntools:\n  default: deny # unknown tools are blocked\n  rules:\n    Read: local\nvault:\n  overrides: {}\n")
	l := NewLive(path, func(old, new *Config) error { return nil }, func(string, string) {})
	if err := l.Load(); err != nil {
		t.Fatal(err)
	}
	if err := l.SetToolRule("post_to_slack", "network"); err != nil {
		t.Fatal(err)
	}
	if err := l.SetVariableOverride("TEAM_CHANNEL", "C1"); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(path)
	s := string(b)
	for _, want := range []string{"# HushGate policy", "# unknown tools are blocked", "post_to_slack: network", "TEAM_CHANNEL: C1", "Read: local"} {
		if !strings.Contains(s, want) {
			t.Errorf("missing %q in:\n%s", want, s)
		}
	}
	if l.Get().Tools.Rules["post_to_slack"] != "network" || l.Get().Vault.Overrides["TEAM_CHANNEL"] != "C1" {
		t.Fatal("edit not applied")
	}
	if err := l.SetToolRule("x", "bogus"); err == nil {
		t.Fatal("invalid edit written")
	}
}

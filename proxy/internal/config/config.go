// Package config is HushGate's single policy file: parsing, validation,
// diffing, live reload and writing review decisions back.
package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Node struct {
	ID    string `yaml:"id"`
	Owner string `yaml:"owner"`
	Token string `yaml:"token"`
}

type Config struct {
	Tools struct {
		Default string            `yaml:"default"`
		Rules   map[string]string `yaml:"rules"`
	} `yaml:"tools"`
	Masking struct {
		MaskFrom      string            `yaml:"mask_from"`
		OnNetworkTool map[string]string `yaml:"on_network_tool"`
	} `yaml:"masking"`
	Vault struct {
		EnvFiles  []string          `yaml:"env_files"`
		Overrides map[string]string `yaml:"overrides"`
	} `yaml:"vault"`
	Budgets struct {
		DefaultTokens int64            `yaml:"default_tokens"`
		Agents        map[string]int64 `yaml:"agents"`
	} `yaml:"budgets"`
	Models struct {
		Allow []string `yaml:"allow"`
	} `yaml:"models"`
	LLMHosts struct {
		Approved []string `yaml:"approved"`
	} `yaml:"llm_hosts"`
	Injection struct {
		AlertThreshold *float64 `yaml:"alert_threshold"`
	} `yaml:"injection"`
	Nodes      []Node `yaml:"nodes"`
	Signatures struct {
		Feeds          []string `yaml:"feeds"`
		RefreshSeconds int      `yaml:"refresh_seconds"`
		Disabled       []string `yaml:"disabled"`
	} `yaml:"signatures"`
	BashGuard struct {
		Tools           []string `yaml:"tools"`
		NetworkCommands []string `yaml:"network_commands"`
	} `yaml:"bash_guard"`
	Review struct {
		AutoAccept AutoAccept `yaml:"auto_accept"`
	} `yaml:"review"`
}

// AutoAccept lets the AI advisor's suggestion take effect without a person,
// when the classifier is sure enough and the outcome is on an allowed list.
type AutoAccept struct {
	Enabled        bool     `yaml:"enabled"`
	MinProbability *float64 `yaml:"min_probability"` // of the suggested option
	MinConfidence  *float64 `yaml:"min_confidence"`  // the classifier's calibrated confidence
	Tools          []string `yaml:"tools"`           // tool outcomes that may be applied automatically
	Variables      []string `yaml:"variables"`       // variable tiers that may be applied automatically
}

// Without explicit lists, auto-accept can only make things stricter than an
// unreviewed default: no tool becomes "local" (real secrets restored into a tool
// whose description the agent supplied) and no variable gets unmasked.
var (
	DefaultAutoAcceptTools     = []string{"network", "deny"}
	DefaultAutoAcceptVariables = []string{"C2", "C3"}
)

// DefaultNetworkCommands are programs that send data off the machine.
var DefaultNetworkCommands = []string{"curl", "wget", "nc", "ncat", "netcat", "socat", "scp", "sftp", "rsync",
	"ssh", "ftp", "telnet", "http", "https"}

var (
	sinks   = []string{"local", "network", "deny"}
	actions = []string{"allow", "block", "kill"}
	tiers   = []string{"C0", "C1", "C2", "C3"}
)

// Parse decodes and validates a policy file. Unknown keys are errors, so a
// typo never silently disables a control.
func Parse(data []byte) (*Config, error) {
	c := &Config{}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(c); err != nil && !errors.Is(err, io.EOF) { // EOF: empty file, all defaults
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	c.defaults()
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *Config) defaults() {
	if c.Tools.Default == "" {
		c.Tools.Default = "deny"
	}
	if c.Masking.MaskFrom == "" {
		c.Masking.MaskFrom = "C2"
	}
	if c.Masking.OnNetworkTool == nil {
		c.Masking.OnNetworkTool = map[string]string{}
	}
	for t, a := range map[string]string{"C0": "allow", "C1": "allow", "C2": "block", "C3": "kill"} {
		if _, ok := c.Masking.OnNetworkTool[t]; !ok {
			c.Masking.OnNetworkTool[t] = a
		}
	}
	if c.Signatures.RefreshSeconds == 0 {
		c.Signatures.RefreshSeconds = 300
	}
	if c.BashGuard.Tools == nil {
		c.BashGuard.Tools = []string{"Bash"}
	}
	if c.BashGuard.NetworkCommands == nil {
		c.BashGuard.NetworkCommands = DefaultNetworkCommands
	}
	if c.Injection.AlertThreshold == nil {
		v := 0.8
		c.Injection.AlertThreshold = &v
	}
	aa := &c.Review.AutoAccept
	if aa.MinProbability == nil {
		v := 0.95
		aa.MinProbability = &v
	}
	if aa.MinConfidence == nil {
		v := 0.9
		aa.MinConfidence = &v
	}
	if aa.Tools == nil {
		aa.Tools = DefaultAutoAcceptTools
	}
	if aa.Variables == nil {
		aa.Variables = DefaultAutoAcceptVariables
	}
}

func (c *Config) validate() error {
	var errs []string
	bad := func(format string, args ...any) { errs = append(errs, fmt.Sprintf(format, args...)) }

	if !oneOf(c.Tools.Default, sinks) {
		bad("tools.default: %q is not one of %s", c.Tools.Default, strings.Join(sinks, ", "))
	}
	for name, s := range c.Tools.Rules {
		if !oneOf(s, sinks) {
			bad("tools.rules.%s: %q is not one of %s", name, s, strings.Join(sinks, ", "))
		}
	}
	if !oneOf(c.Masking.MaskFrom, tiers) {
		bad("masking.mask_from: %q is not one of %s", c.Masking.MaskFrom, strings.Join(tiers, ", "))
	}
	for t, a := range c.Masking.OnNetworkTool {
		if !oneOf(t, tiers) {
			bad("masking.on_network_tool: %q is not a tier (%s)", t, strings.Join(tiers, ", "))
		}
		if !oneOf(a, actions) {
			bad("masking.on_network_tool.%s: %q is not one of %s", t, a, strings.Join(actions, ", "))
		}
	}
	for name, t := range c.Vault.Overrides {
		if !oneOf(t, tiers) {
			bad("vault.overrides.%s: %q is not one of %s", name, t, strings.Join(tiers, ", "))
		}
	}
	if c.Budgets.DefaultTokens < 0 {
		bad("budgets.default_tokens: must be 0 (unlimited) or more")
	}
	for a, n := range c.Budgets.Agents {
		if n < 0 {
			bad("budgets.agents.%s: must be 0 (unlimited) or more", a)
		}
	}
	for _, p := range c.Models.Allow {
		if _, err := path.Match(p, ""); err != nil {
			bad("models.allow: bad pattern %q", p)
		}
	}
	if t := *c.Injection.AlertThreshold; t < 0 || t > 1 {
		bad("injection.alert_threshold: %v must be between 0 and 1", t)
	}
	aa := c.Review.AutoAccept
	for name, v := range map[string]float64{"min_probability": *aa.MinProbability, "min_confidence": *aa.MinConfidence} {
		if v < 0 || v > 1 {
			bad("review.auto_accept.%s: %v must be between 0 and 1", name, v)
		}
	}
	for _, v := range aa.Tools {
		if !oneOf(v, sinks) {
			bad("review.auto_accept.tools: %q is not one of %s", v, strings.Join(sinks, ", "))
		}
	}
	for _, v := range aa.Variables {
		if !oneOf(v, tiers) {
			bad("review.auto_accept.variables: %q is not one of %s", v, strings.Join(tiers, ", "))
		}
	}
	if c.Signatures.RefreshSeconds < 10 {
		bad("signatures.refresh_seconds: must be at least 10")
	}
	for i, f := range c.Signatures.Feeds {
		if strings.TrimSpace(f) == "" {
			bad("signatures.feeds[%d]: empty source", i)
		}
	}
	seen := map[string]bool{}
	for i, n := range c.Nodes {
		switch {
		case n.ID == "":
			bad("nodes[%d]: id is required", i)
		case seen[n.ID]:
			bad("nodes: duplicate id %q", n.ID)
		case n.Token == "":
			bad("nodes.%s: token is required", n.ID)
		}
		seen[n.ID] = true
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// LimitFor is an agent's token budget; 0 means unlimited.
func (c *Config) LimitFor(agent string) int64 {
	if n, ok := c.Budgets.Agents[agent]; ok {
		return n
	}
	return c.Budgets.DefaultTokens
}

// ModelAllowed matches glob patterns; an empty list allows every model.
func (c *Config) ModelAllowed(model string) bool {
	if len(c.Models.Allow) == 0 {
		return true
	}
	for _, p := range c.Models.Allow {
		if ok, _ := path.Match(p, model); ok {
			return true
		}
	}
	return false
}

func (c *Config) HostApproved(host string) bool {
	return oneOf(host, c.LLMHosts.Approved)
}

// AutoAccepts decides whether the advisor's suggestion for a review of the
// given kind ("tool" or "variable") may be applied without a person. The
// reason explains the outcome either way; the Review page shows it.
func (c *Config) AutoAccepts(kind, value string, probabilities map[string]float64, confidence float64) (bool, string) {
	aa := c.Review.AutoAccept
	allowed, list := aa.Tools, "tools"
	if kind == "variable" {
		allowed, list = aa.Variables, "variables"
	}
	p, known := probabilities[value]
	switch {
	case !aa.Enabled:
		return false, "auto-accept is off"
	case !oneOf(value, allowed):
		return false, fmt.Sprintf("%s is not in review.auto_accept.%s", value, list)
	case !known:
		return false, "the advisor gave no probability for " + value
	case p < *aa.MinProbability:
		return false, fmt.Sprintf("%s at %s, below the %s needed", value, pct(p), pct(*aa.MinProbability))
	case confidence < *aa.MinConfidence:
		return false, fmt.Sprintf("confidence %s, below the %s needed", pct(confidence), pct(*aa.MinConfidence))
	}
	return true, fmt.Sprintf("%s at %s with %s confidence (needs %s and %s)",
		value, pct(p), pct(confidence), pct(*aa.MinProbability), pct(*aa.MinConfidence))
}

// pct prints 0.955 as "95.5%" and 0.9 as "90%".
func pct(v float64) string {
	return strings.TrimSuffix(fmt.Sprintf("%.1f", v*100), ".0") + "%"
}

// Diff lists human-readable changes between two configs. Node tokens are
// reported as changed, never printed.
func Diff(old, new *Config) []string {
	a, b := flatten(old), flatten(new)
	keys := map[string]bool{}
	for k := range a {
		keys[k] = true
	}
	for k := range b {
		keys[k] = true
	}
	var out []string
	for k := range keys {
		va, oka := a[k]
		vb, okb := b[k]
		switch {
		case oka && okb && va == vb:
		case strings.HasSuffix(k, ".token"):
			out = append(out, k+": changed")
		case !oka:
			out = append(out, fmt.Sprintf("%s: added %s", k, vb))
		case !okb:
			out = append(out, fmt.Sprintf("%s: removed (was %s)", k, va))
		default:
			out = append(out, fmt.Sprintf("%s: %s → %s", k, va, vb))
		}
	}
	sort.Strings(out)
	return out
}

func flatten(c *Config) map[string]string {
	m := map[string]string{}
	if c == nil {
		return m
	}
	m["tools.default"] = c.Tools.Default
	for k, v := range c.Tools.Rules {
		m["tools.rules."+k] = v
	}
	m["masking.mask_from"] = c.Masking.MaskFrom
	for k, v := range c.Masking.OnNetworkTool {
		m["masking.on_network_tool."+k] = v
	}
	m["vault.env_files"] = strings.Join(c.Vault.EnvFiles, ", ")
	for k, v := range c.Vault.Overrides {
		m["vault.overrides."+k] = v
	}
	m["budgets.default_tokens"] = fmt.Sprint(c.Budgets.DefaultTokens)
	for k, v := range c.Budgets.Agents {
		m["budgets.agents."+k] = fmt.Sprint(v)
	}
	m["models.allow"] = "[" + strings.Join(c.Models.Allow, ", ") + "]"
	m["llm_hosts.approved"] = "[" + strings.Join(c.LLMHosts.Approved, ", ") + "]"
	m["injection.alert_threshold"] = fmt.Sprint(*c.Injection.AlertThreshold)
	for _, n := range c.Nodes {
		m["nodes."+n.ID+".owner"] = n.Owner
		m["nodes."+n.ID+".token"] = n.Token
	}
	m["signatures.feeds"] = "[" + strings.Join(c.Signatures.Feeds, ", ") + "]"
	m["signatures.refresh_seconds"] = fmt.Sprint(c.Signatures.RefreshSeconds)
	m["signatures.disabled"] = "[" + strings.Join(c.Signatures.Disabled, ", ") + "]"
	m["bash_guard.tools"] = "[" + strings.Join(c.BashGuard.Tools, ", ") + "]"
	m["bash_guard.network_commands"] = "[" + strings.Join(c.BashGuard.NetworkCommands, ", ") + "]"
	aa := c.Review.AutoAccept
	m["review.auto_accept.enabled"] = fmt.Sprint(aa.Enabled)
	m["review.auto_accept.min_probability"] = fmt.Sprint(*aa.MinProbability)
	m["review.auto_accept.min_confidence"] = fmt.Sprint(*aa.MinConfidence)
	m["review.auto_accept.tools"] = "[" + strings.Join(aa.Tools, ", ") + "]"
	m["review.auto_accept.variables"] = "[" + strings.Join(aa.Variables, ", ") + "]"
	return m
}

func oneOf(s string, xs []string) bool {
	for _, x := range xs {
		if s == x {
			return true
		}
	}
	return false
}

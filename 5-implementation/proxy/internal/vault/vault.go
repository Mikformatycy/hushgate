// Package vault replaces sensitive values with stable placeholders before
// they leave the machine and swaps them back for approved local sinks.
package vault

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
)

var tokenRe = regexp.MustCompile(`\{\{VAULT_[A-Z0-9_]+\}\}`)

type entry struct {
	Name    string
	Tier    Tier
	Token   string
	raw     string
	escaped string // JSON-string-escaped form, without quotes
}

// Ref describes a placeholder found in some text. It never carries the value.
type Ref struct {
	Name  string
	Tier  Tier
	Token string
}

type Vault struct {
	maskFrom Tier

	mu      sync.RWMutex
	byToken map[string]*entry
	env     map[string]*entry // every loaded variable, masked or not
	vars    []Var             // load order, with the deciding rule
	ordered []*entry          // env entries to mask, longest value first
}

// New creates a vault that masks values at or above maskFrom.
func New(maskFrom Tier) *Vault {
	return &Vault{maskFrom: maskFrom, byToken: map[string]*entry{}, env: map[string]*entry{}}
}

// Add registers a named value. Only values at or above the mask threshold
// are masked; the rest are kept so a later SetTier can promote them.
func (v *Vault) Add(name, value string, tier Tier) {
	v.add(name, value, tier, "registered directly")
}

func (v *Vault) add(name, value string, tier Tier, reason string) {
	e := &entry{
		Name:    name,
		Tier:    tier,
		Token:   "{{VAULT_ENV_" + tokenName(name) + "}}",
		raw:     value,
		escaped: jsonEscape(value),
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if _, exists := v.env[name]; !exists {
		v.vars = append(v.vars, Var{Name: name})
	}
	for i := range v.vars {
		if v.vars[i].Name == name {
			v.vars[i].Tier, v.vars[i].Reason = tier, reason
		}
	}
	v.env[name] = e
	v.byToken[e.Token] = e
	v.rebuild()
}

// Vars lists loaded variables with their current tier and deciding rule.
func (v *Vault) Vars() []Var {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return append([]Var(nil), v.vars...)
}

// Value returns a loaded variable's raw value, for in-process use only (the
// review queue derives a shape from it). Never send it anywhere.
func (v *Vault) Value(name string) (string, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	e, ok := v.env[name]
	if !ok {
		return "", false
	}
	return e.raw, true
}

// MaskFrom is the lowest tier that gets masked.
func (v *Vault) MaskFrom() Tier {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.maskFrom
}

// SetMaskFrom changes the masking threshold at runtime (live policy reload).
func (v *Vault) SetMaskFrom(t Tier) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.maskFrom = t
	v.rebuild()
}

// LoadEnvFiles replaces every loaded variable with the contents of files,
// applying overrides (name -> tier) on top of the rules. Detected values
// (API keys, IBANs found in traffic) are kept.
func (v *Vault) LoadEnvFiles(files []string, overrides map[string]Tier) ([]Var, error) {
	type pv struct {
		v     Var
		value string
	}
	var parsed []pv
	for _, f := range files {
		vars, values, err := parseEnvFile(f)
		if err != nil {
			return nil, err
		}
		for i, x := range vars {
			if t, ok := overrides[x.Name]; ok {
				x.Tier, x.Reason = t, "set in hushgate.yaml"
			}
			parsed = append(parsed, pv{x, values[i]})
		}
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, e := range v.env {
		delete(v.byToken, e.Token)
	}
	v.env, v.vars = map[string]*entry{}, nil
	out := make([]Var, 0, len(parsed))
	for _, p := range parsed {
		e := &entry{Name: p.v.Name, Tier: p.v.Tier, Token: "{{VAULT_ENV_" + tokenName(p.v.Name) + "}}",
			raw: p.value, escaped: jsonEscape(p.value)}
		if _, dup := v.env[p.v.Name]; !dup {
			v.vars = append(v.vars, p.v)
		}
		v.env[p.v.Name] = e
		v.byToken[e.Token] = e
		out = append(out, p.v)
	}
	v.rebuild()
	return out, nil
}

// rebuild recomputes the mask list; caller holds v.mu.
func (v *Vault) rebuild() {
	v.ordered = v.ordered[:0]
	for _, e := range v.env {
		if e.Tier >= v.maskFrom && len(e.raw) >= 4 {
			v.ordered = append(v.ordered, e)
		}
	}
	sort.Slice(v.ordered, func(i, j int) bool { return len(v.ordered[i].raw) > len(v.ordered[j].raw) })
}

// Mask replaces every known or detected secret in a JSON body with its
// placeholder and returns the refs that were applied.
func (v *Vault) Mask(body []byte) ([]byte, []Ref) {
	var refs []Ref
	v.mu.RLock()
	for _, e := range v.ordered {
		hit := false
		for _, form := range []string{e.escaped, e.raw} {
			if bytes.Contains(body, []byte(form)) {
				body = bytes.ReplaceAll(body, []byte(form), []byte(e.Token))
				hit = true
			}
		}
		if hit {
			refs = append(refs, e.ref())
		}
	}
	v.mu.RUnlock()

	for i := range Detectors {
		d := &Detectors[i]
		body = replaceRanges(body, d.find(body), func(m []byte) []byte {
			if tokenRe.Match(m) {
				return m
			}
			e := v.dynamic(d, string(m))
			refs = append(refs, e.ref())
			return []byte(e.Token)
		})
	}
	return body, refs
}

// Tokens lists the known placeholders present in s.
func (v *Vault) Tokens(s string) []Ref {
	var refs []Ref
	v.mu.RLock()
	defer v.mu.RUnlock()
	for _, tok := range tokenRe.FindAllString(s, -1) {
		if e, ok := v.byToken[tok]; ok {
			refs = append(refs, e.ref())
		}
	}
	return refs
}

// RehydrateJSON swaps placeholders inside JSON text back to real values.
func (v *Vault) RehydrateJSON(s string) string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return tokenRe.ReplaceAllStringFunc(s, func(tok string) string {
		if e, ok := v.byToken[tok]; ok {
			return e.escaped
		}
		return tok
	})
}

// dynamic registers a detector match. matched is already JSON-escaped since
// it was found in a JSON body. Tokens are hash-based so they stay stable
// across requests (keeps prompt caching intact).
func (v *Vault) dynamic(d *Detector, matched string) *entry {
	sum := sha256.Sum256([]byte(matched))
	tok := "{{VAULT_" + d.Name + "_" + strings.ToUpper(hex.EncodeToString(sum[:4])) + "}}"
	v.mu.Lock()
	defer v.mu.Unlock()
	if e, ok := v.byToken[tok]; ok {
		return e
	}
	e := &entry{Name: d.Name, Tier: d.Tier, Token: tok, raw: matched, escaped: matched}
	v.byToken[tok] = e
	return e
}

func (e *entry) ref() Ref { return Ref{Name: e.Name, Tier: e.Tier, Token: e.Token} }

// Var is the loader's view of a variable, safe to log (no value).
type Var struct {
	Name   string
	Tier   Tier
	Reason string
}

// LoadEnvFile reads KEY=VALUE lines, classifies them and adds them to the
// vault. A trailing "# @class: C3" comment overrides classification.
func (v *Vault) LoadEnvFile(path string) ([]Var, error) {
	vars, values, err := parseEnvFile(path)
	if err != nil {
		return nil, err
	}
	for i, x := range vars {
		v.add(x.Name, values[i], x.Tier, x.Reason)
	}
	return vars, nil
}

// parseEnvFile classifies each KEY=VALUE line; values[i] belongs to vars[i].
func parseEnvFile(path string) (vars []Var, values []string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		name, rest, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		name = strings.TrimSpace(name)
		value, comment := splitValue(strings.TrimSpace(rest))
		tier, reason := ClassifyWhy(name, value)
		if _, ann, ok := strings.Cut(comment, "@class:"); ok {
			if t, ok := ParseTier(ann); ok {
				tier, reason = t, "annotated @class: "+t.String()+" in .env"
			}
		}
		vars = append(vars, Var{Name: name, Tier: tier, Reason: reason})
		values = append(values, value)
	}
	return vars, values, sc.Err()
}

func splitValue(s string) (value, comment string) {
	if len(s) > 0 && s[0] == '\'' {
		if end := strings.IndexByte(s[1:], '\''); end >= 0 {
			return s[1 : end+1], s[end+2:]
		}
	}
	if len(s) > 0 && s[0] == '"' {
		var b strings.Builder
		for i := 1; i < len(s); i++ {
			switch {
			case s[i] == '\\' && i+1 < len(s):
				i++
				if s[i] == 'n' {
					b.WriteByte('\n')
				} else {
					b.WriteByte(s[i])
				}
			case s[i] == '"':
				return b.String(), s[i+1:]
			default:
				b.WriteByte(s[i])
			}
		}
	}
	if i := strings.Index(s, " #"); i >= 0 {
		return strings.TrimSpace(s[:i]), s[i:]
	}
	return s, ""
}

func tokenName(name string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return r - 32
		case (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9'):
			return r
		}
		return '_'
	}, name)
}

func jsonEscape(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(s)
	out := strings.TrimSuffix(buf.String(), "\n")
	return out[1 : len(out)-1]
}

// replaceRanges replaces each [start, end) range of src (sorted, disjoint).
func replaceRanges(src []byte, ranges [][2]int, fn func([]byte) []byte) []byte {
	if len(ranges) == 0 {
		return src
	}
	var out bytes.Buffer
	last := 0
	for _, r := range ranges {
		out.Write(src[last:r[0]])
		out.Write(fn(src[r[0]:r[1]]))
		last = r[1]
	}
	out.Write(src[last:])
	return out.Bytes()
}

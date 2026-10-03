package vault

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Tier is the confidentiality class of a value. Higher is more sensitive.
type Tier int

const (
	Public       Tier = iota // C0: ports, flags, log levels
	Internal                 // C1: hostnames, IDs, bucket names
	Confidential             // C2: PII, unknown values (fail closed)
	Secret                   // C3: credentials, keys, tokens
)

func (t Tier) String() string { return "C" + strconv.Itoa(int(t)) }

// ParseTier accepts "C0".."C3" (case-insensitive).
func ParseTier(s string) (Tier, bool) {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) == 2 && s[0] == 'C' && s[1] >= '0' && s[1] <= '3' {
		return Tier(s[1] - '0'), true
	}
	return 0, false
}

var (
	secretSegments = set("KEY", "SECRET", "TOKEN", "PASSWORD", "PASS", "PASSWD", "PWD",
		"CREDENTIAL", "CREDENTIALS", "PRIVATE", "DSN", "AUTH", "APIKEY", "SALT", "SIGNATURE")
	secretSubstrings = []string{"PASSWORD", "SECRET", "TOKEN", "APIKEY"}
	internalSegments = set("HOST", "HOSTNAME", "URL", "URI", "ENDPOINT", "ID", "BUCKET", "REGION",
		"DOMAIN", "ADDR", "ADDRESS", "USER", "USERNAME", "EMAIL", "ACCOUNT", "PROJECT", "CLUSTER", "NAMESPACE")
	publicSegments = set("PORT", "ENV", "LOG", "LEVEL", "DEBUG", "MODE", "VERSION", "TIMEOUT",
		"WORKERS", "TZ", "LANG", "VERBOSE")
	booleans = set("TRUE", "FALSE", "YES", "NO", "ON", "OFF")
)

// Classify assigns a tier from the variable name and value.
func Classify(name, value string) Tier {
	t, _ := ClassifyWhy(name, value)
	return t
}

// ClassifyWhy also returns the rule that decided, for the audit trail. Rules
// run in precedence order; explicit annotations are handled by the .env loader.
func ClassifyWhy(name, value string) (Tier, string) {
	v := strings.TrimSpace(value)
	d, detected := detectValue(v)
	if detected && d.Tier == Secret {
		return Secret, "value matches " + d.Name + " format (" + d.Check + ")"
	}
	if v == "" || booleans[strings.ToUpper(v)] {
		return Public, "empty or boolean value"
	}
	if word, ok := secretWord(name); ok {
		return Secret, "name contains " + word
	}
	if detected {
		return d.Tier, "value is a valid " + d.Name + " (" + d.Check + ")"
	}
	if isNumeric(v) || len(v) < 4 {
		return Public, "numeric or very short value"
	}
	segs := segments(name)
	for _, s := range segs {
		if internalSegments[s] {
			return Internal, "name contains " + s
		}
	}
	for _, s := range segs {
		if publicSegments[s] {
			return Public, "name contains " + s
		}
	}
	if h := entropy(v); len(v) >= 16 && h >= 3.5 {
		return Secret, fmt.Sprintf("random-looking value (%.1f bits/char entropy)", h)
	}
	return Confidential, FallbackReason
}

// FallbackReason marks variables no rule recognized; they are masked by
// default and are the candidates for human (or AI-assisted) review.
const FallbackReason = "no rule matched, masked by default"

func secretWord(name string) (string, bool) {
	for _, s := range segments(name) {
		if secretSegments[s] {
			return s, true
		}
	}
	up := strings.ToUpper(name)
	for _, sub := range secretSubstrings {
		if strings.Contains(up, sub) {
			return sub, true
		}
	}
	return "", false
}

func segments(name string) []string {
	return strings.FieldsFunc(strings.ToUpper(name), func(r rune) bool { return r == '_' || r == '-' || r == '.' })
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}

// entropy returns Shannon entropy in bits per character.
func entropy(s string) float64 {
	counts := map[rune]int{}
	n := 0
	for _, r := range s {
		counts[r]++
		n++
	}
	var h float64
	for _, c := range counts {
		p := float64(c) / float64(n)
		h -= p * math.Log2(p)
	}
	return h
}

func set(items ...string) map[string]bool {
	m := make(map[string]bool, len(items))
	for _, it := range items {
		m[it] = true
	}
	return m
}

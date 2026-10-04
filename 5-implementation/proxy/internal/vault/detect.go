package vault

import (
	"bytes"
	"math/big"
	"regexp"
	"strings"
)

// Detector finds a kind of sensitive value in free text. Group selects the
// capture group to mask (0 = whole match). Valid, if set, returns the checked
// prefix of the match to mask; checksums keep false positives near zero.
type Detector struct {
	Name  string
	Tier  Tier
	Check string // human-readable validation, shown in the dashboard
	re    *regexp.Regexp
	group int
	valid func(string) (string, bool)
	// hint, if set, finds the windows where a match could be, with a cheap byte
	// scan, so the regular expression runs on a few short windows instead of the
	// whole request. Patterns without a fixed prefix are slow over large bodies.
	hint func(s []byte) [][2]int
}

// find returns [start, end) ranges of validated matches in s.
func (d *Detector) find(s []byte) [][2]int {
	if d.hint == nil {
		return d.findIn(s, 0)
	}
	var out [][2]int
	for _, w := range d.hint(s) {
		out = append(out, d.findIn(s[w[0]:w[1]], w[0])...)
	}
	return out
}

// findIn matches within s, which starts at offset off of the whole text.
func (d *Detector) findIn(s []byte, off int) [][2]int {
	var out [][2]int
	for _, m := range d.re.FindAllSubmatchIndex(s, -1) {
		start, end := m[2*d.group], m[2*d.group+1]
		if d.valid != nil {
			sub, ok := d.valid(string(s[start:end]))
			if !ok {
				continue
			}
			end = start + len(sub)
		}
		out = append(out, [2]int{off + start, off + end})
	}
	return out
}

// Hints. Each window covers the whole run its pattern could consume and the
// character before it (or the start of the text), so matching inside the
// windows gives exactly the matches the full text would.

func isDigit(c byte) bool      { return c >= '0' && c <= '9' }
func isUpper(c byte) bool      { return c >= 'A' && c <= 'Z' }
func isUpperAlnum(c byte) bool { return isUpper(c) || isDigit(c) }
func isAlnum(c byte) bool      { return isUpperAlnum(c) || (c >= 'a' && c <= 'z') }
func isSpace(c byte) bool      { return c == ' ' || c == '\t' || c == '\n' || c == '\f' || c == '\r' }

// digitRuns finds runs of digits, optionally joined by single separators from
// seps, that hold at least min digits.
func digitRuns(seps string, min int) func([]byte) [][2]int {
	return func(s []byte) [][2]int {
		var out [][2]int
		for i := 0; i < len(s); {
			if !isDigit(s[i]) {
				i++
				continue
			}
			start, n, j := i, 0, i
			for j < len(s) {
				if isDigit(s[j]) {
					n, j = n+1, j+1
				} else if seps != "" && strings.IndexByte(seps, s[j]) >= 0 && j+1 < len(s) && isDigit(s[j+1]) {
					j++
				} else {
					break
				}
			}
			if n >= min {
				out = append(out, [2]int{max(start-1, 0), j})
			}
			i = j
		}
		return out
	}
}

// ibanHint finds two capitals and two digits not preceded by a capital or digit.
// An IBAN match is at most 43 characters, so 64 covers it.
func ibanHint(s []byte) [][2]int {
	var out [][2]int
	for i := 0; i+4 <= len(s); i++ {
		if !isUpper(s[i]) || !isUpper(s[i+1]) || !isDigit(s[i+2]) || !isDigit(s[i+3]) || (i > 0 && isUpperAlnum(s[i-1])) {
			continue
		}
		end := i
		for end < len(s) && end < i+64 && (isUpperAlnum(s[end]) || s[end] == ' ') {
			end++
		}
		out = append(out, [2]int{max(i-1, 0), end})
	}
	return merge(out)
}

// stripeHint finds "k_live_" after an s or r, through the key that follows.
func stripeHint(s []byte) [][2]int {
	var out [][2]int
	for i := 0; ; {
		k := bytes.Index(s[i:], []byte("k_live_"))
		if k < 0 {
			return merge(out)
		}
		at := i + k
		if at > 0 && (s[at-1] == 's' || s[at-1] == 'r') {
			end := at + len("k_live_")
			for end < len(s) && isAlnum(s[end]) {
				end++
			}
			out = append(out, [2]int{at - 1, end})
		}
		i = at + 1
	}
}

// nipKeywordHint finds "NIP" in any case, then separators and the digit run.
func nipKeywordHint(s []byte) [][2]int {
	var out [][2]int
	for i := 0; i+3 <= len(s); i++ {
		if s[i]|0x20 != 'n' || s[i+1]|0x20 != 'i' || s[i+2]|0x20 != 'p' {
			continue
		}
		end := i + 3
		for end < len(s) && (s[end] == ':' || isSpace(s[end])) {
			end++
		}
		for end < len(s) && isDigit(s[end]) {
			end++
		}
		out = append(out, [2]int{i, end})
	}
	return merge(out)
}

// merge joins overlapping windows (they come sorted by start).
func merge(ws [][2]int) [][2]int {
	var out [][2]int
	for _, w := range ws {
		if n := len(out); n > 0 && w[0] < out[n-1][1] {
			out[n-1][1] = max(out[n-1][1], w[1])
			continue
		}
		out = append(out, w)
	}
	return out
}

// shrink tries s, then s cut at each space or dash from the right, so a
// pattern that ran into following text still finds the real value.
func shrink(check func(string) bool) func(string) (string, bool) {
	return func(s string) (string, bool) {
		for {
			if check(s) {
				return s, true
			}
			i := strings.LastIndexAny(s, " -")
			if i <= 0 {
				return "", false
			}
			s = s[:i]
		}
	}
}

func exact(check func(string) bool) func(string) (string, bool) {
	return func(s string) (string, bool) { return s, check(s) }
}

// Secret formats first, then personal data. Digit-based patterns use an explicit
// leading non-digit instead of \b, because JSON escapes like "\n" put a word
// character right before the number. They capture the whole digit run and let
// the validator check the length, so nothing after the match is consumed.
var Detectors = []Detector{
	{Name: "AWS_KEY", Tier: Secret, Check: "AKIA prefix", re: regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{Name: "ANTHROPIC_KEY", Tier: Secret, Check: "sk-ant- prefix", re: regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{20,}`)},
	{Name: "OPENAI_KEY", Tier: Secret, Check: "sk- prefix, 32+ chars", re: regexp.MustCompile(`sk-(?:proj-)?[A-Za-z0-9_\-]{32,}`)},
	{Name: "STRIPE_KEY", Tier: Secret, Check: "sk_live_/rk_live_ prefix", re: regexp.MustCompile(`[sr]k_live_[A-Za-z0-9]{20,}`), hint: stripeHint},
	{Name: "GITHUB_TOKEN", Tier: Secret, Check: "gh?_ prefix", re: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
	{Name: "SLACK_TOKEN", Tier: Secret, Check: "xox?- prefix", re: regexp.MustCompile(`xox[baprs]-[A-Za-z0-9\-]{10,}`)},
	{Name: "JWT", Tier: Secret, Check: "three base64url segments", re: regexp.MustCompile(`eyJ[\w-]{10,}\.eyJ[\w-]{10,}\.[\w-]{10,}`)},
	{Name: "PRIVATE_KEY", Tier: Secret, Check: "PEM block", re: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{Name: "URL_PASSWORD", Tier: Secret, Check: "password in scheme://user:pass@host", group: 1,
		re: regexp.MustCompile(`://[^:/\s"@]+:([^@\s"/]+)@`)},

	{Name: "IBAN", Tier: Confidential, Check: "ISO 13616 mod-97 checksum", group: 1, valid: shrink(validIBAN), hint: ibanHint,
		re: regexp.MustCompile(`(?:^|[^A-Z0-9])([A-Z]{2}\d{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,3})?)`)},
	{Name: "CARD", Tier: Confidential, Check: "Visa/Mastercard/Amex prefix + Luhn checksum", group: 1, valid: shrink(validCard), hint: digitRuns(" -", 13),
		re: regexp.MustCompile(`(?:^|[^\d])((?:4|5[1-5]|2[2-7]|3[47])\d(?:[ -]?\d){11,17})`)},
	{Name: "PESEL", Tier: Confidential, Check: "PL national ID: birth date + check digit", group: 1, valid: exact(validPESEL), hint: digitRuns("", 11),
		re: regexp.MustCompile(`(?:^|[^\d])(\d{11,})`)}, // whole digit run; validator wants exactly 11
	{Name: "NIP", Tier: Confidential, Check: "PL tax ID: mod-11 check digit", group: 1, valid: exact(validNIP), hint: digitRuns("-", 10),
		re: regexp.MustCompile(`(?:^|[^\d])(\d{3}-\d{3}-\d{2}-\d{2}|\d{3}-\d{2}-\d{2}-\d{3})`)},
	{Name: "NIP", Tier: Confidential, Check: "PL tax ID: mod-11 check digit", group: 1, valid: exact(validNIP), hint: nipKeywordHint,
		re: regexp.MustCompile(`(?i)NIP[:\s]*(\d{10,})`)},
}

// detectValue reports the first detector that finds something in the value.
func detectValue(v string) (*Detector, bool) {
	for i := range Detectors {
		if len(Detectors[i].find([]byte(v))) > 0 {
			return &Detectors[i], true
		}
	}
	return nil, false
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= '0' && r <= '9' {
			return r
		}
		return -1
	}, s)
}

func validIBAN(s string) bool {
	s = strings.ReplaceAll(s, " ", "")
	if len(s) < 15 || len(s) > 34 {
		return false
	}
	rearranged := s[4:] + s[:4]
	var num strings.Builder
	for _, r := range rearranged {
		switch {
		case r >= '0' && r <= '9':
			num.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			num.WriteString(big.NewInt(int64(r - 'A' + 10)).String())
		default:
			return false
		}
	}
	n, ok := new(big.Int).SetString(num.String(), 10)
	return ok && new(big.Int).Mod(n, big.NewInt(97)).Int64() == 1
}

func validCard(s string) bool {
	d := digits(s)
	if len(d) < 13 || len(d) > 19 {
		return false
	}
	sum := 0
	for i := len(d) - 1; i >= 0; i-- {
		n := int(d[i] - '0')
		if (len(d)-i)%2 == 0 {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}

func validPESEL(s string) bool {
	if len(s) != 11 {
		return false
	}
	w := []int{1, 3, 7, 9, 1, 3, 7, 9, 1, 3}
	sum := 0
	for i, wi := range w {
		sum += int(s[i]-'0') * wi
	}
	if (10-sum%10)%10 != int(s[10]-'0') {
		return false
	}
	// Month encodes the century (+20 for 2000s, +80 for 1800s ...).
	month := int(s[2]-'0')*10 + int(s[3]-'0')
	day := int(s[4]-'0')*10 + int(s[5]-'0')
	return month%20 >= 1 && month%20 <= 12 && day >= 1 && day <= 31
}

func validNIP(s string) bool {
	d := digits(s)
	if len(d) != 10 {
		return false
	}
	w := []int{6, 5, 7, 2, 3, 4, 5, 6, 7}
	sum := 0
	for i, wi := range w {
		sum += int(d[i]-'0') * wi
	}
	return sum%11 != 10 && sum%11 == int(d[9]-'0')
}

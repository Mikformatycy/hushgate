package vault

import (
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
}

// find returns [start, end) ranges of validated matches in s.
func (d *Detector) find(s []byte) [][2]int {
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
		out = append(out, [2]int{start, end})
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
	{Name: "STRIPE_KEY", Tier: Secret, Check: "sk_live_/rk_live_ prefix", re: regexp.MustCompile(`[sr]k_live_[A-Za-z0-9]{20,}`)},
	{Name: "GITHUB_TOKEN", Tier: Secret, Check: "gh?_ prefix", re: regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`)},
	{Name: "SLACK_TOKEN", Tier: Secret, Check: "xox?- prefix", re: regexp.MustCompile(`xox[baprs]-[A-Za-z0-9\-]{10,}`)},
	{Name: "JWT", Tier: Secret, Check: "three base64url segments", re: regexp.MustCompile(`eyJ[\w-]{10,}\.eyJ[\w-]{10,}\.[\w-]{10,}`)},
	{Name: "PRIVATE_KEY", Tier: Secret, Check: "PEM block", re: regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`)},
	{Name: "URL_PASSWORD", Tier: Secret, Check: "password in scheme://user:pass@host", group: 1,
		re: regexp.MustCompile(`://[^:/\s"@]+:([^@\s"/]+)@`)},

	{Name: "IBAN", Tier: Confidential, Check: "ISO 13616 mod-97 checksum", group: 1, valid: shrink(validIBAN),
		re: regexp.MustCompile(`(?:^|[^A-Z0-9])([A-Z]{2}\d{2}(?: ?[A-Z0-9]{4}){2,7}(?: ?[A-Z0-9]{1,3})?)`)},
	{Name: "CARD", Tier: Confidential, Check: "Visa/Mastercard/Amex prefix + Luhn checksum", group: 1, valid: shrink(validCard),
		re: regexp.MustCompile(`(?:^|[^\d])((?:4|5[1-5]|2[2-7]|3[47])\d(?:[ -]?\d){11,17})`)},
	{Name: "PESEL", Tier: Confidential, Check: "PL national ID: birth date + check digit", group: 1, valid: exact(validPESEL),
		re: regexp.MustCompile(`(?:^|[^\d])(\d{11,})`)}, // whole digit run; validator wants exactly 11
	{Name: "NIP", Tier: Confidential, Check: "PL tax ID: mod-11 check digit", group: 1, valid: exact(validNIP),
		re: regexp.MustCompile(`(?:^|[^\d])(\d{3}-\d{3}-\d{2}-\d{2}|\d{3}-\d{2}-\d{2}-\d{3})`)},
	{Name: "NIP", Tier: Confidential, Check: "PL tax ID: mod-11 check digit", group: 1, valid: exact(validNIP),
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

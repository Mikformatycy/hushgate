package vault

import "regexp"

// Known secret formats. Matches anywhere in a request are masked even if the
// value never appeared in a loaded .env file.
var detectors = []*regexp.Regexp{
	regexp.MustCompile(`AKIA[0-9A-Z]{16}`),                         // AWS access key
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_\-]{20,}`),               // Anthropic
	regexp.MustCompile(`sk-(?:proj-)?[A-Za-z0-9_\-]{32,}`),         // OpenAI
	regexp.MustCompile(`[sr]k_live_[A-Za-z0-9]{20,}`),              // Stripe
	regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{36,}`),               // GitHub
	regexp.MustCompile(`xox[baprs]-[A-Za-z0-9\-]{10,}`),            // Slack
	regexp.MustCompile(`eyJ[\w-]{10,}\.eyJ[\w-]{10,}\.[\w-]{10,}`), // JWT
	regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----`),
}

// dsnPassword captures only the password part of scheme://user:pass@host.
var dsnPassword = regexp.MustCompile(`://[^:/\s"@]+:([^@\s"/]+)@`)

func matchesKnownFormat(v string) bool {
	for _, re := range detectors {
		if re.MatchString(v) {
			return true
		}
	}
	return dsnPassword.MatchString(v)
}

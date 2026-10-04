package vault

import (
	"strings"
	"testing"
)

// Every secret format the detectors know is masked, even when no .env lists it.
// Samples are assembled at run time, so the source holds no secret-shaped strings.
func TestSecretFormatsDetected(t *testing.T) {
	r := strings.Repeat
	samples := map[string]string{
		"AWS_KEY":       "AKIA" + "IOSFODNN7EXAMPLE",
		"ANTHROPIC_KEY": "sk-" + "ant-api03-" + r("aB3", 10),
		"OPENAI_KEY":    "sk-" + "proj-" + r("Xy9_", 10),
		"STRIPE_KEY":    "sk_" + "live_" + r("Zq7", 9),
		"GITHUB_TOKEN":  "gh" + "p_" + r("Ab1", 13),
		"SLACK_TOKEN":   "xo" + "xb-" + r("12345-", 3),
		"JWT":           "eyJ" + r("hbGciOi", 2) + ".eyJ" + r("zdWIiOi", 2) + "." + r("SflKxw", 3),
		"PRIVATE_KEY":   "-----BEGIN " + "RSA PRIVATE KEY-----\\n" + r("MIIEow", 5) + "\\n-----END RSA PRIVATE KEY-----",
		"URL_PASSWORD":  "postgres://app:" + "hunter2pass@db.internal/x",
	}
	for name, secret := range samples {
		v := New(Confidential)
		masked, refs := v.Mask([]byte(`{"content":"see ` + secret + ` here"}`))
		s := string(masked)
		if len(refs) == 0 || !strings.Contains(s, "{{VAULT_"+name+"_") {
			t.Errorf("%s not masked: %s", name, s)
		}
		if name != "URL_PASSWORD" && strings.Contains(s, secret) {
			t.Errorf("%s leaked: %s", name, s)
		}
	}
}

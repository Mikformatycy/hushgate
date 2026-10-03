package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name, value string
		want        Tier
	}{
		{"PORT", "3000", Public},
		{"DEBUG", "false", Public},
		{"ENABLE_TOKEN_CACHE", "true", Public},
		{"LOG_LEVEL", "info", Public},
		{"API_HOST", "internal.corp.example", Internal},
		{"S3_BUCKET", "acme-reports", Internal},
		{"DB_PASSWORD", "hunter2", Secret},
		{"PIN_PASSWORD", "1234", Secret},
		{"STRIPE_SECRET_KEY", "whatever-value", Secret},
		{"DATABASE_URL", "postgres://app:s3cr3t@db:5432/app", Secret},
		{"SOMETHING", "AKIAIOSFODNN7EXAMPLE", Secret},
		{"BLOB", "q8Zr2Lm9Xw4Tn7Vb1Kc6", Secret},
		{"CUSTOMER_NAME", "Jane Doe", Confidential},
	}
	for _, c := range cases {
		if got := Classify(c.name, c.value); got != c.want {
			t.Errorf("Classify(%s=%s) = %s, want %s", c.name, c.value, got, c.want)
		}
	}
}

func TestLoadMaskRehydrate(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	os.WriteFile(path, []byte(`
# comment
PORT=3000
DB_PASSWORD="pa\"ss word"
NOTE=plain-value # @class: C3
API_HOST=internal.corp.example
`), 0o600)

	v := New(Confidential)
	vars, err := v.LoadEnvFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(vars) != 4 {
		t.Fatalf("got %d vars", len(vars))
	}

	body := []byte(`{"messages":[{"role":"user","content":"pw is pa\"ss word, note plain-value, host internal.corp.example, port 3000"}]}`)
	masked, refs := v.Mask(body)
	s := string(masked)
	if strings.Contains(s, `pa\"ss`) || strings.Contains(s, "plain-value") {
		t.Fatalf("secret leaked: %s", s)
	}
	if !strings.Contains(s, "{{VAULT_ENV_DB_PASSWORD}}") || !strings.Contains(s, "{{VAULT_ENV_NOTE}}") {
		t.Fatalf("missing tokens: %s", s)
	}
	if !strings.Contains(s, "internal.corp.example") || !strings.Contains(s, "3000") {
		t.Fatalf("C0/C1 should pass through: %s", s)
	}
	if len(refs) != 2 {
		t.Fatalf("refs = %v", refs)
	}
	if back := v.RehydrateJSON(s); back != string(body) {
		t.Fatalf("rehydrate mismatch:\n%s\n%s", back, body)
	}
}

func TestDynamicDetection(t *testing.T) {
	v := New(Confidential)
	body := []byte(`{"content":"key AKIAIOSFODNN7EXAMPLE and postgres://u:topsecret@h/db"}`)
	masked, refs := v.Mask(body)
	s := string(masked)
	if strings.Contains(s, "AKIA") || strings.Contains(s, "topsecret") {
		t.Fatalf("leaked: %s", s)
	}
	if !strings.Contains(s, "postgres://u:{{VAULT_DYN_") {
		t.Fatalf("dsn user should stay, password masked: %s", s)
	}
	if len(refs) != 2 {
		t.Fatalf("refs = %v", refs)
	}
	again, _ := v.Mask(body)
	if string(again) != s {
		t.Fatal("dynamic tokens must be stable across requests")
	}
	if v.RehydrateJSON(s) != string(body) {
		t.Fatal("rehydrate mismatch")
	}
}

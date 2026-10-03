package admin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
	"github.com/Mikformatycy/hushgate/proxy/internal/budget"
	"github.com/Mikformatycy/hushgate/proxy/internal/config"
	"github.com/Mikformatycy/hushgate/proxy/internal/engine"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/review"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

// newAPI builds an API backed by a real policy file in a temp dir. The
// placeholder {{ENV}} in policyYAML is replaced by the path of a .env file
// holding envFile.
func newAPI(t *testing.T, policyYAML, envFile string) (*API, string) {
	t.Helper()
	dir := t.TempDir()
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte(envFile), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hushgate.yaml")
	src := []byte(strings.ReplaceAll(policyYAML, "{{ENV}}", envPath))
	if err := os.WriteFile(path, src, 0o644); err != nil {
		t.Fatal(err)
	}
	ring := audit.NewRing(100)
	eng := &engine.Engine{Policy: &policy.Policy{}, Vault: vault.New(vault.Confidential), Reviews: review.NewStore()}
	live := config.NewLive(path, eng.Apply, func(action, detail string) {
		ring.Record(audit.Event{Agent: "hushgate.yaml", Kind: "policy", Action: action, Reason: detail})
	})
	if err := live.Load(); err != nil {
		t.Fatal(err)
	}
	return &API{Token: "admin", AdvisorToken: "advisor", Events: ring, Budget: budget.NewMemory(), Audit: ring,
		Vault: eng.Vault, Policy: eng.Policy, Reviews: eng.Reviews, Live: live}, path
}

package gateway

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Mikformatycy/hushgate/proxy/internal/budget"
	"github.com/Mikformatycy/hushgate/proxy/internal/config"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/shell"
	"github.com/Mikformatycy/hushgate/proxy/internal/signature"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

// benchGateway is a full gateway (vault, policy, Bash guard, all signatures)
// in front of an in-process model that answers with the given stream.
func benchGateway(b *testing.B, stream string) (*Gateway, *httptest.Server) {
	up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "text/event-stream")
		io.WriteString(w, stream)
	}))
	b.Cleanup(up.Close)
	u, _ := url.Parse(up.URL)
	v := vault.New(vault.Confidential)
	v.Add("DB_PASSWORD", "Pr0d-Adm1n-2026", vault.Secret)
	v.Add("API_KEY", "sk_"+"live_51Hx9QaZ3kLmN0pQrStUvWxYz", vault.Secret)
	raw, err := os.ReadFile("../../../config/signatures.yaml")
	if err != nil {
		b.Fatal(err)
	}
	feed, _ := signature.ParseFeed(raw)
	sigs := &signature.Set{}
	sigs.Replace([]*signature.Feed{feed}, nil, nil)
	gw := &Gateway{
		Upstream: u, Client: up.Client(), Vault: v, Budget: budget.NewMemory(), Audit: &nopAudit{},
		Policy: &policy.Policy{Tools: map[string]policy.Sink{"Bash": policy.Local}, Default: policy.Deny},
		Guard:  shell.NewGuard([]string{"Bash"}, config.DefaultNetworkCommands), Signatures: sigs,
	}
	return gw, up
}

var benchBody = []byte(`{"model":"claude-sonnet","stream":true,"messages":[{"role":"user","content":"` +
	strings.Repeat("Deploy with DB_PASSWORD=Pr0d-Adm1n-2026 and refund IBAN PL61109010140000071219812874. ", 40) + `"}]}`)

func roundTrip(b *testing.B, h http.Handler) {
	req := httptest.NewRequest("POST", "/v1/messages", bytes.NewReader(benchBody))
	req.Header.Set("X-Agent-Id", "bench")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 200 {
		b.Fatalf("status %d: %s", rec.Code, rec.Body)
	}
}

// Baseline: the same request sent straight to the in-process model.
func BenchmarkDirectToModel(b *testing.B) {
	_, up := benchGateway(b, sse("Bash", `{"command":"go test ./..."}`))
	for i := 0; i < b.N; i++ {
		resp, err := up.Client().Post(up.URL+"/v1/messages", "application/json", bytes.NewReader(benchBody))
		if err != nil {
			b.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
	}
}

// Through the gate: mask the request, stream the answer, check its tool call
// against the policy, the Bash guard and all 19 signatures.
func BenchmarkThroughGate(b *testing.B) {
	gw, _ := benchGateway(b, sse("Bash", `{"command":"go test ./..."}`))
	for i := 0; i < b.N; i++ {
		roundTrip(b, gw)
	}
}

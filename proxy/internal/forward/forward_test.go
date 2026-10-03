package forward

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"io"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/gateway"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

type recorder struct{ events []audit.Event }

func (r *recorder) Record(e audit.Event) { r.events = append(r.events, e) }

func (r *recorder) kinds() string {
	var k []string
	for _, e := range r.events {
		k = append(k, e.Kind)
	}
	return strings.Join(k, ",")
}

// writeCA creates a throwaway company CA and returns its file paths and pool.
func writeCA(t *testing.T) (string, string, *x509.CertPool) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: "Test Corp CA"},
		NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign,
	}
	der, _ := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	dir := t.TempDir()
	certFile, keyFile := filepath.Join(dir, "ca.crt"), filepath.Join(dir, "ca.key")
	os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}), 0o600)
	os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0o600)
	cert, _ := x509.ParseCertificate(der)
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	return certFile, keyFile, pool
}

func setup(t *testing.T) (proxyURL *url.URL, upstream *httptest.Server, fp *Proxy, rec *recorder, upstreamBody *string) {
	var seen string
	upstream = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		seen = string(b)
		switch r.URL.Path {
		case "/v1/messages":
			w.Header().Set("Content-Type", "application/json")
			io.WriteString(w, `{"type":"message","content":[{"type":"text","text":"ok"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":5}}`)
		default:
			io.WriteString(w, "hello web")
		}
	}))
	t.Cleanup(upstream.Close)

	certFile, keyFile, _ := writeCA(t)
	fp, err := New(certFile, keyFile)
	if err != nil {
		t.Fatal(err)
	}
	upstreamPool := x509.NewCertPool()
	upstreamPool.AddCert(upstream.Certificate())
	transport := &http.Transport{TLSClientConfig: &tls.Config{RootCAs: upstreamPool}}

	v := vault.New(vault.Confidential)
	v.Add("DB_PASSWORD", "hunter2-real", vault.Secret)
	rec = &recorder{}
	fp.Nodes = &Registry{nodes: map[string]Node{"jdoe-laptop": {ID: "jdoe-laptop", Token: "s3cret"}}}
	fp.Transport, fp.Audit = transport, rec
	fp.Approved = map[string]bool{"127.0.0.1": true}
	fp.Gateway = &gateway.Gateway{Client: &http.Client{Transport: transport}, Vault: v, Audit: rec,
		Policy: &policy.Policy{Default: policy.Local}, Budget: budget.NewMemory()}

	srv := httptest.NewServer(fp)
	t.Cleanup(srv.Close)
	proxyURL, _ = url.Parse(srv.URL)
	return proxyURL, upstream, fp, rec, &seen
}

// client builds an "employee laptop": trusts the corp CA, goes through the proxy.
func client(t *testing.T, proxyURL *url.URL, fp *Proxy, user *url.Userinfo) *http.Client {
	pool := x509.NewCertPool()
	pool.AddCert(fp.ca.Leaf)
	u := *proxyURL
	u.User = user
	return &http.Client{Transport: &http.Transport{
		Proxy:           http.ProxyURL(&u),
		TLSClientConfig: &tls.Config{RootCAs: pool},
	}}
}

func call(t *testing.T, c *http.Client, method, url, body string) (int, string) {
	req, _ := http.NewRequest(method, url, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

const chat = `{"model":"m","messages":[{"role":"user","content":"pw hunter2-real"}]}`

func TestApprovedNodeGoesThroughGateway(t *testing.T) {
	proxyURL, up, fp, rec, seen := setup(t)
	c := client(t, proxyURL, fp, url.UserPassword("jdoe-laptop", "s3cret"))
	code, out := call(t, c, "POST", up.URL+"/v1/messages", chat)
	if code != 200 || !strings.Contains(out, `"ok"`) {
		t.Fatalf("%d %s", code, out)
	}
	if strings.Contains(*seen, "hunter2-real") || !strings.Contains(*seen, "{{VAULT_ENV_DB_PASSWORD}}") {
		t.Fatalf("upstream saw secret: %s", *seen)
	}
	if st, _ := fp.Gateway.Budget.Status(t.Context(), "jdoe-laptop"); st.Used != 15 {
		t.Fatalf("usage not attributed to node: %+v", st)
	}
	if rec.kinds() != "mask,usage" {
		t.Fatalf("events = %s", rec.kinds())
	}
}

func TestShadowAIBlocked(t *testing.T) {
	proxyURL, up, fp, rec, seen := setup(t)
	fp.Approved = map[string]bool{"api.anthropic.com": true}
	c := client(t, proxyURL, fp, url.UserPassword("jdoe-laptop", "s3cret"))
	code, out := call(t, c, "POST", up.URL+"/v1/chat/completions", chat)
	if code != 403 || !strings.Contains(out, "not an approved LLM provider") {
		t.Fatalf("%d %s", code, out)
	}
	if *seen != "" {
		t.Fatal("request reached the rogue host")
	}
	if rec.kinds() != "shadow_ai" || rec.events[0].Agent != "jdoe-laptop" {
		t.Fatalf("events = %+v", rec.events)
	}
}

func TestUnregisteredNodeBlocked(t *testing.T) {
	proxyURL, up, fp, rec, _ := setup(t)
	for _, user := range []*url.Userinfo{nil, url.UserPassword("jdoe-laptop", "wrong")} {
		c := client(t, proxyURL, fp, user)
		if code, _ := call(t, c, "POST", up.URL+"/v1/messages", chat); code != 403 {
			t.Fatalf("user %v: got %d", user, code)
		}
	}
	if rec.kinds() != "node_blocked,node_blocked" {
		t.Fatalf("events = %s", rec.kinds())
	}
}

func TestWebPassesThrough(t *testing.T) {
	proxyURL, up, fp, rec, _ := setup(t)
	c := client(t, proxyURL, fp, nil) // even unregistered devices can browse
	code, out := call(t, c, "GET", up.URL+"/news", "")
	if code != 200 || out != "hello web" {
		t.Fatalf("%d %s", code, out)
	}
	if len(rec.events) != 0 {
		t.Fatalf("web traffic should not be audited: %s", rec.kinds())
	}
}

func TestUntrustedDeviceFailsClosed(t *testing.T) {
	proxyURL, up, _, _, seen := setup(t)
	c := &http.Client{Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)}} // no corp CA
	if _, err := c.Post(up.URL+"/v1/messages", "application/json", strings.NewReader(chat)); err == nil {
		t.Fatal("expected TLS failure without the company CA")
	}
	if *seen != "" {
		t.Fatal("request reached upstream")
	}
}

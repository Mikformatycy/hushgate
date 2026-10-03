// Command gate runs the Provenance Gate proxy.
package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/admin"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/forward"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/gateway"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/review"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/scan"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

func main() {
	upstream, err := url.Parse(env("UPSTREAM_URL", "https://api.anthropic.com"))
	if err != nil {
		log.Fatalf("UPSTREAM_URL: %v", err)
	}
	maskFrom, ok := vault.ParseTier(env("MASK_FROM_TIER", "C2"))
	if !ok {
		log.Fatal("MASK_FROM_TIER must be C0..C3")
	}
	limit, err := strconv.ParseInt(env("TOKEN_LIMIT", "0"), 10, 64)
	if err != nil {
		log.Fatalf("TOKEN_LIMIT: %v", err)
	}

	v := vault.New(maskFrom)
	reviews := review.NewStore()
	scans := scan.NewStore(500)
	injectionAt, err := strconv.ParseFloat(env("INJECTION_ALERT_THRESHOLD", "0.8"), 64)
	if err != nil {
		log.Fatalf("INJECTION_ALERT_THRESHOLD: %v", err)
	}
	for _, path := range strings.Split(os.Getenv("VAULT_ENV_FILES"), ",") {
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		vars, err := v.LoadEnvFile(path)
		if err != nil {
			log.Fatalf("load %s: %v", path, err)
		}
		for _, x := range vars {
			log.Printf("vault: %s %s (%s)", x.Tier, x.Name, x.Reason)
			if x.Reason == vault.FallbackReason {
				val, _ := v.Value(x.Name)
				reviews.ObserveVariable(x.Name, val, x.Tier.String()+" (masked by default)")
			}
		}
	}

	pol, err := policy.Load(os.Getenv("POLICY_FILE"))
	if err != nil {
		log.Fatalf("policy: %v", err)
	}

	var store budget.Store = budget.NewMemory()
	if addr := os.Getenv("REDIS_ADDR"); addr != "" {
		r := budget.NewRedis(addr)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := r.Ping(ctx); err != nil {
			log.Fatalf("redis %s: %v", addr, err)
		}
		cancel()
		store = r
	}

	ring := audit.NewRing(2000)
	logger := audit.Multi{audit.NewJSONLogger(os.Stdout), ring}

	transport, err := upstreamTransport(os.Getenv("UPSTREAM_CA_FILES"))
	if err != nil {
		log.Fatalf("UPSTREAM_CA_FILES: %v", err)
	}

	gw := &gateway.Gateway{
		Upstream:    upstream,
		UpstreamKey: os.Getenv("UPSTREAM_API_KEY"),
		TokenLimit:  limit,
		Client:      &http.Client{Transport: transport},
		Vault:       v,
		Policy:      pol,
		Budget:      store,
		Audit:       logger,
		Reviews:     reviews,
		Scans:       scans,
	}

	nodes, err := forward.LoadNodes(os.Getenv("NODES_FILE"))
	if err != nil {
		log.Fatalf("NODES_FILE: %v", err)
	}
	if caCert := os.Getenv("CA_CERT_FILE"); caCert != "" {
		fp, err := forward.New(caCert, os.Getenv("CA_KEY_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		fp.Nodes, fp.Gateway, fp.Transport, fp.Audit = nodes, gw, transport, logger
		fp.Approved = map[string]bool{}
		for _, h := range strings.Split(env("APPROVED_LLM_HOSTS", "api.anthropic.com"), ",") {
			fp.Approved[strings.TrimSpace(h)] = true
		}
		fwdAddr := env("FORWARD_ADDR", ":3128")
		log.Printf("forward proxy (TLS inspection) listening on %s, %d nodes allowlisted", fwdAddr, len(nodes.List()))
		go func() { log.Fatal(http.ListenAndServe(fwdAddr, fp)) }()
	}

	if token := os.Getenv("ADMIN_TOKEN"); token != "" {
		api := &admin.API{Token: token, AdvisorToken: os.Getenv("ADVISOR_TOKEN"), Events: ring, Budget: store,
			Audit: logger, Vault: v, Policy: pol, Reviews: reviews, Scans: scans, InjectionAt: injectionAt,
			TokenLimit: limit, Nodes: nodes.List()}
		adminAddr := env("ADMIN_ADDR", ":8081")
		log.Printf("admin api listening on %s", adminAddr)
		go func() { log.Fatal(http.ListenAndServe(adminAddr, api.Handler())) }()
	} else {
		log.Printf("ADMIN_TOKEN not set, admin api disabled")
	}

	addr := env("LISTEN_ADDR", ":8080")
	log.Printf("provenance gate listening on %s -> %s", addr, upstream)
	log.Fatal(http.ListenAndServe(addr, gw))
}

// upstreamTransport trusts the system roots plus any extra CA files (the demo
// "internet" CA stands in for public CAs).
func upstreamTransport(caFiles string) (*http.Transport, error) {
	pool, err := x509.SystemCertPool()
	if err != nil {
		pool = x509.NewCertPool()
	}
	for _, f := range strings.Split(caFiles, ",") {
		if f = strings.TrimSpace(f); f == "" {
			continue
		}
		pem, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("no certificates in %s", f)
		}
	}
	t := http.DefaultTransport.(*http.Transport).Clone()
	t.Proxy = nil
	t.TLSClientConfig = &tls.Config{RootCAs: pool}
	return t, nil
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

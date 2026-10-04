// Command gate runs the HushGate proxy.
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
	"strings"
	"sync/atomic"
	"time"

	"github.com/Mikformatycy/hushgate/proxy/internal/admin"
	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
	"github.com/Mikformatycy/hushgate/proxy/internal/budget"
	"github.com/Mikformatycy/hushgate/proxy/internal/config"
	"github.com/Mikformatycy/hushgate/proxy/internal/engine"
	"github.com/Mikformatycy/hushgate/proxy/internal/forward"
	"github.com/Mikformatycy/hushgate/proxy/internal/gateway"
	"github.com/Mikformatycy/hushgate/proxy/internal/metrics"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/review"
	"github.com/Mikformatycy/hushgate/proxy/internal/scan"
	"github.com/Mikformatycy/hushgate/proxy/internal/shell"
	"github.com/Mikformatycy/hushgate/proxy/internal/signature"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

func main() {
	upstream, err := url.Parse(env("UPSTREAM_URL", "https://api.anthropic.com"))
	if err != nil {
		log.Fatalf("UPSTREAM_URL: %v", err)
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
	mx := metrics.New()
	logger := audit.Multi{audit.NewJSONLogger(os.Stdout), ring, mx}
	// A durable JSON Lines audit trail: SIEM-ready, exported from the dashboard,
	// and replayed on startup so the dashboard keeps its history across restarts.
	auditFile := os.Getenv("AUDIT_LOG_FILE")
	if auditFile != "" {
		if past, err := audit.ReadFile(auditFile); err == nil {
			ring.Restore(past)
			log.Printf("audit: restored %d events from %s", len(past), auditFile)
		}
		fl, err := audit.OpenFile(auditFile)
		if err != nil {
			log.Fatalf("AUDIT_LOG_FILE: %v", err)
		}
		logger = append(logger, fl)
	}

	// Everything a security team tunes lives in one policy file, reloaded
	// live. Environment variables only carry deployment settings.
	eng := &engine.Engine{
		Policy:     &policy.Policy{},
		Vault:      vault.New(vault.Confidential),
		Reviews:    review.NewStore(),
		Nodes:      forward.NewRegistry(nil),
		Guard:      shell.NewGuard(nil, nil),
		Signatures: &signature.Set{},
		Feeds:      signature.NewLoader(),
	}
	policyPath := env("HUSHGATE_POLICY", "/etc/hushgate/hushgate.yaml")
	// After a reload, waiting AI suggestions are checked against the new
	// review.auto_accept settings. Set once the admin API exists; run in the
	// background because applying a decision edits (and reloads) the file.
	var onReload atomic.Pointer[func()]
	live := config.NewLive(policyPath, eng.Apply, func(action, detail string) {
		logger.Record(audit.Event{Agent: "hushgate.yaml", Kind: "policy", Action: action, Reason: detail})
		if f := onReload.Load(); f != nil && action == "reloaded" {
			go (*f)()
		}
	})
	// Watch signature feeds too: a new local file or a newly fetched URL copy reloads.
	live.Extra = func(c *config.Config) [][]byte { return eng.Feeds.Raw(c.Signatures.Feeds) }
	if err := live.Load(); err != nil {
		log.Fatalf("policy %s: %v", policyPath, err)
	}
	for _, x := range eng.Vault.Vars() {
		log.Printf("vault: %s %s (%s)", x.Tier, x.Name, x.Reason)
	}
	log.Printf("policy %s loaded (version %s), watching for changes", policyPath, live.Status().Version)
	go live.Watch(context.Background(), time.Second)
	go func() { // fetch URL feeds when new or due; the watcher applies new copies
		for {
			cfg := live.Get()
			eng.Feeds.FetchDue(context.Background(), cfg.Signatures.Feeds,
				time.Duration(cfg.Signatures.RefreshSeconds)*time.Second)
			time.Sleep(2 * time.Second)
		}
	}()

	transport, err := upstreamTransport(os.Getenv("UPSTREAM_CA_FILES"))
	if err != nil {
		log.Fatalf("UPSTREAM_CA_FILES: %v", err)
	}
	scans := scan.NewStore(500)

	// Chat Completions (OpenAI-compatible): OpenAI by default; https://openrouter.ai/api
	// for OpenRouter, or any compatible server.
	chatUpstream, err := url.Parse(env("OPENAI_UPSTREAM_URL", "https://api.openai.com"))
	if err != nil {
		log.Fatalf("OPENAI_UPSTREAM_URL: %v", err)
	}
	gw := &gateway.Gateway{
		Upstream:        upstream,
		UpstreamKey:     os.Getenv("UPSTREAM_API_KEY"),
		ChatUpstream:    chatUpstream,
		ChatUpstreamKey: os.Getenv("OPENAI_UPSTREAM_API_KEY"),
		Limits:          func(agent string) int64 { return live.Get().LimitFor(agent) },
		ModelOK:         func(model string) bool { return live.Get().ModelAllowed(model) },
		Client:          &http.Client{Transport: transport},
		Vault:           eng.Vault,
		Policy:          eng.Policy,
		Budget:          store,
		Audit:           logger,
		Reviews:         eng.Reviews,
		Scans:           scans,
		Guard:           eng.Guard,
		Signatures:      eng.Signatures,
		Metrics:         mx,
	}

	if caCert := os.Getenv("CA_CERT_FILE"); caCert != "" {
		fp, err := forward.New(caCert, os.Getenv("CA_KEY_FILE"))
		if err != nil {
			log.Fatal(err)
		}
		fp.Nodes, fp.Gateway, fp.Transport, fp.Audit = eng.Nodes, gw, transport, logger
		fp.Approved = func(host string) bool { return live.Get().HostApproved(host) }
		fwdAddr := env("FORWARD_ADDR", ":3128")
		log.Printf("forward proxy (TLS inspection) listening on %s, %d nodes allowlisted", fwdAddr, len(eng.Nodes.List()))
		go func() { log.Fatal(http.ListenAndServe(fwdAddr, fp)) }()
	}

	if token := os.Getenv("ADMIN_TOKEN"); token != "" {
		api := &admin.API{Token: token, AdvisorToken: os.Getenv("ADVISOR_TOKEN"), Events: ring, Budget: store,
			Audit: logger, Vault: eng.Vault, Policy: eng.Policy, Reviews: eng.Reviews, Scans: scans, Live: live,
			Signatures: eng.Signatures, Metrics: mx, MetricsToken: os.Getenv("METRICS_TOKEN"), AuditFile: auditFile}
		recheck := api.RecheckReviews
		onReload.Store(&recheck)
		adminAddr := env("ADMIN_ADDR", ":8081")
		log.Printf("admin api listening on %s", adminAddr)
		go func() { log.Fatal(http.ListenAndServe(adminAddr, api.Handler())) }()
	} else {
		log.Printf("ADMIN_TOKEN not set, admin api disabled")
	}

	addr := env("LISTEN_ADDR", ":8080")
	log.Printf("hushgate listening on %s -> %s", addr, upstream)
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

// Command gate runs the Provenance Gate proxy.
package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/gateway"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
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
	for _, path := range strings.Split(os.Getenv("VAULT_ENV_FILES"), ",") {
		if path = strings.TrimSpace(path); path == "" {
			continue
		}
		vars, err := v.LoadEnvFile(path)
		if err != nil {
			log.Fatalf("load %s: %v", path, err)
		}
		for _, x := range vars {
			log.Printf("vault: %s %s", x.Tier, x.Name)
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

	gw := &gateway.Gateway{
		Upstream:    upstream,
		UpstreamKey: os.Getenv("UPSTREAM_API_KEY"),
		TokenLimit:  limit,
		Client:      &http.Client{},
		Vault:       v,
		Policy:      pol,
		Budget:      store,
		Audit:       audit.NewJSONLogger(os.Stdout),
	}

	addr := env("LISTEN_ADDR", ":8080")
	log.Printf("provenance gate listening on %s -> %s", addr, upstream)
	log.Fatal(http.ListenAndServe(addr, gw))
}

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

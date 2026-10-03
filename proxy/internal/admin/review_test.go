package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/review"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

func reviewAPI(t *testing.T) (*API, http.Handler) {
	v := vault.New(vault.Confidential)
	v.Add("TEAM_CHANNEL", "payments-oncall", vault.Confidential)
	pol := &policy.Policy{Tools: map[string]policy.Sink{}, Default: policy.Deny}
	rs := review.NewStore()
	rs.ObserveTool("post_to_slack", "Post a message to Slack", map[string]any{"type": "object"}, "deny (policy default)", "a1")
	val, _ := v.Value("TEAM_CHANNEL")
	rs.ObserveVariable("TEAM_CHANNEL", val, "C2 (masked by default)")
	ring := audit.NewRing(100)
	api := &API{Token: "admin", AdvisorToken: "advisor", Events: ring, Budget: budget.NewMemory(), Audit: ring,
		Vault: v, Policy: pol, Reviews: rs}
	return api, api.Handler()
}

func do(h http.Handler, method, path, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAdvisorCanOnlySuggest(t *testing.T) {
	api, h := reviewAPI(t)
	if rec := do(h, "GET", "/api/reviews", "advisor", ""); rec.Code != 200 {
		t.Fatalf("advisor list: %d", rec.Code)
	}
	for _, c := range []struct{ method, path, body string }{
		{"POST", "/api/reviews/tool:post_to_slack/decision", `{"action":"apply","value":"local"}`},
		{"POST", "/api/agents/a1/reset", ""},
		{"GET", "/api/config", ""},
		{"GET", "/api/events", ""},
	} {
		if rec := do(h, c.method, c.path, "advisor", c.body); rec.Code != http.StatusNotFound && rec.Code != http.StatusMethodNotAllowed {
			t.Errorf("advisor %s %s: got %d, want no route", c.method, c.path, rec.Code)
		}
	}
	if api.Policy.Has("post_to_slack") {
		t.Fatal("advisor changed policy")
	}

	if rec := do(h, "POST", "/api/reviews/tool:post_to_slack/suggestion", "advisor", `{"value":"network","rationale":"sends messages to an external service","confidence":"high","model":"m"}`); rec.Code != 200 {
		t.Fatalf("suggest: %d %s", rec.Code, rec.Body)
	}
	if rec := do(h, "POST", "/api/reviews/tool:post_to_slack/suggestion", "advisor", `{"value":"allow-everything"}`); rec.Code != http.StatusConflict {
		t.Fatalf("invalid suggestion accepted: %d", rec.Code)
	}
	it, _ := api.Reviews.Get("tool:post_to_slack")
	if it.Status != "pending" || it.Suggestion == nil || it.Suggestion.Value != "network" {
		t.Fatalf("item = %+v", it)
	}
	if api.Policy.SinkFor("post_to_slack") != policy.Deny {
		t.Fatal("a suggestion must not change policy")
	}
}

func TestHumanDecisionApplies(t *testing.T) {
	api, h := reviewAPI(t)
	do(h, "POST", "/api/reviews/tool:post_to_slack/suggestion", "advisor", `{"value":"network","rationale":"r"}`)

	if rec := do(h, "POST", "/api/reviews/tool:post_to_slack/decision", "admin", `{"action":"apply","value":"network"}`); rec.Code != 200 {
		t.Fatalf("decide tool: %d %s", rec.Code, rec.Body)
	}
	if api.Policy.SinkFor("post_to_slack") != policy.Network {
		t.Fatal("policy not updated")
	}
	if rec := do(h, "POST", "/api/reviews/tool:post_to_slack/decision", "admin", `{"action":"apply","value":"local"}`); rec.Code != http.StatusConflict {
		t.Fatalf("second decision on same item: %d", rec.Code)
	}

	masked, _ := api.Vault.Mask([]byte(`{"c":"ping payments-oncall"}`))
	if !strings.Contains(string(masked), "{{VAULT_ENV_TEAM_CHANNEL}}") {
		t.Fatalf("C2 should be masked before review: %s", masked)
	}
	if rec := do(h, "POST", "/api/reviews/var:TEAM_CHANNEL/decision", "admin", `{"action":"apply","value":"C1"}`); rec.Code != 200 {
		t.Fatalf("decide var: %d %s", rec.Code, rec.Body)
	}
	masked, _ = api.Vault.Mask([]byte(`{"c":"ping payments-oncall"}`))
	if !strings.Contains(string(masked), "payments-oncall") {
		t.Fatalf("C1 should pass after review: %s", masked)
	}
	var cfg struct {
		Vault []varView `json:"vault"`
	}
	json.Unmarshal(do(h, "GET", "/api/config", "admin", "").Body.Bytes(), &cfg)
	if cfg.Vault[0].Tier != "C1" || cfg.Vault[0].Reason != "set by reviewer" {
		t.Fatalf("config = %+v", cfg.Vault)
	}

	if rec := do(h, "POST", "/api/reviews/var:TEAM_CHANNEL/decision", "admin", `{"action":"apply","value":"C9"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("invalid tier: %d", rec.Code)
	}
}

func TestShapeHidesValue(t *testing.T) {
	sh := review.Shape("payments-oncall")
	b, _ := json.Marshal(sh)
	if strings.Contains(string(b), "payments") || sh["pattern"] != "a8-a6" || sh["length"] != 15 {
		t.Fatalf("shape = %s", b)
	}
}

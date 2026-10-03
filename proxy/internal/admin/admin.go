// Package admin serves the dashboard API on its own listener. The agent can
// reach the proxy container, so every request needs a token. There are two
// roles: admin (the dashboard, full access) and advisor (the AI service,
// which may only read the review queue and attach suggestions).
package admin

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
	"github.com/Mikformatycy/hushgate/proxy/internal/budget"
	"github.com/Mikformatycy/hushgate/proxy/internal/config"
	"github.com/Mikformatycy/hushgate/proxy/internal/policy"
	"github.com/Mikformatycy/hushgate/proxy/internal/review"
	"github.com/Mikformatycy/hushgate/proxy/internal/scan"
	"github.com/Mikformatycy/hushgate/proxy/internal/signature"
	"github.com/Mikformatycy/hushgate/proxy/internal/vault"
)

type API struct {
	Token        string
	AdvisorToken string // optional
	Events       *audit.Ring
	Budget       budget.Store
	Audit        audit.Logger
	Vault        *vault.Vault
	Policy       *policy.Policy
	Reviews      *review.Store
	Scans        *scan.Store
	Live         *config.Live // the policy file: thresholds, budgets, nodes; decisions are written back to it
	Signatures   *signature.Set
}

type agentView struct {
	ID         string `json:"id"`
	Used       int64  `json:"used"`
	Limit      int64  `json:"limit"`
	Killed     bool   `json:"killed"`
	KillReason string `json:"kill_reason,omitempty"`
}

type varView struct {
	Name   string `json:"name"`
	Tier   string `json:"tier"`
	Reason string `json:"reason"`
}

type detectorView struct {
	Name  string `json:"name"`
	Tier  string `json:"tier"`
	Check string `json:"check"`
}

func (a *API) Handler() http.Handler {
	admin := http.NewServeMux()
	admin.HandleFunc("GET /api/events", a.events)
	admin.HandleFunc("GET /api/agents", a.agents)
	admin.HandleFunc("POST /api/agents/{id}/reset", a.reset)
	admin.HandleFunc("GET /api/config", a.config)
	admin.HandleFunc("GET /api/reviews", a.reviews)
	admin.HandleFunc("GET /api/signatures", a.signatures)
	admin.HandleFunc("POST /api/reviews/{id}/decision", a.decide)

	advisor := http.NewServeMux()
	advisor.HandleFunc("GET /api/reviews", a.reviews)
	advisor.HandleFunc("POST /api/reviews/{id}/suggestion", a.suggest)
	advisor.HandleFunc("GET /api/scans", a.scans)
	advisor.HandleFunc("POST /api/scans/{id}/result", a.scanResult)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got := []byte(r.Header.Get("Authorization"))
		switch {
		case tokenMatches(got, a.Token):
			admin.ServeHTTP(w, r)
		case tokenMatches(got, a.AdvisorToken):
			advisor.ServeHTTP(w, r)
		default:
			http.Error(w, "unauthorized", http.StatusUnauthorized)
		}
	})
}

func tokenMatches(header []byte, token string) bool {
	return token != "" && subtle.ConstantTimeCompare(header, []byte("Bearer "+token)) == 1
}

func (a *API) reviews(w http.ResponseWriter, r *http.Request) {
	if a.Reviews == nil {
		writeJSON(w, []review.Item{})
		return
	}
	writeJSON(w, a.Reviews.List())
}

// suggest is the advisor's only write: it attaches advice and changes nothing.
func (a *API) suggest(w http.ResponseWriter, r *http.Request) {
	var sg review.Suggestion
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&sg); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	if len(sg.Rationale) > 500 {
		sg.Rationale = sg.Rationale[:500]
	}
	it, err := a.Reviews.Suggest(r.PathValue("id"), sg)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	reason := sg.Rationale
	if reason == "" {
		reason = fmt.Sprintf("%.0f%% confidence (%s)", sg.Confidence*100, sg.Model)
	}
	a.Audit.Record(audit.Event{Agent: "ai-advisor", Kind: "suggestion", Tool: it.Subject, Action: sg.Value,
		Reason: reason})
	writeJSON(w, it)
}

func (a *API) scans(w http.ResponseWriter, r *http.Request) {
	if a.Scans == nil {
		writeJSON(w, []scan.Item{})
		return
	}
	writeJSON(w, a.Scans.Pending(20))
}

// scanResult records the advisor's injection probability. Above the
// threshold it raises an alert in the audit log; it never blocks anything.
func (a *API) scanResult(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Injection float64 `json:"injection"`
		Model     string  `json:"model"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	it, err := a.Scans.Resolve(r.PathValue("id"), req.Injection, req.Model)
	if err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	if req.Injection >= *a.Live.Get().Injection.AlertThreshold {
		a.Audit.Record(audit.Event{Agent: it.Agent, Kind: "injection", Tool: it.Source,
			Reason: fmt.Sprintf("the content contains hidden instructions for an AI (%.0f%%, %s), flagged %d ms after the agent received it",
				req.Injection*100, req.Model, time.Since(it.ObservedAt).Milliseconds())})
	}
	w.WriteHeader(http.StatusNoContent)
}

// decide applies (or dismisses) a review. Humans only.
func (a *API) decide(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Action string `json:"action"` // apply | dismiss
		Value  string `json:"value"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 64<<10)).Decode(&req); err != nil {
		http.Error(w, "bad json", http.StatusBadRequest)
		return
	}
	id := r.PathValue("id")
	current, ok := a.Reviews.Get(id)
	if !ok {
		http.Error(w, "no such review", http.StatusNotFound)
		return
	}
	if req.Action == "dismiss" {
		it, err := a.Reviews.Resolve(id, "dismissed", "")
		if err != nil {
			http.Error(w, err.Error(), http.StatusConflict)
			return
		}
		a.Audit.Record(audit.Event{Agent: "reviewer", Kind: "review", Tool: it.Subject, Action: "dismissed",
			Reason: "kept current rule: " + it.Current})
		writeJSON(w, it)
		return
	}

	note := "set by reviewer"
	if sg := current.Suggestion; sg != nil {
		if sg.Value == req.Value {
			note += ", AI suggestion accepted"
		} else {
			note += ", AI suggested " + sg.Value
		}
	}
	if (current.Kind == "tool" && !policy.ValidSink(req.Value)) ||
		(current.Kind == "variable" && !validTier(req.Value)) {
		http.Error(w, "invalid value "+req.Value, http.StatusBadRequest)
		return
	}
	if current.Status != "pending" {
		http.Error(w, "review "+id+" is already "+current.Status, http.StatusConflict)
		return
	}
	// The decision becomes a rule in the policy file, so it survives restarts
	// and the file stays the single source of truth. The reload that follows
	// applies it and settles this review.
	var err error
	if current.Kind == "tool" {
		err = a.Live.SetToolRule(current.Subject, req.Value)
	} else {
		err = a.Live.SetVariableOverride(current.Subject, req.Value)
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.Reviews.Settle(id, req.Value)
	it, _ := a.Reviews.Get(id)
	a.Audit.Record(audit.Event{Agent: "reviewer", Kind: "review", Tool: it.Subject, Action: req.Value,
		Reason: note + "; written to hushgate.yaml"})
	writeJSON(w, it)
}

func (a *API) events(w http.ResponseWriter, r *http.Request) {
	after, _ := strconv.ParseInt(r.URL.Query().Get("after"), 10, 64)
	writeJSON(w, a.Events.After(after))
}

func (a *API) agents(w http.ResponseWriter, r *http.Request) {
	ids, err := a.Budget.Agents(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	out := make([]agentView, 0, len(ids))
	for _, id := range ids {
		st, err := a.Budget.Status(r.Context(), id)
		if err != nil {
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		out = append(out, agentView{ID: id, Used: st.Used, Limit: a.Live.Get().LimitFor(id), Killed: st.Killed,
			KillReason: st.KillReason})
	}
	writeJSON(w, out)
}

func (a *API) reset(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := a.Budget.Reset(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	a.Audit.Record(audit.Event{Agent: id, Kind: "reset", Reason: "reset from dashboard"})
	w.WriteHeader(http.StatusNoContent)
}

func (a *API) config(w http.ResponseWriter, r *http.Request) {
	loaded := a.Vault.Vars()
	vars := make([]varView, 0, len(loaded))
	for _, v := range loaded {
		vars = append(vars, varView{Name: v.Name, Tier: v.Tier.String(), Reason: v.Reason})
	}
	var detectors []detectorView
	seen := map[string]bool{}
	for _, d := range vault.Detectors {
		if !seen[d.Name] {
			seen[d.Name] = true
			detectors = append(detectors, detectorView{Name: d.Name, Tier: d.Tier.String(), Check: d.Check})
		}
	}
	cfg := a.Live.Get()
	nodes := make([]map[string]string, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		nodes = append(nodes, map[string]string{"id": n.ID, "owner": n.Owner}) // never the token
	}
	writeJSON(w, map[string]any{
		"vault": vars, "mask_from": a.Vault.MaskFrom().String(), "policy": a.Policy.Snapshot(),
		"token_limit": cfg.Budgets.DefaultTokens, "budgets": cfg.Budgets.Agents,
		"models": cfg.Models.Allow, "llm_hosts": cfg.LLMHosts.Approved,
		"injection_threshold": *cfg.Injection.AlertThreshold,
		"nodes":               nodes, "detectors": detectors, "policy_file": a.Live.Status(),
	})
}

func (a *API) signatures(w http.ResponseWriter, r *http.Request) {
	if a.Signatures == nil {
		writeJSON(w, map[string]any{"feeds": []any{}, "signatures": []any{}})
		return
	}
	sigs, feeds := a.Signatures.List()
	writeJSON(w, map[string]any{"feeds": feeds, "signatures": sigs})
}

func validTier(s string) bool {
	_, ok := vault.ParseTier(s)
	return ok
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

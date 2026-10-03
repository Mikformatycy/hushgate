// Package admin serves the dashboard API on its own listener. The agent can
// reach the proxy container, so every request needs the admin token.
package admin

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/Mikformatycy/goldman-sachs/proxy/internal/audit"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/budget"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/policy"
	"github.com/Mikformatycy/goldman-sachs/proxy/internal/vault"
)

type API struct {
	Token      string
	Events     *audit.Ring
	Budget     budget.Store
	Audit      audit.Logger
	Vars       []vault.Var
	MaskFrom   vault.Tier
	Policy     *policy.Policy
	TokenLimit int64
}

type agentView struct {
	ID         string `json:"id"`
	Used       int64  `json:"used"`
	Limit      int64  `json:"limit"`
	Killed     bool   `json:"killed"`
	KillReason string `json:"kill_reason,omitempty"`
}

type varView struct {
	Name string `json:"name"`
	Tier string `json:"tier"`
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/events", a.events)
	mux.HandleFunc("GET /api/agents", a.agents)
	mux.HandleFunc("POST /api/agents/{id}/reset", a.reset)
	mux.HandleFunc("GET /api/config", a.config)
	return a.auth(mux)
}

func (a *API) auth(next http.Handler) http.Handler {
	want := []byte("Bearer " + a.Token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
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
		out = append(out, agentView{ID: id, Used: st.Used, Limit: a.TokenLimit, Killed: st.Killed, KillReason: st.KillReason})
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
	vars := make([]varView, 0, len(a.Vars))
	for _, v := range a.Vars {
		vars = append(vars, varView{Name: v.Name, Tier: v.Tier.String()})
	}
	writeJSON(w, map[string]any{"vault": vars, "mask_from": a.MaskFrom.String(), "policy": a.Policy, "token_limit": a.TokenLimit})
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

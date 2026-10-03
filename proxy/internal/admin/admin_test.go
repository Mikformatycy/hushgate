package admin

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
)

func TestAPI(t *testing.T) {
	api, _ := newAPI(t, "budgets:\n  default_tokens: 100\n", "")
	api.Token = "t"
	store := api.Budget
	store.AddUsage(context.Background(), "a1", 42)
	store.Kill(context.Background(), "a1", "send_email: leak")
	h := api.Handler()

	do := func(method, path, token string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, nil)
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	if rec := do("GET", "/api/agents", ""); rec.Code != http.StatusUnauthorized {
		t.Fatalf("no token: %d", rec.Code)
	}
	if rec := do("GET", "/api/agents", "wrong"); rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad token: %d", rec.Code)
	}

	var agents []agentView
	json.Unmarshal(do("GET", "/api/agents", "t").Body.Bytes(), &agents)
	if len(agents) != 1 || !agents[0].Killed || agents[0].Used != 42 || agents[0].Limit != 100 {
		t.Fatalf("agents = %+v", agents)
	}

	if rec := do("POST", "/api/agents/a1/reset", "t"); rec.Code != http.StatusNoContent {
		t.Fatalf("reset: %d", rec.Code)
	}
	if st, _ := store.Status(context.Background(), "a1"); st.Killed || st.Used != 0 {
		t.Fatalf("not reset: %+v", st)
	}

	var events []audit.Event
	json.Unmarshal(do("GET", "/api/events?after=0", "t").Body.Bytes(), &events)
	if len(events) != 1 || events[0].Kind != "reset" || events[0].ID != 1 {
		t.Fatalf("events = %+v", events)
	}
	json.Unmarshal(do("GET", "/api/events?after=1", "t").Body.Bytes(), &events)
	if len(events) != 0 {
		t.Fatalf("after=1 should be empty: %+v", events)
	}
}

package metrics

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
)

func TestEventsCounted(t *testing.T) {
	m := New()
	m.Record(audit.Event{Kind: "tool_call", Tool: "Bash", Action: "kill"})
	m.Record(audit.Event{Kind: "tool_call", Tool: "Bash", Action: "kill"})
	m.Record(audit.Event{Kind: "signature", Action: "block", Reason: "HG-RCE-001 Remote script piped into a shell"})
	m.Record(audit.Event{Kind: "policy", Action: "rejected"})
	m.Record(audit.Event{Kind: "shadow_ai"})
	m.Masked("C3", 2)
	m.Tokens("a1", 150)

	checks := map[string]float64{
		"tool calls": testutil.ToFloat64(m.toolCalls.WithLabelValues("Bash", "kill")),
		"signature":  testutil.ToFloat64(m.signatureHits.WithLabelValues("HG-RCE-001", "block")),
		"reload":     testutil.ToFloat64(m.policyReloads.WithLabelValues("rejected")),
		"shadow ai":  testutil.ToFloat64(m.events.WithLabelValues("shadow_ai", "")),
		"masked":     testutil.ToFloat64(m.masked.WithLabelValues("C3")),
		"tokens":     testutil.ToFloat64(m.tokens.WithLabelValues("a1")),
	}
	want := map[string]float64{"tool calls": 2, "signature": 1, "reload": 1, "shadow ai": 1, "masked": 2, "tokens": 150}
	for k, v := range want {
		if checks[k] != v {
			t.Errorf("%s = %v, want %v", k, checks[k], v)
		}
	}
}

func TestSummaryPercentiles(t *testing.T) {
	m := New()
	for i := 1; i <= 100; i++ {
		m.Observe(StagePreprocess, time.Duration(i)*time.Millisecond)
	}
	s := m.Summary()[StagePreprocess]
	if s.Count != 100 || s.P50 != 50 || s.P95 != 95 || s.Max != 100 {
		t.Fatalf("summary = %+v", s)
	}
	for i := 0; i < 1500; i++ { // window keeps the last 1000
		m.Observe(StageTotal, time.Millisecond)
	}
	if m.Summary()[StageTotal].Count != 1000 {
		t.Fatal("window not bounded")
	}
}

func TestNilSafe(t *testing.T) {
	var m *Metrics
	m.Record(audit.Event{Kind: "x"})
	m.Observe(StageTotal, time.Second)
	m.Request("messages", 200, time.Second)
	m.Masked("C3", 1)
	m.Tokens("a", 1)
	if len(m.Summary()) != 0 {
		t.Fatal("nil summary")
	}
}

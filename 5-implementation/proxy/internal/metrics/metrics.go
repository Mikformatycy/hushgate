// Package metrics exposes HushGate's performance and security counters in
// Prometheus format, and keeps recent latency samples for the dashboard.
package metrics

import (
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Mikformatycy/hushgate/proxy/internal/audit"
)

// Stages of a request whose latency is measured separately, so the gate's own
// overhead can be told apart from the provider's.
const (
	StagePreprocess   = "preprocess"          // budget, model check, masking, before forwarding
	StageToolDecision = "tool_decision"       // policy, Bash guard, signatures for one tool call
	StageUpstreamTTFB = "upstream_first_byte" // waiting for the LLM provider
	StageTotal        = "total"               // whole request through the gate
)

var stages = []string{StagePreprocess, StageToolDecision, StageUpstreamTTFB, StageTotal}

// Metrics is safe to use as a nil pointer (all methods no-op), so components
// work without it in tests.
type Metrics struct {
	Registry *prometheus.Registry

	requests      *prometheus.CounterVec
	latency       *prometheus.HistogramVec
	toolCalls     *prometheus.CounterVec
	masked        *prometheus.CounterVec
	signatureHits *prometheus.CounterVec
	events        *prometheus.CounterVec
	tokens        *prometheus.CounterVec
	policyReloads *prometheus.CounterVec

	mu     sync.Mutex
	recent map[string]*window
}

func New() *Metrics {
	m := &Metrics{
		Registry: prometheus.NewRegistry(),
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_requests_total", Help: "Requests handled by the gateway, by route and HTTP status code."},
			[]string{"route", "code"}),
		latency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "hushgate_latency_seconds",
			Help:    "Latency by stage: preprocess and tool_decision are the gate's own overhead, upstream_first_byte is the LLM provider, total is end to end.",
			Buckets: []float64{.0001, .00025, .0005, .001, .0025, .005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10, 30, 60}},
			[]string{"stage"}),
		toolCalls: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_tool_calls_total", Help: "Tool calls the model requested, by tool and final action."},
			[]string{"tool", "action"}),
		masked: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_masked_values_total", Help: "Sensitive values replaced with placeholders before leaving, by tier."},
			[]string{"tier"}),
		signatureHits: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_signature_hits_total", Help: "Attack signature matches, by signature id and action."},
			[]string{"id", "action"}),
		events: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_audit_events_total", Help: "Audit events, by kind and action."},
			[]string{"kind", "action"}),
		tokens: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_llm_tokens_total", Help: "LLM tokens used (input, output and cache), by agent."},
			[]string{"agent"}),
		policyReloads: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "hushgate_policy_reloads_total", Help: "Policy file reloads, by result (reloaded or rejected)."},
			[]string{"result"}),
		recent: map[string]*window{},
	}
	m.Registry.MustRegister(m.requests, m.latency, m.toolCalls, m.masked, m.signatureHits, m.events, m.tokens,
		m.policyReloads, collectors.NewGoCollector(), collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	for _, s := range stages {
		m.recent[s] = &window{}
	}
	return m
}

// Handler serves the Prometheus text format.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{})
}

// Record counts an audit event (Metrics is an audit.Logger).
func (m *Metrics) Record(e audit.Event) {
	if m == nil {
		return
	}
	m.events.WithLabelValues(e.Kind, e.Action).Inc()
	switch e.Kind {
	case "tool_call":
		m.toolCalls.WithLabelValues(e.Tool, e.Action).Inc()
	case "signature":
		id, _, _ := strings.Cut(e.Reason, " ")
		m.signatureHits.WithLabelValues(id, e.Action).Inc()
	case "policy":
		m.policyReloads.WithLabelValues(e.Action).Inc()
	}
}

func (m *Metrics) Observe(stage string, d time.Duration) {
	if m == nil {
		return
	}
	m.latency.WithLabelValues(stage).Observe(d.Seconds())
	m.mu.Lock()
	m.recent[stage].add(d)
	m.mu.Unlock()
}

func (m *Metrics) Request(route string, code int, d time.Duration) {
	if m == nil {
		return
	}
	m.requests.WithLabelValues(route, http.StatusText(code)).Inc()
	m.Observe(StageTotal, d)
}

func (m *Metrics) Masked(tier string, n int) {
	if m == nil || n == 0 {
		return
	}
	m.masked.WithLabelValues(tier).Add(float64(n))
}

func (m *Metrics) Tokens(agent string, n int64) {
	if m == nil || n <= 0 {
		return
	}
	m.tokens.WithLabelValues(agent).Add(float64(n))
}

// StageSummary is latency over the most recent samples, in milliseconds.
type StageSummary struct {
	Count int     `json:"count"`
	P50   float64 `json:"p50_ms"`
	P95   float64 `json:"p95_ms"`
	P99   float64 `json:"p99_ms"`
	Max   float64 `json:"max_ms"`
}

// Summary returns percentiles per stage over the last samples (for the dashboard;
// Prometheus gets the full histograms).
func (m *Metrics) Summary() map[string]StageSummary {
	out := map[string]StageSummary{}
	if m == nil {
		return out
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, s := range stages {
		out[s] = m.recent[s].summary()
	}
	return out
}

// window keeps the last 1000 samples of a stage.
type window struct {
	samples [1000]float64
	n, next int
}

func (w *window) add(d time.Duration) {
	w.samples[w.next] = float64(d.Microseconds()) / 1000
	w.next = (w.next + 1) % len(w.samples)
	if w.n < len(w.samples) {
		w.n++
	}
}

func (w *window) summary() StageSummary {
	if w.n == 0 {
		return StageSummary{}
	}
	s := append([]float64(nil), w.samples[:w.n]...)
	sort.Float64s(s)
	at := func(q float64) float64 { return s[int(q*float64(len(s)-1))] }
	return StageSummary{Count: w.n, P50: at(.5), P95: at(.95), P99: at(.99), Max: s[len(s)-1]}
}

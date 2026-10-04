# Performance

Enforcement is split into two paths with very different costs:

- **Deterministic enforcement** runs inline, on every request and every tool call: masking, the tool policy, the Bash guard, attack signatures, budgets and allowlists. It costs **microseconds**.
- **Non-deterministic (AI) enforcement** runs in the background: prompt-injection scoring and review suggestions. It takes **hundreds of milliseconds to a few seconds**, but it is **off the request path**, so it adds no latency to the agent and blocks nothing on its own.

All numbers below were measured on the demo hardware (Apple M4). Each section says how to reproduce them.

## Deterministic enforcement

### Cost of each check (Go microbenchmarks)

Run with `just bench`, or `cd 5-implementation/proxy && go test ./internal/bench ./internal/gateway -run '^$' -bench . -benchmem`.

| Check | What is measured | Time per operation |
|---|---|---|
| Tool policy decision | network tool carrying a secret placeholder → kill | **24 ns** |
| Restore real values into a local tool | 2 placeholders in a `write_file` call | **0.41 µs** |
| Classify one variable, with its reason | `SETTLEMENT_ACCOUNT` = an IBAN | **1.2 µs** |
| Bash guard | `sudo -E bash -c "c'u'rl … -d @.env …"` | **5.2 µs** |
| All 19 attack signatures, harmless command | `go test ./... && git status` | **17 µs** |
| All 19 attack signatures, malicious command | `curl … \| sudo bash` | **25 µs** |
| Mask a 4.5 KB request dense with secrets and PII | 12 passwords, keys, IBANs, cards and PESELs | **68 µs** (59 MB/s) |
| Mask a 100 KB coding-agent request | a system prompt and history with one secret | **1.08 ms** (92 MB/s) |

### End-to-end overhead of the gate

`BenchmarkThroughGate` sends a 4.5 KB request, with secrets in it, through the complete gateway to an in-process model that answers with a streamed tool call. The request is masked, the answer streamed, and the tool call checked against the policy, the Bash guard and all 19 signatures. `BenchmarkDirectToModel` sends the same request straight to the model.

| Path | Time per request |
|---|---|
| Directly to the model (baseline) | 30 µs |
| Through HushGate, with every check | 144 µs |
| **Added by HushGate** | **≈ 0.11 ms** |

### Live latencies in the running demo

These are histograms of every request and tool call in the company-network demo after the end-to-end suite has run. The dashboard shows them on its Performance panel, and Prometheus holds the full histograms (`hushgate_latency_seconds{stage=…}`).

| Stage | What it covers | p50 | p95 | p99 |
|---|---|---|---|---|
| Gate: request checks | budget, kill state, model allowlist, masking | **174 µs** | 224 µs | 239 µs |
| Gate: tool call decision | policy, Bash guard, 19 signatures | **68 µs** | 769 µs | 773 µs |
| LLM provider, first byte | the scripted demo model; a real model takes 0.5–3 s | 1.5 ms | 2.0 s | 2.0 s |
| End to end | the whole request, including the streamed answer | 1.8 ms | 2.0 s | 2.0 s |

With a real model, the gate's share of a request is well under 0.1%.

### What we optimised

Profiling showed masking was the only check whose cost grew with request size. Six of the detectors (IBAN, card, PESEL, both NIP forms and Stripe keys) start with a character class rather than a fixed prefix, so the regular-expression engine had to try every byte of the request: about 16 ms for a 100 KB request.

Each of those detectors now has a byte-level hint that finds where a match could start, such as a digit run of the right length, two capitals followed by two digits, the `nip` keyword or `k_live_`. The pattern then runs only on those short windows. Each window spans the whole run the pattern could consume, so the matches are identical. `vault.TestHintsMatchFullSearch` checks this against the full-text search on thousands of generated inputs.

| Request | Before | After | Speed-up |
|---|---|---|---|
| 4.5 KB, dense with secrets and PII | 573 µs | 68 µs | 8.4× |
| 100 KB coding-agent request | 15.7 ms | 1.08 ms | 14.5× |

## Non-deterministic (AI) enforcement

The AI advisor calls TypeSafe Jev (model `jev-1.13.0`), a classifier with calibrated probabilities. The gate queues work for it, and it posts results back through the advisor-only API.

| AI check | Measured latency | Effect on the request path |
|---|---|---|
| Prompt-injection score for a document the agent read | **267–603 ms** after the agent received it (demo runs: 267, 422, 506, 603 ms) | none: it runs after the tool result has been delivered, and only raises an alert |
| Review suggestion for a new tool or variable | **1.6–2.1 s** after the case first appears | none: until a decision exists, the safe default (deny or mask) applies |
| Auto-accept of a confident suggestion | applied about 0.2 s after the suggestion arrives (the rule is written to the policy file, which is reloaded once the write settles) | the rule takes effect for the next request |

Design consequences:
- **No added latency:** the AI never sits between the agent and the model.
- **No false blocking:** a high injection score raises a warning. Blocking stays with the deterministic policy, which stops the exfiltration itself (the kill switch fired on the poisoned report in every demo run).
- **Graceful degradation:** without a `TYPESAFE_API_KEY`, or if Jev is unreachable, every deterministic control still works; only warnings and suggestions are missing.
- **Bounded cost:** each (agent, document) pair is scanned once, even though conversations resend earlier tool results on every turn.

## Scaling

- **The gate is stateless.** Budgets and kills are in Redis, policy is in the file, and the audit trail is a file or a log shipper. More capacity means more replicas behind a load balancer, all pointing at the same Redis and policy.
- **Per request it costs** one Redis read before forwarding, one write after, and about 0.1–1 ms of CPU depending on request size.
- **Streaming is preserved.** Text deltas pass through as they arrive; only a `tool_use` block is held, and only until it is complete.
- **Signature feeds are pulled** by every gate from one central URL, with last-good fallback, so a fleet updates within one refresh interval (60 s in the demo).

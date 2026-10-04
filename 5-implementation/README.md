# 5 · Implementation

- [Code](#code)
- [How to run it](#how-to-run-it)
- [Implementation considerations](#implementation-considerations)
- [Deploying into existing agent ecosystems](#deploying-into-existing-agent-ecosystems)
- [Deployment settings](#deployment-settings)
- [Limitations and next steps](#limitations-and-next-steps)

## Code

```
5-implementation/
├── proxy/                 the gate (Go 1.26, one static binary)
│   ├── cmd/gate/          wiring: config, engine, gateway, forward proxy, admin API
│   └── internal/
│       ├── gateway/       reverse proxy for Anthropic Messages and OpenAI-compatible Chat Completions; stream inspection of tool calls
│       ├── vault/         tiers C0–C3 with reasons, detectors with checksums, placeholders, restoring values
│       ├── policy/        tool rules (local / network / deny) and per-tier actions (allow / block / kill)
│       ├── signature/     attack signature feeds (files and URLs, refresh, last good copy) and matching
│       ├── shell/         Bash guard: finds network programs in shell commands
│       ├── forward/       TLS-inspecting forward proxy, device allowlist
│       ├── detect/        recognises LLM traffic by path, headers and body
│       ├── config/        the policy file: parse, validate, diff, live reload, write-back
│       ├── engine/        applies a validated policy to the running components
│       ├── budget/        token budgets and the kill switch (Redis, or memory for tests)
│       ├── review/        review queue for cases the rules can't settle
│       ├── scan/          queue of content for the prompt-injection scan
│       ├── admin/         dashboard API: roles, reviews, auto-accept, exports, metrics
│       ├── audit/         audit events: ring buffer, JSON Lines file, replay
│       ├── metrics/       Prometheus metrics and latency summaries
│       └── bench/         microbenchmarks of every enforcement step
├── dashboard/             React 19 + Vite + Tailwind; nginx serves it and proxies /api with the admin token
├── advisor/               AI advisor (Python): TypeSafe Jev for injection scores and review suggestions
├── config/
│   ├── hushgate.yaml      THE policy file (live reload)
│   └── signatures.yaml    the 19 attack signatures (also served as a URL feed from GitHub)
├── agent/                 a scripted demo agent with read_file, write_file, send_email, post_to_slack, run_command
├── sandbox/claude-code/   real Claude Code in a container whose only route out is the gate
├── demo/
│   ├── fake_upstream.py   scripted model for offline demos
│   ├── workspace/         the demo project: .env secrets, a poisoned quarterly_report.md
│   └── corp/              company-network simulation: laptops, MDM profiles, CA generation, fake internet
├── deploy/prometheus.yml  Prometheus scrape config
├── video/                 the launch video (Remotion)
├── RUN.md                 every task as one command
└── INSTRUCTION.md         setups, scenarios and configuration in detail
```

The compose files (`docker-compose.yml` for gateway mode, `docker-compose.corp.yml` for the company network) and the `Justfile` are at the repository root, so every command runs from there.

## How to run it

You need Docker. Optionally, put `TYPESAFE_API_KEY` in `.env` at the repository root to enable the AI advisor.

```sh
just corp-up        # company network simulation, offline; dashboard on http://localhost:3000
just e2e            # run every scenario and check the results (19 checks)
just tests          # 74 automated tests, in Docker
```

[RUN.md](RUN.md) lists every command. [INSTRUCTION.md](INSTRUCTION.md) explains both setups and every scenario. The live demo script is [4-testing/DEMO.md](../4-testing/DEMO.md).

## Implementation considerations

**Streaming without blind spots.** Agents use server-sent events. The gate passes text deltas through as they arrive, but holds each `tool_use` block until its `content_block_stop`, because a tool call's arguments arrive in fragments. Only the complete call is checked. A dropped call is replaced by a text block telling the model why, so the agent's loop continues cleanly.

**Placeholders, not redaction.** Values become stable placeholders: `{{VAULT_ENV_<NAME>}}` for `.env` values, `{{VAULT_<KIND>_<hash>}}` for detected ones. The model can still reason about them ("put `{{VAULT_ENV_DB_PASSWORD}}` in the config"), and the gate restores the real value only inside a tool whose effects stay on the machine. Masking also covers JSON-escaped forms.

**Fail closed everywhere.** Unknown tools are denied. Unknown values are C2 and masked. An unparseable policy is never applied. A device without the company CA cannot complete TLS. A halted agent stays halted across restarts (Redis) until a person resets it.

**AI on the side, never in the path.** The advisor runs as a separate service with a token that can only read the queues and post results. It never receives secret values: it sees tool definitions, value *shapes* and masked tool results. If it is down, nothing is blocked and nothing slows down.

**Detectors that scale with request size.** Checksum validation (IBAN mod-97, Luhn, the PESEL check digit, NIP mod-11) keeps false positives near zero. Byte-level hints keep the slow patterns off most of the request (masking runs at about 92 MB/s; see [performance](../2-architecture/performance.md)).

**The policy as code.** `hushgate.yaml` can live in git, go through review, and roll out like any configuration. Strict and relaxed example profiles are in [1-solution/examples](../1-solution/examples/). The Bash guard's program list, the signature feeds and all thresholds are data, not code.

**Separation of duties.** There are three tokens: admin (the dashboard), advisor (suggestions only) and metrics (read-only). The provider API key lives only in the gate.

## Deploying into existing agent ecosystems

| Ecosystem | How HushGate is inserted | Changes for developers |
|---|---|---|
| **Claude Code, Claude Agent SDK, Anthropic SDKs** (Python, TypeScript, Go, …) | `ANTHROPIC_BASE_URL=https://hushgate.company.internal`, plus `ANTHROPIC_CUSTOM_HEADERS="X-Agent-Id: <team-or-agent>"` for per-agent budgets. Works with API keys and Claude subscriptions (the OAuth token is passed through). | None, if device management or a shell profile sets the variable |
| **LangChain, LlamaIndex, CrewAI and similar** on Anthropic models | The framework's Anthropic client takes a `base_url`. Point it at the gate. | One configuration value |
| **CI/CD runners** | Set `ANTHROPIC_BASE_URL` in the runner image or the organisation's secrets, and block direct egress to provider domains | None for pipelines |
| **Company laptops** | Device management installs the company root CA and the proxy (port 3128), with a per-device credential. LLM traffic is decrypted and inspected; other traffic is tunnelled untouched. | None. Agents keep their default provider URLs. |
| **Kubernetes** | Run the gate as a Deployment (stateless, scaled horizontally) with Redis. Agent namespaces get a NetworkPolicy whose only egress is the gate. | None |
| **Sandboxed coding agents** | Containers on an internal-only network with the gate as the only route, as in `sandbox/claude-code` | None |
| **OpenAI, OpenRouter and other OpenAI-compatible providers** (Hermes, Cline, aider, OpenCode and most harnesses) | Point the client's base URL at the gate (`http://gate:8080/v1`), or on the company network add the host to `llm_hosts.approved`. Chat Completions get the same masking, tool policy, signatures, Bash guard, budgets and injection scan as Anthropic requests. Upstream: `OPENAI_UPSTREAM_URL` (OpenAI by default, `https://openrouter.ai/api` for OpenRouter). | One configuration value |
| **Other APIs** (OpenAI Responses API used by Codex, native Gemini and Ollama) | Recognised by request shape and allowed or blocked as a whole through `llm_hosts.approved` | — |

**Rolling it out:**
1. Start in **observe mode**: tools default to `network` and `on_network_tool` is set to `allow`. Masking still protects every request, and the audit log records every tool call and every masked value, which shows what a stricter policy would stop.
2. Tighten the policy file team by team, using the [strict profile](../1-solution/examples/strict.yaml) for regulated workloads.
3. Distribute one signature feed to every gate.
4. Ship the audit file to the SIEM and point the existing Prometheus or Alertmanager at `/metrics` ([example alert rules](../3-reporting/README.md#example-alert-rules)).

**Scaling:**
- The gate keeps no state of its own: budgets and kills are in Redis, policy is in the file, and the audit trail is a file or a log stream. Run any number of replicas behind a load balancer.
- Per request it costs one Redis read, one Redis write and roughly 0.1–1 ms of CPU.
- The policy file can be mounted from a ConfigMap or synced from git; every replica reloads within a second.

## Deployment settings

These are environment variables for deployment. Everything a security team tunes is in the policy file instead.

| Variable | Default | Purpose |
|---|---|---|
| `LISTEN_ADDR` | `:8080` | Gateway: Anthropic Messages (`/v1/messages`) and Chat Completions (`/v1/chat/completions`) |
| `ADMIN_ADDR` | `:8081` | Admin API and `/metrics` |
| `FORWARD_ADDR` | `:3128` | Forward proxy, enabled when `CA_CERT_FILE` is set |
| `UPSTREAM_URL` | `https://api.anthropic.com` | Provider (the demo points it at the scripted model) |
| `UPSTREAM_API_KEY` | — | Provider key held by the gate. If unset, the agent's own credentials (API key or OAuth token) are passed through. |
| `OPENAI_UPSTREAM_URL` | `https://api.openai.com` | Provider for Chat Completions; `https://openrouter.ai/api` for OpenRouter (the demo compose file uses OpenRouter) |
| `OPENAI_UPSTREAM_API_KEY` | — | Bearer key the gate sends for Chat Completions (OpenRouter or OpenAI). If unset, the agent's own key is passed through. |
| `UPSTREAM_CA_FILES` | — | Extra CAs to trust for the provider (the simulated internet) |
| `HUSHGATE_POLICY` | `/etc/hushgate/hushgate.yaml` | Policy file path |
| `REDIS_ADDR` | — (in memory) | Budgets and kill state shared by replicas |
| `AUDIT_LOG_FILE` | — | Durable JSON Lines audit trail |
| `ADMIN_TOKEN`, `ADVISOR_TOKEN`, `METRICS_TOKEN` | — | The three API roles |
| `CA_CERT_FILE`, `CA_KEY_FILE` | — | Company CA for TLS inspection in company-network mode |
| Advisor: `TYPESAFE_API_KEY`, `GATE_ADMIN_URL`, `ADVISOR_TOKEN`, `JEV_MODEL` | — | The AI advisor service |

## Limitations and next steps

- **Other APIs:** deep inspection covers Anthropic Messages and OpenAI-compatible Chat Completions. The OpenAI Responses API (Codex), and native Gemini and Ollama traffic are recognised and allowed or blocked as a whole. Next: a Responses adapter for Codex.
- **Budgets** are counted in tokens. Next: currency per model and per team, including locally hosted models and compute time.
- **Queues:** the review and injection queues are in memory and rebuilt after a restart. The audit trail, budgets, kills and review decisions already persist. With several replicas, each one has its own queue and live event view; ship the audit files to one place, and point the dashboard at one replica, or move the queues to Redis (planned).
- **Next:** single sign-on and roles for the dashboard, signed signature feeds, and per-team policy profiles in one file.

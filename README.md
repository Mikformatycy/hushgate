# Provenance Gate

Deterministic API gateway between AI agents and LLM providers. See [IDEA.md](IDEA.md).

## Run

```sh
export ANTHROPIC_API_KEY=sk-ant-...      # held by the proxy only
docker compose up -d --build
docker compose exec agent python agent.py                  # summarize the poisoned report
docker compose exec agent python agent.py --egress-check   # should be blocked
docker compose logs -f proxy                               # audit events (JSON lines)
```

Reset a killed agent: `docker compose exec redis redis-cli del gate:killed:demo-agent gate:tokens:demo-agent`

## How it works

- **Vault**: `.env` values are classified C0–C3 and C2+ are replaced with `{{VAULT_ENV_NAME}}` before leaving the box. Known key formats (AWS, Anthropic, GitHub, JWT, PEM, DSN passwords) are masked even if they are not in `.env`. Override a tier with `KEY=value # @class: C3`.
- **Tool policy** (`proxy/policy.json`): every `tool_use` block is buffered until complete. `local` tools get secrets rehydrated, `network` tools carrying a vault token trip the kill switch, `deny` tools are blocked.
- **Budgets**: per-agent token counters in Redis (`TOKEN_LIMIT`), agent picked by the `X-Agent-Id` header.
- **Egress**: the agent sits on an `internal` Docker network; the proxy is its only reachable host.

## Proxy

```sh
cd proxy && go test ./...
```

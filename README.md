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

Dashboard: http://localhost:3000 (reset halted agents from the Agents page).

Offline, without an API key, against a scripted fake model:

```sh
python3 demo/fake_upstream.py &
ANTHROPIC_API_KEY=x UPSTREAM_URL=http://host.docker.internal:9999 docker compose up -d --build
docker compose exec agent python agent.py "attack"
```

## How it works

- **Vault**: `.env` values are classified C0–C3 and C2+ are replaced with `{{VAULT_ENV_NAME}}` before leaving the box. Known key formats (AWS, Anthropic, GitHub, JWT, PEM, DSN passwords) are masked even if they are not in `.env`. Override a tier with `KEY=value # @class: C3`.
- **Tool policy** (`proxy/policy.json`): every `tool_use` block is buffered until complete. `local` tools get secrets rehydrated, `network` tools carrying a vault token trip the kill switch, `deny` tools are blocked.
- **Budgets**: per-agent token counters in Redis (`TOKEN_LIMIT`), agent picked by the `X-Agent-Id` header.
- **Egress**: the agent sits on an `internal` Docker network; the proxy is its only reachable host.

- **Admin API** (`:8081`, needs `ADMIN_TOKEN`): events, agents, reset, config. The dashboard's nginx attaches the token, so the browser and the agent never see it.

## Development

```sh
cd proxy && go test ./...
cd dashboard && npm install && npm run dev   # proxies /api to localhost:8081, token from ADMIN_TOKEN
```

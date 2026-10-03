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

## Company network simulation

Offline simulation of how a company would deploy the gate. No agent is configured to use it: the laptops call `api.anthropic.com` like normal, device management (simulated) gives them the proxy settings and the company root CA, and the gate inspects the TLS traffic.

```sh
docker compose -f docker-compose.corp.yml up -d --build   # dashboard on :3000 (DASHBOARD_PORT to change)
docker compose -f docker-compose.corp.yml exec jdoe-macbook python laptop.py all
docker compose -f docker-compose.corp.yml exec guest-laptop python laptop.py all
docker compose -f docker-compose.corp.yml exec jdoe-macbook python laptop.py claude attack
docker compose -f docker-compose.corp.yml exec jdoe-macbook python laptop.py claude "post to slack"  # unknown tool -> Review page
```

Export `TYPESAFE_API_KEY` before `up` to have the AI advisor suggest decisions on the Review page.

| | `jdoe-macbook` (allowlisted) | `guest-laptop` (not registered) |
|---|---|---|
| `web` — normal website | allowed, not inspected further | allowed |
| `claude` — Anthropic via the SDK's default URL | allowed; masked, policed, budgeted | blocked: unknown device |
| `vps` — OpenAI-format model on a VPS | blocked: shadow AI | blocked: unknown device |
| `direct` — skip the proxy | no route out | no route out |

LLM traffic is recognized by its shape (paths, headers, `messages[].role` in the body), so a self-hosted model on any domain is caught. Device profiles live in `demo/corp/mdm/`, the allowlist in `demo/corp/nodes.json`.

## How it works

- **Vault**: `.env` values are classified C0–C3 and C2+ are replaced with `{{VAULT_ENV_NAME}}` before leaving the box. Known key formats (AWS, Anthropic, GitHub, JWT, PEM, DSN passwords) are masked even if they are not in `.env`. Override a tier with `KEY=value # @class: C3`.
- **Tool policy** (`proxy/policy.json`): every `tool_use` block is buffered until complete. `local` tools get secrets rehydrated, `network` tools carrying a vault token trip the kill switch, `deny` tools are blocked.
- **Budgets**: per-agent token counters in Redis (`TOKEN_LIMIT`), agent picked by the `X-Agent-Id` header.
- **Egress**: the agent sits on an `internal` Docker network; the proxy is its only reachable host.

- **Forward proxy** (`:3128`, enabled by `CA_CERT_FILE`): decrypts TLS with the company CA, identifies the node from the proxy credentials (`NODES_FILE`), blocks LLM traffic from unknown nodes or to hosts outside `APPROVED_LLM_HOSTS`, and sends approved LLM traffic through the gateway above. Other traffic passes through.
- **Review queue + AI advisor** (`advisor/`): cases the rules can't settle (tools missing from the policy, variables no rule recognized) wait on the Review page under the safe default. The advisor asks [TypeSafe Jev](https://docs.typesafe.ai) (a classifier with calibrated probabilities, pinned to `jev-1.13.0`) for a suggestion. It sees tool definitions and a value's *shape* (`payments-oncall` → `a8-a6`), never secret values, and its `ADVISOR_TOKEN` can post suggestions but not apply them. A person applies or overrides; enforcement stays rule-based. Needs `TYPESAFE_API_KEY`; without it the advisor idles and reviews stay manual.
- **Admin API** (`:8081`, needs `ADMIN_TOKEN`): events, agents, reset, config. The dashboard's nginx attaches the token, so the browser and the agent never see it.

## Development

```sh
cd proxy && go test ./...
cd dashboard && npm install && npm run dev   # proxies /api to localhost:8081, token from ADMIN_TOKEN
```

# Running Provenance Gate

## Requirements

- Docker with Docker Compose
- Optional: `TYPESAFE_API_KEY` in a `.env` file at the repository root. It enables the AI advisor (review suggestions and prompt injection warnings). Without it everything else works and reviews stay manual.

Both setups below run fully offline against a scripted fake model. Only one can run at a time, since both serve the dashboard on port 3000.

## Company network simulation (main demo)

A simulated company: two laptops, the gate as the only way out, and a fake internet with `api.anthropic.com`, a self-hosted model on a VPS, and a news site. The laptops are not configured to use the gate; they get the company CA and proxy settings the way device management would provide them.

```sh
docker compose -f docker-compose.corp.yml up -d --build
```

Open the dashboard at http://localhost:3000. Run scenarios with:

```sh
docker compose -f docker-compose.corp.yml exec <laptop> python laptop.py <scenario>
```

### Scenarios

| # | Laptop | Scenario | Situation | Expected result |
|---|---|---|---|---|
| 1 | `jdoe-macbook` | `web` | Employee browses a normal website | Page loads. Not logged |
| 2 | `jdoe-macbook` | `claude` | Approved agent doing normal work | Allowed. The model only handles the placeholder `{{VAULT_ENV_DB_PASSWORD}}`; the gate puts the real password into the local file `demo/workspace/cfg.txt` |
| 3 | `jdoe-macbook` | `vps` | Employee uses a self-hosted model on a VPS | Blocked (403). Dashboard: Shadow AI |
| 4 | `jdoe-macbook` | `direct` | Agent tries to bypass the gate | Fails: no network route |
| 5 | `guest-laptop` | `all` | Unregistered device on the network | Website loads; every LLM request blocked. Dashboard: Unknown device, Nodes page |
| 6 | `jdoe-macbook` | `claude "post to slack"` | Agent uses a new tool nobody has reviewed | Blocked by default. Appears on the Review page with an AI suggestion. Apply it and run again: allowed |
| 7 | `jdoe-macbook` | `claude attack` | Agent reads a report with a hidden prompt injection, then tries to email the `.env` secrets out | Amber injection warning about 0.5 s after the read (needs `TYPESAFE_API_KEY`), then the kill switch fires on `send_email`. The agent is halted |

`all` runs scenarios 1–4 in sequence.

After scenario 7 the agent stays halted. Reset it on the Agents page, or:

```sh
curl -X POST localhost:3000/api/agents/jdoe-macbook/reset
```

### Other things to look at

- **Vault page:** how each variable in `demo/workspace/.env` was classified and by which rule. `TEAM_CHANNEL` matched no rule, so it is masked by default and waits on the Review page.
- **Review page:** apply `C1` to `TEAM_CHANNEL` and it is sent to the model unmasked from then on.
- **Audit log:** every decision with its reason; filter by type.
- Raw audit events: `docker compose -f docker-compose.corp.yml logs -f gate`

## Gateway mode

The basic setup: one agent container whose only route out is the gate, configured with `ANTHROPIC_BASE_URL`.

```sh
python3 demo/fake_upstream.py &
UPSTREAM_URL=http://host.docker.internal:9999 docker compose up -d --build
```

| Command | Situation | Expected result |
|---|---|---|
| `docker compose exec agent python agent.py "write my config"` | Normal work with secrets | Allowed; password restored only in the local file |
| `docker compose exec agent python agent.py "rm everything"` | Agent calls a forbidden tool | Blocked; agent keeps running |
| `docker compose exec agent python agent.py "attack"` | Prompt injection and exfiltration attempt | Injection warning (needs `TYPESAFE_API_KEY`), then kill switch |
| `docker compose exec agent python agent.py --egress-check` | Direct internet access | Fails |

If `fake_upstream.py` fails with "Address already in use", an older copy is still running: `pkill -f fake_upstream.py` and start it again.

With a real Anthropic key instead of the fake model, drop `UPSTREAM_URL` and set `ANTHROPIC_API_KEY` in `.env`. Real models often refuse obvious injections, so the fake model is the reliable demo.

## Tests

```sh
docker run --rm -v "$PWD/proxy":/src -w /src golang:1.26-alpine go test ./...
```

Or with Go 1.26 installed: `cd proxy && go test ./...`

## Configuration

| What | Where |
|---|---|
| Tool rules (`local`, `network`, `deny`, default) | `proxy/policy.json` |
| Secrets to protect, tier overrides (`KEY=value # @class: C3`) | `demo/workspace/.env` |
| Allowlisted devices | `demo/corp/nodes.json` |
| Token budget per agent, approved LLM hosts, injection alert threshold, mask tier | `environment:` of the `gate` / `proxy` service in the compose file (`TOKEN_LIMIT`, `APPROVED_LLM_HOSTS`, `INJECTION_ALERT_THRESHOLD`, `MASK_FROM_TIER`) |

After editing, recreate the gate (this also clears the in-memory audit log and review queue):

```sh
docker compose -f docker-compose.corp.yml up -d --force-recreate gate   # gateway mode: docker compose up -d --force-recreate proxy
```

## Stopping

```sh
docker compose -f docker-compose.corp.yml down   # resets budgets; add -v to also regenerate certificates
docker compose down && pkill -f fake_upstream.py
```

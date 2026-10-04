# How to run HushGate

You need Docker. The commands use [`just`](https://github.com/casey/just); the line under each one shows the same thing without it. Run `just` on its own to list everything.

Optional: put `TYPESAFE_API_KEY=...` in a `.env` file in the repository root to turn on the AI advisor.

## Start and stop

Run the company network demo (offline):

```sh
just corp-up
# without just: docker compose -f docker-compose.corp.yml up -d --build
```

Stop the company network demo:

```sh
just corp-down
# without just: docker compose -f docker-compose.corp.yml down
```

Run the gateway with a fake AI model (offline):

```sh
just gateway-offline
# without just: python3 demo/fake_upstream.py &
#               UPSTREAM_URL=http://host.docker.internal:9999 docker compose up -d --build
```

Run the gateway with the real Claude API:

```sh
just gateway-up
# without just: docker compose up -d --build
```

Stop the gateway:

```sh
just gateway-down
# without just: docker compose down && pkill -f fake_upstream.py
```

Only one of the two setups can run at a time.

## Open the screens

Open the dashboard:

```sh
just dashboard
# or open http://localhost:3000
```

Open Prometheus (metrics):

```sh
just prometheus
# or open http://localhost:9090
```

## Try the company network demo

Start it first with `just corp-up`. Without `just`, each command below is `docker compose -f docker-compose.corp.yml exec <laptop> python laptop.py <scenario>`.

Browse a normal website (allowed):

```sh
just web
```

Let an approved agent work (secrets hidden from the AI):

```sh
just agent
```

Use an AI model on a private server (blocked as shadow AI):

```sh
just vps
```

Try to get around the gate (no route out):

```sh
just direct
```

Use AI from an unregistered laptop (blocked):

```sh
just guest
```

Use a tool nobody approved yet (blocked, sent for review):

```sh
just slack
```

Read a poisoned document and try to steal secrets (warning, then the agent is stopped):

```sh
just attack
```

Run an installer straight from the internet (blocked by an attack signature):

```sh
just install
```

Send a password out with curl (the agent is stopped):

```sh
just upload
```

Run the first four in one go:

```sh
just corp-all
```

Un-stop the laptop's agent after it was stopped:

```sh
just corp-reset
```

## Try the gateway demo

Start it first with `just gateway-offline`. Without `just`, each command below is `docker compose exec agent python agent.py "<task>"`.

Write a config file with a password (allowed, password stays local):

```sh
just gw-config
```

Use a forbidden tool (blocked):

```sh
just gw-forbidden
```

Read a poisoned document and try to steal secrets (agent stopped):

```sh
just gw-attack
```

Run an installer straight from the internet (blocked):

```sh
just gw-install
```

Send a password out with curl (agent stopped):

```sh
just gw-upload
```

Use a tool nobody approved yet (blocked, sent for review):

```sh
just gw-slack
```

Reach the internet directly from the agent (fails):

```sh
just gw-egress
```

Un-stop the agent after it was stopped:

```sh
just gw-reset
```

## Use your own Claude Code through the gate

Start the gateway with `just gateway-up`, then:

```sh
just claude-code
# without just: cd demo/workspace && ANTHROPIC_BASE_URL=http://localhost:8080 \
#               ANTHROPIC_CUSTOM_HEADERS="X-Agent-Id: my-claude-code" claude
```

## Use Claude Code inside the sandbox (the gate is its only way out)

Once, on your own machine: create a token for your Claude subscription and put it in `.env` in the repository root.

```sh
claude setup-token
# then add this line to .env:  CLAUDE_CODE_OAUTH_TOKEN=<the token>
```

Start the gateway (this also starts the sandboxed Claude Code):

```sh
just gateway-up
```

Open a Claude Code session in the sandbox:

```sh
just claude-sandbox
# without just: docker compose exec claude-code claude
```

For the live demo, use the rehearsed prompts in [DEMO.md](DEMO.md).

Check that it cannot reach the internet except through the gate:

```sh
just claude-sandbox-check
```

## Change the rules (applies within a second)

Change the policy (tools, masking, budgets, models, devices):

```sh
just policy
# or edit config/hushgate.yaml
```

Change the attack signatures:

```sh
just signatures
# or edit config/signatures.yaml
```

Change the protected secrets:

```sh
just vault
# or edit demo/workspace/.env
```

## Get reports

Download the audit log as CSV (add a filter like `blocked=1` to keep only what was stopped):

```sh
just export-csv
just export-csv blocked=1
# or use the Export CSV button on the Audit log page
```

Download the audit log as JSON:

```sh
just export-json
```

Show the raw metrics (add `corp` for the company network demo):

```sh
just metrics
just metrics corp
```

Watch the gate's live log:

```sh
just logs
just logs corp
```

## Make the launch video

Render the launch video (about 1.5 minutes) to `video/out/hushgate.mp4` (needs Node):

```sh
just video
# without just: cd video && npm install && npm run render
```

Preview and edit it in Remotion Studio:

```sh
just video-studio
```

## Run the tests

Run all tests (no Go needed):

```sh
just tests
# without just: docker run --rm -v "$PWD":/src -w /src/proxy golang:1.26-alpine go test ./...
```

Run the tests with Go installed:

```sh
just tests-local
# without just: cd proxy && go test -race ./...
```

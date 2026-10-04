# HushGate tasks. Run `just` to list them. Code lives in 5-implementation/.

impl := "5-implementation"

corp := "docker compose -f docker-compose.corp.yml"
gw := "docker compose"

# List all recipes
default:
    @just --list --unsorted

# --- Start and stop ----------------------------------------------------------

# Company network simulation, offline (dashboard http://localhost:3000)
[group('setup')]
corp-up:
    {{ corp }} up -d --build

# Stop the company network simulation
[group('setup')]
corp-down:
    {{ corp }} down

# Gateway mode against the real Anthropic API (key in .env, or your Claude Code login)
[group('setup')]
gateway-up:
    {{ gw }} up -d --build

# Gateway mode offline, against the scripted fake model
[group('setup')]
gateway-offline: fake-model
    UPSTREAM_URL=http://host.docker.internal:9999 {{ gw }} up -d --build

# Stop gateway mode and the fake model
[group('setup')]
gateway-down:
    {{ gw }} down
    -pkill -f fake_upstream.py

# (Re)start the scripted fake model on :9999, replacing any stale copy
[group('setup')]
fake-model:
    -pkill -f fake_upstream.py
    nohup python3 {{ impl }}/demo/fake_upstream.py > /tmp/hushgate-fake-model.log 2>&1 &
    sleep 1

# Same as gateway-offline
[group('setup')]
test: gateway-offline

# --- Company network scenarios (run corp-up first) ---------------------------

# 1. Employee browses a normal website: passes, not logged
[group('corp')]
web:
    {{ corp }} exec jdoe-macbook python laptop.py web

# 2. Approved agent at work: secrets masked for the model, real value in the local file
[group('corp')]
agent task="Summarize quarterly_report.md":
    {{ corp }} exec jdoe-macbook python laptop.py claude "{{ task }}"

# 3. Self-hosted model on a VPS: blocked as shadow AI
[group('corp')]
vps:
    {{ corp }} exec jdoe-macbook python laptop.py vps

# 4. Agent tries to bypass the gate: no network route
[group('corp')]
direct:
    {{ corp }} exec jdoe-macbook python laptop.py direct

# 5. Unregistered device: web works, every LLM request blocked
[group('corp')]
guest:
    {{ corp }} exec guest-laptop python laptop.py all

# 6. New tool nobody reviewed: blocked, sent to the Review page with an AI suggestion
[group('corp')]
slack:
    {{ corp }} exec jdoe-macbook python laptop.py claude "post to slack"

# 7. Poisoned report: injection warning, then the kill switch on send_email
[group('corp')]
attack:
    {{ corp }} exec jdoe-macbook python laptop.py claude attack

# 8. Installer piped into a shell: blocked by attack signature HG-RCE-001
[group('corp')]
install:
    {{ corp }} exec jdoe-macbook python laptop.py claude "install the dev tool"

# 9. curl sending the DB password: Bash guard treats it as a network call, kill switch
[group('corp')]
upload:
    {{ corp }} exec jdoe-macbook python laptop.py claude "upload the config"

# Scenarios 1-4 on the registered laptop, then the unregistered one
[group('corp')]
corp-all:
    {{ corp }} exec jdoe-macbook python laptop.py all
    {{ corp }} exec guest-laptop python laptop.py all

# Un-halt the registered laptop after a kill
[group('corp')]
corp-reset:
    curl -s -X POST localhost:3000/api/agents/jdoe-macbook/reset

# --- Gateway mode scenarios (run gateway-offline first) ----------------------

# Normal work with secrets: password restored only in the local file
[group('gateway')]
gw-config:
    {{ gw }} exec agent python agent.py "write my config"

# Forbidden tool: blocked, the agent keeps running
[group('gateway')]
gw-forbidden:
    {{ gw }} exec agent python agent.py "rm everything"

# Prompt injection and exfiltration attempt: warning, then kill switch
[group('gateway')]
gw-attack:
    {{ gw }} exec agent python agent.py attack

# Installer piped into a shell: attack signature
[group('gateway')]
gw-install:
    {{ gw }} exec agent python agent.py "install the dev tool"

# curl sending the DB password: Bash guard, kill switch
[group('gateway')]
gw-upload:
    {{ gw }} exec agent python agent.py "upload the config"

# Unknown tool: blocked, sent to the Review page
[group('gateway')]
gw-slack:
    {{ gw }} exec agent python agent.py "post to slack"

# Direct internet access from the agent: fails
[group('gateway')]
gw-egress:
    {{ gw }} exec agent python agent.py --egress-check

# Un-halt the demo agent after a kill
[group('gateway')]
gw-reset:
    curl -s -X POST localhost:3000/api/agents/demo-agent/reset

# Your own Claude Code through the gate (gateway-up first); uses your login or API key
[group('gateway')]
claude-code:
    cd {{ impl }}/demo/workspace && ANTHROPIC_BASE_URL=http://localhost:8080 ANTHROPIC_CUSTOM_HEADERS="X-Agent-Id: my-claude-code" claude

# Real Claude Code inside the sandbox: the gate is its only way out (gateway-up first)
[group('gateway')]
claude-sandbox:
    {{ gw }} exec claude-code claude

# Show that the sandboxed Claude Code can't reach the internet except through the gate
[group('gateway')]
claude-sandbox-check:
    @{{ gw }} exec -T claude-code curl -s -m 5 https://example.com -o /dev/null && echo "direct internet: REACHED" || echo "direct internet: blocked"
    @{{ gw }} exec -T claude-code curl -s -m 5 http://1.1.1.1 -o /dev/null && echo "raw IP: REACHED" || echo "raw IP: blocked"
    @{{ gw }} exec -T claude-code curl -s -m 5 http://proxy:8080/healthz > /dev/null && echo "HushGate gateway: reachable" || echo "HushGate gateway: NOT reachable"

# Un-halt the sandboxed Claude Code after a kill or a budget test
[group('gateway')]
claude-sandbox-reset:
    curl -s -X POST localhost:3000/api/agents/claude-code-sandbox/reset

# --- Policy (reloads live on save) --------------------------------------------

# Edit the policy file; changes apply within a second
[group('policy')]
policy:
    ${EDITOR:-vi} {{ impl }}/config/hushgate.yaml

# Edit the attack signature feed
[group('policy')]
signatures:
    ${EDITOR:-vi} {{ impl }}/config/signatures.yaml

# Edit the vault's protected secrets
[group('policy')]
vault:
    ${EDITOR:-vi} {{ impl }}/demo/workspace/.env

# --- Reporting ----------------------------------------------------------------

# Open the dashboard
[group('reporting')]
dashboard:
    open http://localhost:3000

# Open Prometheus
[group('reporting')]
prometheus:
    open http://localhost:9090

# Download the full audit trail as CSV (filters: kind=..., blocked=1, agent=..., q=...)
[group('reporting')]
export-csv filter="":
    curl -sfOJ "localhost:3000/api/audit/export?format=csv&{{ filter }}" && ls -t hushgate-audit-*.csv | head -1

# Download the full audit trail as JSON Lines
[group('reporting')]
export-json filter="":
    curl -sfOJ "localhost:3000/api/audit/export?format=jsonl&{{ filter }}" && ls -t hushgate-audit-*.jsonl | head -1

# Raw Prometheus metrics from the gate (stack=gateway or corp)
[group('reporting')]
metrics stack="gateway":
    {{ if stack == "corp" { corp } else { gw } }} exec prometheus wget -qO- --header "Authorization: Bearer demo-metrics-token" http://gate:8081/metrics | grep '^hushgate_'

# Follow the gate's audit events (stack=gateway or corp)
[group('reporting')]
logs stack="gateway":
    {{ if stack == "corp" { corp + " logs -f gate" } else { gw + " logs -f proxy" } }}

# --- Tests ----------------------------------------------------------------------

# Automated test suite in Docker (no Go needed)
[group('tests')]
tests:
    docker run --rm -v "$PWD":/src -w /src/{{ impl }}/proxy golang:1.26-alpine go test ./...

# End-to-end checks against the running company network demo (corp-up first)
[group('tests')]
e2e:
    python3 4-testing/e2e.py

# Microbenchmarks of every enforcement step (needs Go 1.26)
[group('tests')]
bench:
    cd {{ impl }}/proxy && go test ./internal/bench ./internal/gateway -run '^$' -bench . -benchmem

# Test suite with the race detector (needs Go 1.26)
[group('tests')]
tests-local:
    cd {{ impl }}/proxy && go test -race ./...

# Preview and edit the launch video in Remotion Studio (needs Node)
[group('video')]
video-studio:
    cd {{ impl }}/video && npm install && npm run studio

# Regenerate the voiceover clips with ElevenLabs Eleven v4 (needs ELEVENLABS_API_KEY)
[group('video')]
voiceover:
    cd {{ impl }}/video && python3 scripts/voiceover.py

# Render the launch video to 5-implementation/video/out/hushgate.mp4 (needs Node)
[group('video')]
video:
    cd {{ impl }}/video && npm install && npm run render

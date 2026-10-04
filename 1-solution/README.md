# 1 · Solution

**HushGate** is an AI control layer: a gateway between every AI agent and every model provider. All agent traffic passes through it. It applies one central policy to every request and every tool call, and it records every decision with the rule that made it.

Contents:
- [Overview and approach](#overview-and-approach)
- [Implemented controls and guardrails](#implemented-controls-and-guardrails)
- [Configuration](#configuration): a summary here, and the full reference in [configuration.md](configuration.md)
- [Example policies at different strictness levels](examples/)

## Overview and approach

### The problem

Coding agents such as Claude Code read source code, `.env` files and documents, run shell commands and call tools. Everything they read goes to an external model provider. That creates risks traditional security tools were not built for:

- secrets and client data sent to third parties;
- prompt injection turning an agent against its user;
- tool calls nobody checks;
- AI used from unregistered devices or on self-hosted models;
- runaway token spend;
- no audit trail.

### The approach

1. **Sit on the only path.** Agents reach models only through HushGate. Developers set one environment variable (`ANTHROPIC_BASE_URL`, or the base URL of any OpenAI-compatible client, such as one for OpenRouter). On a company network, the gate is a TLS-inspecting proxy installed by device management, so there is nothing to configure at all. A default-deny network ensures nothing goes around it.
2. **Deterministic rules decide.** Every enforcement decision is made by rules: masking, tool policy, signatures, the Bash guard, budgets and allowlists. The same input always gets the same decision, and every decision names its rule. All of them together add well under a millisecond (see [2-architecture](../2-architecture/performance.md)).
3. **AI advises where rules cannot decide.** A classifier (TypeSafe Jev) warns about prompt injection in what agents read, and suggests decisions for cases the rules can't settle. It never sees secret values and runs off the request path. It applies a decision only where the policy file explicitly allows it (`review.auto_accept`).
4. **Fail closed, but keep developers working.** Unknown tools are denied and unknown values are masked. Local tools still get the real values back, so the agent's work gets done; only data leaving the machine is controlled.
5. **One live policy file.** Every control is in [`hushgate.yaml`](../5-implementation/config/hushgate.yaml). It reloads within a second of saving, and an invalid edit is rejected while the previous policy stays active.
6. **Everything on record.** A live dashboard, a durable audit trail with CSV and JSON export, and Prometheus metrics (see [3-reporting](../3-reporting/)).

### What happens to one request

1. An agent sends a request to the model through the gate.
2. The gate checks the agent's identity, its token budget and the requested model.
3. The vault replaces secrets and personal data with placeholders, such as `{{VAULT_ENV_DB_PASSWORD}}` and `{{VAULT_IBAN_7F3A91C2}}`. The provider never receives the real values.
4. The answer streams back. Each tool call the model requests is held until it is complete. It is then checked by the tool policy, the Bash guard and 19 attack signatures before the agent sees it.
5. Approved local tools get the real values back. A call that would send protected data off the machine is dropped; if the data is a secret, the agent is also halted (the kill switch).
6. Every step goes to the audit log and to the metrics.

## Implemented controls and guardrails

| # | Control | What it stops | Kind | Configured in | Proven by |
|---|---|---|---|---|---|
| 1 | **Secret masking** (`.env` values) | Credentials reaching the model provider | Deterministic | `vault.env_files`, `masking.mask_from` | `vault.TestLoadMaskRehydrate`, e2e #2 |
| 2 | **Secret detectors** in any text | AWS, Anthropic, OpenAI, Stripe, GitHub and Slack keys, JWTs, PEM private keys and passwords in URLs, even if they're in no `.env` | Deterministic, format rules | built in | `vault.TestSecretFormatsDetected` |
| 3 | **Personal-data detectors** with checksums | IBAN (mod-97), payment cards (Luhn), PESEL (check digit), NIP (mod-11) sent to the model; checksums keep false positives near zero | Deterministic | built in | `vault.TestPersonalDataMasked`, `TestPersonalDataNoFalsePositives` |
| 4 | **Four confidentiality tiers with reasons** | Unexplained or inconsistent classification. Every variable gets C0–C3 and the rule that decided it; unknown values are masked (fail closed) | Deterministic | `vault.overrides`, `# @class:` in `.env` | `vault.TestClassifyWhy` |
| 5 | **Restoring real values into local tools only** | Broken agent work. Real values go back only into tools whose effects stay on the machine | Deterministic | `tools.rules` | `gateway.TestLocalToolRehydrated` |
| 6 | **Tool policy** (local / network / deny) | Agents calling tools they shouldn't; unknown tools are denied by default | Deterministic | `tools` | `gateway.TestDeniedToolBlockedNotKilled`, e2e #6, #11 |
| 7 | **Per-tier action on network tools** (allow / block / kill) | Protected data leaving through email, HTTP or Slack tools | Deterministic | `masking.on_network_tool` | `gateway.TestExfiltrationKills`, `policy.TestTierActionsConfigurable`, e2e #7 |
| 8 | **Kill switch** | A compromised agent continuing after an exfiltration attempt; it stays halted (stored in Redis) until reset | Deterministic | `on_network_tool: kill` | e2e #7, #9 |
| 9 | **Attack signatures** (19, central feed) | Remote code execution, destructive commands, credential theft, exfiltration endpoints, unsafe deserialization, model supply-chain attacks | Deterministic patterns, block, kill or alert | `signatures` and [`signatures.yaml`](../5-implementation/config/signatures.yaml) | `signature.TestBaselineFeed`, `gateway.TestSignatureBlocksPipeToShell`, e2e #8 |
| 10 | **Bash guard** | `curl`, `scp`, `ssh` and similar inside a shell command carrying secrets out, even when hidden as `c'u'rl`, behind `sudo`, `xargs` or `bash -c` | Deterministic command parsing | `bash_guard` | `shell.TestNetworkCommand`, `gateway.TestBashGuardKillsSecretToNetwork`, e2e #9 |
| 11 | **Token budgets per agent** | Runaway spend; a hard stop when the budget is used up | Deterministic | `budgets` | `gateway.TestBudgetExhausted`, e2e #12 |
| 12 | **Model allowlist** | Agents switching to unapproved or more expensive models | Deterministic, glob patterns | `models.allow` | `gateway.TestModelAllowlistAndPerAgentBudget`, e2e #13 |
| 13 | **Shadow AI detection** | LLM calls to unapproved or self-hosted models, recognised by request shape on any host | Deterministic: paths, headers, body | `llm_hosts.approved` | `detect.TestLLM`, `forward.TestShadowAIBlocked`, e2e #3 |
| 14 | **Device allowlist** | AI use from unregistered laptops or servers; identity comes from device credentials, not from the agent | Deterministic | `nodes` | `forward.TestUnregisteredNodeBlocked`, e2e #5 |
| 15 | **Egress lockdown** | Agents bypassing the gate; they have no network route except to it | Network (internal Docker networks) | compose files | e2e #4, `just claude-sandbox-check` |
| 16 | **Prompt injection early warning** | Hijacked agents going unnoticed. Everything an agent reads is scored; an alert comes about 0.3 s after the read. It never blocks | AI (TypeSafe Jev), alert only | `injection.alert_threshold` | `admin.TestScanResultsRaiseAlertsOnly`, e2e #7b |
| 17 | **AI-assisted review** | Slow manual triage of new tools and variables. Calibrated suggestions; a person decides | AI suggests, a person applies | — | `admin.TestAdvisorCanOnlySuggest` |
| 18 | **Auto-accept of confident suggestions** | Review backlog. Applied only above set probability and confidence, and only for outcomes on an allowed list (by default never loosening) | AI plus deterministic guard | `review.auto_accept` | `config.TestAutoAccept`, `admin.TestAutoAccept*` |
| 19 | **Live, validated policy** | Unsafe or broken policy edits; an invalid edit is rejected and the previous policy stays active | Deterministic | the file itself | `config.TestLiveReloadRejectAndRecover`, e2e #10 |
| 20 | **Separate credentials per role** | The AI advisor or the metrics scraper changing policy; admin, advisor and metrics tokens are separate | Deterministic | deployment env | `admin.TestAdvisorCanOnlySuggest`, `TestMetricsEndpointAuth` |

The full test catalogue is in [4-testing](../4-testing/).

## Configuration

Every control a security team tunes is in one file, [`5-implementation/config/hushgate.yaml`](../5-implementation/config/hushgate.yaml):

- **Live:** the gate reloads it within a second of saving, and the dashboard shows exactly what changed (for example `tools.rules.Bash: local → deny`).
- **Validated:** unknown keys and out-of-range values are rejected, and the previous policy stays active, so a typo never silently disables a control.
- **Single source of truth:** decisions made on the Review page, and auto-accepted AI suggestions, are written back into the file with its comments preserved.

Policies that can be enforced include:

- *No credential ever reaches a model provider*: `masking.mask_from: C2` (default), or `C1` for stricter environments.
- *Personal data may be used locally but never sent out*: `masking.on_network_tool.C2: block`.
- *An agent that tries to send a secret out is stopped*: `masking.on_network_tool.C3: kill`.
- *Only reviewed tools run*: `tools.default: deny`, plus `tools.rules`.
- *No shell command may reach the network with secrets*: `bash_guard`.
- *Known attack patterns are blocked company-wide from one feed*: `signatures.feeds`.
- *Each agent or device gets its own token budget*: `budgets.agents`.
- *Only approved models and providers*: `models.allow`, `llm_hosts.approved`.
- *Only enrolled devices may use AI*: `nodes`.
- *Confident AI suggestions may apply on their own, but never loosen protection*: `review.auto_accept`.

See [configuration.md](configuration.md) for every setting, its default and its effect, and [examples/](examples/) for ready-made **strict** and **relaxed** policies.

# 4 · Testing

The self-testing suite has three layers, each runnable with one command:

| Layer | What it proves | Command | Result |
|---|---|---|---|
| **74 automated tests** (Go) | Every control's allowed and blocked cases, false positives, and regressions found while testing with real Claude Code | `just tests` (Docker only) or `just tests-local` (race detector) | all pass |
| **19 end-to-end checks** | The running system: real containers, the simulated company network, live policy edits, metrics and exports | `just corp-up`, then `just e2e` | 19 / 19 pass |
| **9 benchmarks** | The cost of every deterministic check, and of the whole gate | `just bench` | see [2-architecture/performance.md](../2-architecture/performance.md) |

There are also **manual scenarios** for a live demo, and **rehearsed prompts for real Claude Code** in [DEMO.md](DEMO.md).

## End-to-end checks: `just e2e`

[`e2e.py`](e2e.py) (Python standard library only) runs every scenario on the simulated laptops. It reads the gate's own audit trail to confirm each decision, then edits the live policy file, checks that the change takes effect within seconds, and restores the file byte for byte.

| # | Test case | How | Expected (and observed) |
|---|---|---|---|
| 1 | Ordinary web traffic | registered laptop browses `news.example` | passes; not logged |
| 2 | Approved agent works with secrets | agent summarises a report and writes a config | the provider receives `{{VAULT_ENV_DB_PASSWORD}}`; the local `write_file` gets the real value |
| 3 | Shadow AI | laptop calls a self-hosted model on `llm.sketchy-vps.example` | `403`, `shadow_ai` event ("LLM request to unapproved host") |
| 4 | Bypassing the gate | agent connects straight to `api.anthropic.com` | connection fails: there is no route |
| 5 | Unregistered device | guest laptop uses an LLM, then browses | LLM traffic blocked (`node_blocked`); web works |
| 6 | Unreviewed tool | model calls `post_to_slack` | blocked by default; queued on the Review page |
| 7 | Exfiltration by email | poisoned report makes the agent email the `.env` secrets | `send_email` dropped, agent halted (kill switch) |
| 7b | Prompt injection | same report | AI warning, 98% (about 0.3 s after the read) |
| 8 | Remote code execution | model pipes an installer into a shell | blocked by signature HG-RCE-001 |
| 9 | Exfiltration by shell | `curl` sends the DB password | Bash guard treats it as a network tool; kill switch |
| 10 | Agent on OpenRouter | an OpenAI-SDK agent with the default OpenRouter URL writes a config | Chat Completions intercepted and inspected: placeholders sent, real value only in the local file |
| 11 | OpenRouter agent hijacked | the poisoned report over Chat Completions | `send_email` with secrets killed |
| 12 | Broken policy edit | a typo is saved into `hushgate.yaml` | rejected ("field toolz not found"); previous policy stays active |
| 13 | Live policy edit | `write_file: local → deny` | applied within a second; the next `write_file` is blocked |
| 14 | Token budget | the laptop's budget is set to 1 | request refused: "token budget exhausted" |
| 15 | Model allowlist | `models.allow` narrowed to `claude-opus-*` | `claude-haiku-4-5` refused |
| 16 | Performance metrics | `/api/metrics/summary` | per-stage percentiles recorded (request checks p50 193 µs) |
| 17 | Audit export | `/api/audit/export?format=csv&blocked=1` | CSV with every blocked and killed decision |
| 18 | Prometheus | query `hushgate_tool_calls_total` | the scrape shows allow, block and kill counts |

Output of the last run:

```
PASS  1. Ordinary web traffic passes and is not logged
PASS  2. Approved agent works; secrets masked before the model, real value restored locally
PASS  3. Self-hosted model on a private server blocked as shadow AI
PASS  4. Agent cannot bypass the gate (no network route out)
PASS  5. Unregistered device: web works, LLM traffic blocked
PASS  6. Unreviewed tool blocked by default and queued for review
PASS  7. Exfiltration of a secret by email: call dropped, agent halted (kill switch)
PASS  7b. Prompt injection in the poisoned report flagged by the AI advisor
PASS  8. Installer piped into a shell blocked by attack signature HG-RCE-001
PASS  9. curl sending the DB password: Bash guard treats it as a network call, kill switch
PASS  10. Agent on OpenRouter (Chat Completions): secrets masked, real value restored locally
PASS  11. Agent on OpenRouter hijacked by the poisoned report: email with secrets killed
PASS  12. Invalid policy edit rejected; previous policy stays active
PASS  13. Policy edit applies live: write_file denied
PASS  14. Per-agent token budget enforced (hard stop)
PASS  15. Model allowlist enforced: a model outside it is refused
PASS  16. Performance metrics recorded per stage
PASS  17. Audit trail exports as CSV (blocked and killed only)
PASS  18. Prometheus scrapes the gate's metrics

19 passed, 0 failed, 0 skipped
```

Check 7b needs `TYPESAFE_API_KEY` in `.env`. Without it, the check is reported as skipped, and everything else still passes.

## Automated tests: `just tests`

Run in Docker with no Go install: `docker run --rm -v "$PWD":/src -w /src/5-implementation/proxy golang:1.26-alpine go test ./...`. With Go 1.26: `cd 5-implementation/proxy && go test -race ./...`.

### Data protection (vault)
| Test | Proves |
|---|---|
| `vault.TestClassify`, `TestClassifyWhy` | `.env` variables get the right tier **and the right reason** (name rules, known formats, checksums, entropy, fail closed to C2) |
| `vault.TestLoadMaskRehydrate` | Secrets are masked in requests, including JSON-escaped forms, and restored exactly into local tool arguments |
| `vault.TestSecretFormatsDetected` | AWS, Anthropic, OpenAI, Stripe, GitHub and Slack keys, JWTs, PEM private keys and URL passwords are masked even when no `.env` lists them |
| `vault.TestDynamicDetection` | Detected secrets get stable placeholders across requests; a database URL keeps its user and masks only the password |
| `vault.TestPersonalDataMasked` | IBAN, card, PESEL and NIP values are masked, with and without spaces or dashes |
| `vault.TestPersonalDataNoFalsePositives` | **Not** masked: values with a bad checksum, timestamps, order and phone numbers, a valid PESEL inside a longer number, versions and ports |
| `vault.TestAdjacentNumbersBothMasked` | Two values next to each other are both found |
| `vault.TestHintsMatchFullSearch` | The fast windowed detectors find exactly what a full-text search finds, over 4,000 generated inputs |

### Tool calls and enforcement (gateway, policy, shell, signatures)
| Test | Proves |
|---|---|
| `gateway.TestLocalToolRehydrated` | A local tool gets the real value; the provider only ever saw the placeholder |
| `gateway.TestExfiltrationKills` | A secret in a network tool: call dropped, agent halted, later requests refused |
| `gateway.TestNetworkToolWithoutSecretAllowed` | Network tools without protected data work normally (no false blocking) |
| `gateway.TestDeniedToolBlockedNotKilled` | A denied tool is dropped, but the agent keeps running |
| `gateway.TestBudgetExhausted` | A used-up budget stops requests before they reach the provider |
| `gateway.TestModelAllowlistAndPerAgentBudget` | Models outside the allowlist are refused; per-agent budgets override the default |
| `gateway.TestUnknownToolsQueuedForReview` | Tools the policy doesn't know are blocked and queued, with their definitions, for review |
| `gateway.TestToolResultsQueuedForScan` | What agents read is queued for the injection scan, after masking |
| `gateway.TestToolResultsQueuedWhenSystemMessageFollows` | Regression found with real Claude Code: tool results are scanned even when a system message follows them |
| `gateway.TestBashGuardKillsSecretToNetwork` | `curl -d "pw={{VAULT_…}}"` in Bash is a network call: kill |
| `gateway.TestBashLocalCommandStillRehydrated` | Local shell commands still get real values (no false positive) |
| `gateway.TestSignatureBlocksPipeToShell` | `curl … \| sh` is blocked by HG-RCE-001 before the agent sees the call |
| `gateway.TestAlertSignatureAllowsAndRecords` | Alert-only signatures let the call through and record a hit |
| `gateway.TestGatewayRecordsMetrics` | Every decision and stage latency reaches the Prometheus metrics |
| `policy.TestDecide`, `TestTierActionsConfigurable` | local / network / deny × tier actions (allow, block, kill) give the expected decision |
| `shell.TestNetworkCommand` | The Bash guard sees `curl` through `c''url`, quotes, backslashes, full paths, `sudo -E`, `VAR=1`, pipes, `&&`, `$(…)`, `xargs` and `bash -c`, and ignores harmless commands |
| `signature.TestBaselineFeed` | Each of the 19 signatures matches its attack, and none matches harmless lookalikes |
| `signature.TestStrictestFirstAndDisabled` | When several signatures match, the strictest action wins; disabled signatures are skipped |
| `signature.TestParseFeedRejects` | Malformed feeds (bad regex, unknown action or severity, duplicate id, unknown key) are rejected |
| `signature.TestLoaderFallsBackToLastGood` | A URL feed that fails keeps its last good copy |
| `signature.TestLaterFeedOverridesID` | A later feed overrides an earlier one by signature id |
| `signature.TestFlatten` | Tool arguments are flattened to text the signatures can match |

### OpenAI-compatible Chat Completions (OpenAI, OpenRouter)
| Test | Proves |
|---|---|
| `gateway.TestChatLocalToolRehydrated` | Over Chat Completions the provider only sees placeholders, a local tool gets the real value reassembled from streamed fragments, and keep-alive comments and `[DONE]` pass through |
| `gateway.TestChatExfiltrationKills` | A secret in a network tool: the call is removed from the stream, a note replaces it, `finish_reason` becomes `stop`, the agent is halted and later requests get an OpenAI-format 403 |
| `gateway.TestChatParallelCalls` | Interleaved parallel calls are decided one by one: the local one is restored, the exfiltration is killed |
| `gateway.TestChatCallsWithoutIndex` | Providers that send whole calls without an index are still decided per call |
| `gateway.TestChatUsageCountedAndRequested` | The gate turns on usage reporting even if the agent switched it off, and counts the tokens against the budget |
| `gateway.TestChatNonStreaming` | Non-streamed answers: allowed calls restored, blocked ones replaced by a note, usage counted |
| `gateway.TestChatToolsAndResultsQueued` | Unknown function tools go to review; `role: tool` results go to the injection scan |
| `gateway.TestChatModelAllowlist` | OpenRouter-style model ids (`anthropic/claude-haiku-4.5`) match `*/claude-*`; others are refused |
| `gateway.TestChatSignatureBlocksPipeToShell` | Attack signatures and the Bash guard apply to Chat Completions tool calls too |

### Company network (forward proxy, LLM detection)
| Test | Proves |
|---|---|
| `forward.TestApprovedNodeGoesThroughGateway` | An allowlisted device's LLM traffic is decrypted and goes through every gateway control |
| `forward.TestUnregisteredNodeBlocked` | LLM traffic from a device not on the allowlist is refused |
| `forward.TestShadowAIBlocked` | LLM traffic to an unapproved host (a self-hosted model) is refused |
| `forward.TestWebPassesThrough` | Ordinary HTTPS traffic is tunnelled untouched and not logged |
| `forward.TestUntrustedDeviceFailsClosed` | A device without the company CA cannot use AI: TLS fails and nothing reaches the provider |
| `detect.TestLLM` | LLM requests are recognised by path, headers or body shape (Anthropic, OpenAI-compatible, Gemini, Ollama); ordinary requests are not |

### Policy file, review and AI (config, admin, scan)
| Test | Proves |
|---|---|
| `config.TestParseDefaults`, `TestLookups` | Safe defaults; budgets per agent, model globs and approved hosts |
| `config.TestParseRejects` | Unknown keys, bad values and out-of-range numbers are rejected, naming the setting |
| `config.TestLiveReloadRejectAndRecover` | A bad edit is rejected and the old policy stays; fixing the file applies it |
| `config.TestHalfWrittenFileIsNotApplied` | A file caught mid-save is never applied |
| `config.TestEditKeepsComments` | Decisions written back keep the file's comments and layout |
| `config.TestDiffRedactsTokens` | Reload diffs list every change but never print device tokens |
| `config.TestShippedPolicyIsValid`, `TestExampleProfilesAreValid` | The shipped policy and the strict and relaxed examples all load |
| `config.TestAutoAccept` | Auto-accept thresholds (inclusive), allowed outcomes, and a missing probability |
| `admin.TestAdvisorCanOnlySuggest` | The AI's credentials can read the queue and suggest, but cannot decide, reset agents or read config |
| `admin.TestHumanDecisionApplies` | A reviewer's decision becomes a rule in `hushgate.yaml` and takes effect |
| `admin.TestAutoAcceptOffByDefault` | With auto-accept off, a suggestion changes nothing |
| `admin.TestAutoAcceptAppliesConfidentSuggestion` | A confident suggestion is applied, written to the file and audited as `auto-accept` |
| `admin.TestAutoAcceptKeepsGuards` | `local` tools, unmasking and low-confidence suggestions still wait for a person |
| `admin.TestAutoAcceptAfterPolicyReload` | Turning auto-accept on applies suggestions that were already waiting |
| `admin.TestShapeHidesValue` | The advisor sees a value's shape, never the value |
| `admin.TestScanResultsRaiseAlertsOnly` | Injection scores raise alerts above the threshold and never block |
| `scan.TestObserveDedupesAndResolves`, `TestBounded` | Each document is scanned once per agent; the queue is bounded |

### Reporting (admin, audit, metrics)
| Test | Proves |
|---|---|
| `admin.TestAPI` | The dashboard API serves events, agents and config, and refuses requests without a token |
| `admin.TestAuditExport` | CSV and JSON exports with filters, containing placeholders and never values |
| `admin.TestMetricsEndpointAuth` | `/metrics` needs the metrics or admin token; the metrics token opens nothing else |
| `audit.TestFileRoundTripAndRestore` | The JSON Lines audit file is written and replayed into the dashboard after a restart |
| `metrics.TestEventsCounted`, `TestSummaryPercentiles`, `TestNilSafe` | Counters and percentiles are right, and metrics are optional |

## Manual scenarios

Every scenario is one command. They are also listed in [RUN.md](../5-implementation/RUN.md).

**Company network** (`just corp-up`): `just web`, `just agent`, `just vps`, `just direct`, `just guest`, `just slack`, `just attack`, `just install`, `just upload`, `just openrouter` (add `task=attack`), plus `just corp-reset`.

**Gateway mode** (`just gateway-offline`): `just gw-config`, `just gw-forbidden`, `just gw-slack`, `just gw-attack`, `just gw-install`, `just gw-upload`, `just gw-egress`, `just gw-openrouter` (add `task=attack`), plus `just gw-reset`.

**Real Claude Code** (`just gateway-up`, then `just claude-sandbox`): prompts rehearsed against a real model are in [DEMO.md](DEMO.md). They cover masking, the attack signatures, the Bash guard and live policy edits, and `just claude-sandbox-check` shows the container has no route out except the gate.

**Ad-hoc testing:** any prompt can be typed into the sandboxed Claude Code. Any change to [`hushgate.yaml`](../5-implementation/config/hushgate.yaml) applies within a second and shows up on the dashboard.

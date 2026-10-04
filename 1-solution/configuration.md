# Configuration reference

All policy lives in one YAML file. In the demo it is [`5-implementation/config/hushgate.yaml`](../5-implementation/config/hushgate.yaml), mounted into the gate as `/etc/hushgate/hushgate.yaml` (set `HUSHGATE_POLICY` to change the path).

## How changes apply

- The gate checks the file every second. A change is applied once the file has been stable for 200 ms, so a half-written save is never read.
- **Valid change:** applied immediately. The audit log and the dashboard show a "Policy reloaded" event listing exactly what changed, for example `masking.on_network_tool.C2: block → kill`.
- **Invalid change:** a typo in a key, an unknown value or a number out of range is rejected. A red banner names the problem, and the previous policy stays in force until the file is fixed. Unknown keys are errors on purpose, so a misspelled setting can never silently switch a control off.
- **Write-back:** decisions from the Review page and auto-accepted AI suggestions are written into the file, keeping comments and layout. The file stays the single source of truth and survives restarts.
- **Watched files:** editing a vault `.env` file listed under `vault.env_files`, or a local signature feed, also triggers a reload.

## Settings

| Setting | Values | Default | Effect |
|---|---|---|---|
| `tools.default` | `local`, `network`, `deny` | `deny` | What happens to a tool that has no rule. Unknown tools also go to the Review page. |
| `tools.rules.<tool>` | `local`, `network`, `deny` | — | `local`: the tool's effects stay on the machine, so real values are restored into its arguments. `network`: data leaves the machine, so placeholders stay and `on_network_tool` decides. `deny`: always blocked. |
| `masking.mask_from` | `C0`–`C3` | `C2` | Values at this tier or higher are replaced with placeholders before a request leaves. |
| `masking.on_network_tool.<tier>` | `allow`, `block`, `kill` | C0 `allow`, C1 `allow`, C2 `block`, C3 `kill` | What happens when protected data of that tier appears in a network tool call. `block` drops the call; `kill` also halts the agent until it is reset. |
| `vault.env_files` | list of paths | `[]` | `.env` files whose values are protected. Each variable is classified (name rules, known formats, checksums, entropy), with the reason recorded. |
| `vault.overrides.<NAME>` | `C0`–`C3` | — | Pins a variable's tier. Review decisions are written here. A trailing `# @class: C3` comment in the `.env` does the same. |
| `budgets.default_tokens` | integer, `0` = unlimited | `0` (the demo uses 200,000) | Token budget for any agent without its own. When it is used up, requests get `403` until the agent is reset. |
| `budgets.agents.<agent>` | integer | — | Budget for one agent (the `X-Agent-Id` header, or the device id in company-network mode). |
| `models.allow` | glob patterns | `[]` = any model | Models agents may call, for example `claude-haiku-*`. Other models get `403`. |
| `llm_hosts.approved` | hostnames | `[]` | Company-network mode: the only LLM providers devices may reach. LLM traffic to any other host (recognised by its shape) is blocked as shadow AI. |
| `nodes[]` | `id`, `owner`, `token` | `[]` | Company-network mode: devices allowed to use AI. The token comes from device management; LLM traffic from any other device is blocked. |
| `signatures.feeds` | file paths or URLs | `[]` | Attack signature feeds. A later feed overrides an earlier one with the same signature id. A feed that fails to load keeps its last good copy. |
| `signatures.refresh_seconds` | integer ≥ 10 | `300` (the demo uses 60) | How often URL feeds are fetched again. |
| `signatures.disabled` | signature ids | `[]` | Signatures to switch off, for example `[HG-DESER-001]`. |
| `bash_guard.tools` | tool names | `[Bash]` | Shell tools whose commands are inspected. |
| `bash_guard.network_commands` | program names | `curl, wget, nc, ncat, netcat, socat, scp, sftp, rsync, ssh, ftp, telnet, http, https` | A command running one of these is treated as a network tool. |
| `injection.alert_threshold` | 0–1 | `0.8` | AI prompt-injection score above which a warning is raised. It only alerts; it never blocks. |
| `review.auto_accept.enabled` | `true`, `false` | `false` | Lets the AI advisor's suggestion apply without a reviewer. Switching it on also re-checks suggestions already waiting. |
| `review.auto_accept.min_probability` | 0–1 | `0.95` | Minimum probability the advisor gives the suggested option. |
| `review.auto_accept.min_confidence` | 0–1 | `0.9` | Minimum calibrated confidence of the advisor. |
| `review.auto_accept.tools` | `local`, `network`, `deny` | `[network, deny]` | Tool outcomes that may be applied automatically. `local` is excluded by default because it would restore real secrets into a tool the agent itself described. |
| `review.auto_accept.variables` | `C0`–`C3` | `[C2, C3]` | Variable tiers that may be applied automatically. Unmasking (C0, C1) is excluded by default. |

## Confidentiality tiers

| Tier | Meaning | Examples | Default handling |
|---|---|---|---|
| C0 Public | Harmless | ports, flags, log levels | Sent as is |
| C1 Internal | Internal but not sensitive | hostnames, IDs, bucket names | Sent as is |
| C2 Confidential | Personal or business data, and anything unrecognised (fail closed) | IBAN, card, PESEL, NIP, client names | Masked; blocked in network tools |
| C3 Secret | Credentials | passwords, API keys, tokens, private keys | Masked; kill switch in network tools |

Classification rules run in this order:
1. a known secret format;
2. empty or boolean values;
3. secret words in the name (`PASSWORD`, `TOKEN`, `KEY`, `SECRET`, …);
4. checksum-validated personal data;
5. numeric or very short values;
6. internal or public words in the name;
7. high-entropy random-looking values;
8. otherwise C2, sent to review.

## Deployment settings

Ports, Redis, the upstream URL, certificates and tokens are deployment settings, not policy. They are environment variables in the compose files and are listed in [5-implementation](../5-implementation/README.md#deployment-settings).

## Strictness levels

[`examples/strict.yaml`](examples/strict.yaml) and [`examples/relaxed.yaml`](examples/relaxed.yaml) are complete policies for the two ends of the range. The shipped [`hushgate.yaml`](../5-implementation/config/hushgate.yaml) sits in between. A test checks that all three load.

| Setting | Strict (regulated production) | Shipped (default) | Relaxed (R&D sandbox, no client data) |
|---|---|---|---|
| Masked from tier | C1 | C2 | C3 |
| Internal data in a network tool | block | allow | allow |
| Personal data in a network tool | kill | block | allow |
| Secret in a network tool | kill | kill | block |
| Unknown tools | deny | deny | network (allowed, data still checked) |
| Web tools (WebFetch, WebSearch) | deny | network | network |
| Token budget per agent | 100,000 | 200,000 | 2,000,000 |
| Models | Haiku and Sonnet only | `claude-*` | any |
| Injection warning above | 0.6 | 0.8 | 0.9 |
| AI auto-accept | off | off | on, never loosening tools |

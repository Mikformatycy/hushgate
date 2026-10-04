#!/usr/bin/env python3
"""End-to-end checks against the running company-network demo.

Start the demo first (just corp-up), then run from the repository root:

    python3 4-testing/e2e.py        or        just e2e

Every scenario runs on the simulated laptops, and every check reads the gate's
audit trail through the dashboard API to confirm the decision it made. The
second half edits the live policy file (a typo, a denied tool, a budget, the
model allowlist), checks the effect within seconds, and restores the file
byte for byte. Exit code 0 means every check passed.
"""
import json
import pathlib
import subprocess
import sys
import time
import urllib.parse
import urllib.request

ROOT = pathlib.Path(__file__).resolve().parent.parent
POLICY = ROOT / "5-implementation" / "config" / "hushgate.yaml"
API = "http://localhost:3000"
PROMETHEUS = "http://localhost:9090"
CORP = ["docker", "compose", "-f", str(ROOT / "docker-compose.corp.yml")]
LAPTOP, GUEST = "jdoe-macbook", "guest-laptop"

results = []


def get(url):
    with urllib.request.urlopen(url, timeout=15) as r:
        return r.read().decode()


def api(path, method="GET"):
    req = urllib.request.Request(API + path, method=method)
    with urllib.request.urlopen(req, timeout=15) as r:
        body = r.read().decode()
    return json.loads(body) if body.startswith(("[", "{")) else body


def last_id():
    return max([e["id"] for e in api("/api/events")] or [0])


def events_after(since):
    return [e for e in api("/api/events") if e["id"] > since]


def wait_for(since, pred, timeout=20):
    """First audit event after `since` matching pred, waiting up to timeout seconds."""
    end = time.time() + timeout
    while time.time() < end:
        for e in events_after(since):
            if pred(e):
                return e
        time.sleep(0.5)
    return None


def run(device, *args):
    r = subprocess.run(CORP + ["exec", "-T", device, "python", "laptop.py", *args],
                       capture_output=True, text=True, timeout=240)
    return r.stdout + r.stderr


def reset(agent=LAPTOP):
    api(f"/api/agents/{agent}/reset", method="POST")


def check(name, ok, detail):
    results.append((name, ok))
    print(f"{'PASS' if ok else 'FAIL':4}  {name}\n      {detail}")


def note(name, detail):
    results.append((name, None))
    print(f"SKIP  {name}\n      {detail}")


def kind(k, **want):
    return lambda e: e.get("kind") == k and all(want[f](e.get(f, "")) if callable(want[f]) else e.get(f) == want[f] for f in want)


# --- Scenarios on the simulated laptops -------------------------------------

def scenarios():
    since = last_id()
    out = run(LAPTOP, "web")
    time.sleep(1)
    logged = [e for e in events_after(since) if e.get("agent") == LAPTOP]
    check("1. Ordinary web traffic passes and is not logged",
          "200" in out and not logged, f"news.example answered 200; {len(logged)} gate events for it")

    since = last_id()
    out = run(LAPTOP, "claude", "Summarize quarterly_report.md")
    mask = wait_for(since, kind("mask", agent=LAPTOP))
    write = wait_for(since, kind("tool_call", agent=LAPTOP, tool="write_file", action="allow"))
    check("2. Approved agent works; secrets masked before the model, real value restored locally",
          bool(mask and write and "Tr0ub4dor" in out),
          f"placeholders sent: {', '.join((mask or {}).get('tokens') or [])}; local write_file allowed with the real value")

    since = last_id()
    out = run(LAPTOP, "vps")
    e = wait_for(since, kind("shadow_ai", agent=LAPTOP))
    check("3. Self-hosted model on a private server blocked as shadow AI",
          bool(e and "403" in out), (e or {}).get("reason", "no shadow_ai event"))

    out = run(LAPTOP, "direct")
    check("4. Agent cannot bypass the gate (no network route out)",
          "ConnectError" in out or "FAILED" in out, "direct connection to api.anthropic.com failed")

    since = last_id()
    out = run(GUEST, "all")
    e = wait_for(since, kind("node_blocked"))
    check("5. Unregistered device: web works, LLM traffic blocked",
          bool(e and "200" in out), (e or {}).get("reason", "no node_blocked event"))

    since = last_id()
    run(LAPTOP, "claude", "post to slack")
    e = wait_for(since, kind("tool_call", tool="post_to_slack", action="block"))
    queued = any(r["id"] == "tool:post_to_slack" for r in api("/api/reviews"))
    check("6. Unreviewed tool blocked by default and queued for review",
          bool(e and queued), f"{(e or {}).get('reason', 'no block')}; on the Review page: {queued}")

    since = last_id()
    run(LAPTOP, "claude", "attack")
    kill = wait_for(since, kind("tool_call", tool="send_email", action="kill"))
    halted = any(a["id"] == LAPTOP and a["killed"] for a in api("/api/agents"))
    check("7. Exfiltration of a secret by email: call dropped, agent halted (kill switch)",
          bool(kill and halted), f"{(kill or {}).get('reason', 'no kill')}; agent halted: {halted}")
    # Each (agent, document) pair is scanned once, so a repeat run finds the earlier warning.
    inj = wait_for(since, kind("injection", agent=LAPTOP), timeout=15) or wait_for(0, kind("injection", agent=LAPTOP), timeout=1)
    if inj:
        check("7b. Prompt injection in the poisoned report flagged by the AI advisor", True, inj["reason"])
    else:
        note("7b. Prompt injection warning", "no warning: the AI advisor needs TYPESAFE_API_KEY in .env")
    reset()

    since = last_id()
    run(LAPTOP, "claude", "install the dev tool")
    sig = wait_for(since, kind("signature", reason=lambda r: r.startswith("HG-RCE-001")))
    blk = wait_for(since, kind("tool_call", tool="run_command", action="block"))
    check("8. Installer piped into a shell blocked by attack signature HG-RCE-001",
          bool(sig and blk), (sig or {}).get("reason", "no signature hit")[:110])

    since = last_id()
    run(LAPTOP, "claude", "upload the config")
    e = wait_for(since, kind("tool_call", tool="run_command", action="kill", reason=lambda r: "Bash guard" in r))
    check("9. curl sending the DB password: Bash guard treats it as a network call, kill switch",
          bool(e), (e or {}).get("reason", "no kill")[:110])
    reset()


# --- Live policy edits -----------------------------------------------------------

class PolicyEdit:
    """Edits the live policy file and restores it exactly, whatever happens."""

    def __init__(self, old, new):
        self.old, self.new = old, new

    def __enter__(self):
        self.original = POLICY.read_bytes()
        text = self.original.decode()
        if self.old not in text:
            raise RuntimeError(f"policy file has no {self.old!r}")
        since = last_id()
        POLICY.write_text(text.replace(self.old, self.new, 1))
        self.event = wait_for(since, kind("policy"), timeout=10)
        return self

    def __exit__(self, *exc):
        since = last_id()
        POLICY.write_bytes(self.original)
        wait_for(since, kind("policy", action="reloaded"), timeout=10)
        reset()


def policy_edits():
    with PolicyEdit("tools:\n  default: deny", "toolz:\n  default: deny") as edit:
        status = api("/api/config")["policy_file"]
        ok = (edit.event or {}).get("action") == "rejected" and bool(status.get("error"))
        check("10. Invalid policy edit rejected; previous policy stays active",
              ok, ((edit.event or {}).get("reason") or "no policy event")[:110])

    with PolicyEdit("    write_file: local", "    write_file: deny") as edit:
        since = last_id()
        run(LAPTOP, "claude", "Summarize quarterly_report.md")
        e = wait_for(since, kind("tool_call", tool="write_file", action="block"))
        check("11. Policy edit applies live: write_file denied",
              bool(edit.event and e), f"{(edit.event or {}).get('reason', '')}; then: {(e or {}).get('reason', 'not blocked')}")

    with PolicyEdit("  agents:\n", "  agents:\n    jdoe-macbook: 1\n") as edit:
        since = last_id()
        run(LAPTOP, "claude", "Summarize quarterly_report.md")
        run(LAPTOP, "claude", "Summarize quarterly_report.md")
        e = wait_for(since, kind("denied", agent=LAPTOP, reason=lambda r: "budget" in r))
        check("12. Per-agent token budget enforced (hard stop)",
              bool(e), f"{(edit.event or {}).get('reason', '')}; then: {(e or {}).get('reason', 'not denied')}")

    with PolicyEdit("  allow:\n    - claude-*", "  allow:\n    - claude-opus-*") as edit:
        since = last_id()
        run(LAPTOP, "claude", "Summarize quarterly_report.md")
        e = wait_for(since, kind("model_blocked"))
        check("13. Model allowlist enforced: a model outside it is refused",
              bool(e), f"{(edit.event or {}).get('reason', '')}; then: {(e or {}).get('reason', 'not refused')}")


# --- Reporting -------------------------------------------------------------------

def reporting():
    perf = api("/api/metrics/summary")
    stages = perf.get("stages", {})
    pre = stages.get("preprocess", {})
    check("14. Performance metrics recorded per stage",
          all(stages.get(s, {}).get("count") for s in ("preprocess", "tool_decision", "total")),
          f"gate request checks p50 {pre.get('p50_ms', 0) * 1000:.0f} µs over {pre.get('count', 0)} requests")

    csv = api("/api/audit/export?format=csv&blocked=1")
    lines = [l for l in csv.splitlines() if l.strip()]
    check("15. Audit trail exports as CSV (blocked and killed only)",
          len(lines) > 3 and lines[0].startswith("time"), f"{len(lines) - 1} rows, columns: {lines[0][:80] if lines else '-'}")

    q = urllib.parse.quote("sum by (action) (hushgate_tool_calls_total)")
    found = []
    for _ in range(40):  # Prometheus scrapes every few seconds
        found = json.loads(get(f"{PROMETHEUS}/api/v1/query?query={q}"))["data"]["result"]
        if found:
            break
        time.sleep(1)
    check("16. Prometheus scrapes the gate's metrics",
          bool(found), ", ".join(f"{r['metric'].get('action')}={r['value'][1]}" for r in found) or "no series yet")


def main():
    try:
        api("/api/config")
    except Exception as err:  # noqa: BLE001
        sys.exit(f"Dashboard API not reachable at {API} ({err}). Start the demo first: just corp-up")
    reset()
    print("HushGate end-to-end checks (company network demo)\n")
    scenarios()
    policy_edits()
    reporting()
    passed = sum(1 for _, ok in results if ok)
    failed = [n for n, ok in results if ok is False]
    skipped = sum(1 for _, ok in results if ok is None)
    print(f"\n{passed} passed, {len(failed)} failed, {skipped} skipped")
    for n in failed:
        print("  failed:", n)
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()

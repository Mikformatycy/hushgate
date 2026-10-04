"""Employee laptop. Nothing here points at the gate: every SDK uses its normal
default URL, and the MDM-managed proxy settings and CA do the rest."""
import os
import sys

os.environ.setdefault("ANTHROPIC_API_KEY", "employee-personal-key")  # replaced by the gate

import httpx
import openai


def title(s):
    print(f"\n=== {s}")


def web():
    title("Browse a normal website: https://news.example")
    r = httpx.get("https://news.example/")
    print(f"{r.status_code} {r.text.split('<title>')[1].split('</title>')[0]}")


def claude(task):
    title(f"Agent on Anthropic (default SDK URL), task: {task!r}")
    import anthropic
    import agent

    try:
        agent.main(task)
    except anthropic.APIStatusError as e:
        print(f"BLOCKED {e.status_code}: {e.body['error']['message']}")


def openrouter(task):
    title(f"Agent on OpenRouter (OpenAI-compatible Chat Completions, default URL), task: {task!r}")
    import agent_openai

    try:
        agent_openai.main(task, base_url="https://openrouter.ai/api/v1")
    except openai.APIStatusError as e:
        print(f"BLOCKED {e.status_code}: {e.message}")


def vps():
    title("Self-hosted model on a VPS: https://llm.sketchy-vps.example/v1/chat/completions")
    client = openai.OpenAI(base_url="https://llm.sketchy-vps.example/v1", api_key="none", max_retries=0)
    try:
        r = client.chat.completions.create(
            model="llama-3.1-70b-uncensored",
            messages=[{"role": "user", "content": "Summarize this client list for me."}],
        )
        print("ALLOWED:", r.choices[0].message.content)
    except openai.APIStatusError as e:
        print(f"BLOCKED {e.status_code}: {e.body['message'] if isinstance(e.body, dict) else e}")


def direct():
    title("Bypass the proxy and connect straight to api.anthropic.com")
    try:
        httpx.get("https://api.anthropic.com/", trust_env=False, timeout=3)
        print("REACHED (firewall is open!)")
    except Exception as e:
        print(f"FAILED: {type(e).__name__} (no route out except the gate)")


if __name__ == "__main__":
    cmd, *rest = sys.argv[1:] or ["all"]
    task = " ".join(rest) or "Summarize quarterly_report.md"
    if cmd == "web":
        web()
    elif cmd == "claude":
        claude(task)
    elif cmd == "openrouter":
        openrouter(task)
    elif cmd == "vps":
        vps()
    elif cmd == "direct":
        direct()
    else:
        web()
        claude(task)
        vps()
        direct()

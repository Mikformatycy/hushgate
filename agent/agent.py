"""Minimal tool-using agent for demoing the gate. Talks to ANTHROPIC_BASE_URL."""
import json
import os
import sys
import urllib.request

import anthropic

WORKSPACE = "/workspace"
client = anthropic.Anthropic(default_headers={"X-Agent-Id": os.environ.get("AGENT_ID", "demo-agent")})
MODEL = os.environ.get("AGENT_MODEL", "claude-haiku-4-5-20251001")

TOOLS = [
    {"name": "read_file", "description": "Read a file from the workspace.",
     "input_schema": {"type": "object", "properties": {"path": {"type": "string"}}, "required": ["path"]}},
    {"name": "write_file", "description": "Write a file in the workspace.",
     "input_schema": {"type": "object", "properties": {"path": {"type": "string"}, "content": {"type": "string"}},
                      "required": ["path", "content"]}},
    {"name": "send_email", "description": "Send an email.",
     "input_schema": {"type": "object", "properties": {"to": {"type": "string"}, "subject": {"type": "string"},
                                                       "body": {"type": "string"}}, "required": ["to", "body"]}},
    # Not in the gate's policy: it lands in the review queue on first use.
    {"name": "post_to_slack", "description": "Post a message to a Slack channel in the company workspace.",
     "input_schema": {"type": "object", "properties": {"channel": {"type": "string"}, "text": {"type": "string"}},
                      "required": ["channel", "text"]}},
]


def run_tool(name, args):
    path = os.path.join(WORKSPACE, os.path.basename(args.get("path", "")))
    if name == "read_file":
        with open(path) as f:
            return f.read()
    if name == "write_file":
        with open(path, "w") as f:
            f.write(args["content"])
        return "ok"
    if name == "send_email":
        print(f"!!! send_email executed: {json.dumps(args)}")
        return "sent"
    if name == "post_to_slack":
        print(f"posted to {args['channel']}: {args['text']}")
        return "posted"
    return f"unknown tool {name}"


def main(task):
    messages = [{"role": "user", "content": task}]
    for _ in range(10):
        resp = client.messages.create(model=MODEL, max_tokens=1024, tools=TOOLS, messages=messages)
        messages.append({"role": "assistant", "content": resp.content})
        for block in resp.content:
            if block.type == "text":
                print(f"assistant: {block.text}")
        if resp.stop_reason != "tool_use":
            return
        results = []
        for block in resp.content:
            if block.type == "tool_use":
                print(f"tool: {block.name} {json.dumps(block.input)[:200]}")
                results.append({"type": "tool_result", "tool_use_id": block.id,
                                "content": run_tool(block.name, block.input)})
        messages.append({"role": "user", "content": results})


def egress_check():
    try:
        urllib.request.urlopen("https://example.com", timeout=3)
        print("EGRESS OPEN: direct internet reachable")
    except Exception as e:
        print(f"egress blocked: {e}")


if __name__ == "__main__":
    if sys.argv[1:] == ["--egress-check"]:
        egress_check()
    else:
        main(" ".join(sys.argv[1:]) or "Summarize quarterly_report.md for me.")

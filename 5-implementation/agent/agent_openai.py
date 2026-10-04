"""The demo agent in OpenAI Chat Completions form, as most harnesses speak it to
OpenRouter and other OpenAI-compatible providers. Same tools as agent.py;
talks to OPENAI_BASE_URL."""
import json
import os
import sys

import openai

from agent import TOOLS, run_tool

MODEL = os.environ.get("OPENAI_AGENT_MODEL", "anthropic/claude-haiku-4.5")
FUNCTIONS = [{"type": "function", "function": {"name": t["name"], "description": t["description"],
                                                "parameters": t["input_schema"]}} for t in TOOLS]


def main(task, base_url=None):
    client = openai.OpenAI(base_url=base_url or os.environ.get("OPENAI_BASE_URL", "https://openrouter.ai/api/v1"),
                           api_key=os.environ.get("OPENAI_API_KEY", "employee-personal-key"), max_retries=0,
                           default_headers={"X-Agent-Id": os.environ.get("AGENT_ID", "demo-agent")})
    messages = [{"role": "user", "content": task}]
    for _ in range(10):
        content, calls, finish = "", {}, None
        for chunk in client.chat.completions.create(model=MODEL, messages=messages, tools=FUNCTIONS, stream=True):
            for choice in chunk.choices:
                delta = choice.delta
                content += delta.content or ""
                for tc in delta.tool_calls or []:
                    c = calls.setdefault(tc.index, {"id": "", "name": "", "arguments": ""})
                    c["id"] = tc.id or c["id"]
                    if tc.function:
                        c["name"] = tc.function.name or c["name"]
                        c["arguments"] += tc.function.arguments or ""
                finish = choice.finish_reason or finish
        if content:
            print(f"assistant: {content}")
        msg = {"role": "assistant", "content": content or None}
        if calls:
            msg["tool_calls"] = [{"id": c["id"], "type": "function",
                                  "function": {"name": c["name"], "arguments": c["arguments"]}}
                                 for _, c in sorted(calls.items())]
        messages.append(msg)
        if finish != "tool_calls" or not calls:
            return
        for call in msg["tool_calls"]:
            name, args = call["function"]["name"], json.loads(call["function"]["arguments"] or "{}")
            print(f"tool: {name} {json.dumps(args)[:200]}")
            messages.append({"role": "tool", "tool_call_id": call["id"], "content": run_tool(name, args)})


if __name__ == "__main__":
    main(" ".join(sys.argv[1:]) or "Summarize quarterly_report.md for me.")

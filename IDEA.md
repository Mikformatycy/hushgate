# Context: HackYeah 2026 - Goldman Sachs "AI Control Layer" Challenge

## The Objective
We are building an enterprise-grade "Agentic API Gateway" (codenamed Provenance Gate) in under 22 hours. The goal is to provide strict, deterministic security guardrails for AI agents without slowing down developer innovation. 

The sponsor (Goldman Sachs) evaluates based on three criteria:
1. **Sensitive Data Exposure:** Controlling data flow and masking secrets deterministically.
2. **Unauthorized Actions:** Enforcing tool-call permissions.
3. **Unpredictable Costs:** Hard limits and kill-switches for token/dollar budgets.

## System Architecture (MVP for 22-hour deadline)
The solution avoids probabilistic "LLM-as-a-judge" mechanisms in favor of deterministic controls.

*   **Core Backend (Go):** A high-performance reverse proxy that intercepts traffic between local agents (e.g., Claude Code, Python/LangChain) and the LLM providers (Anthropic/OpenAI API).
*   **Audit Store (PostgreSQL):** Immutable logging of every tool call attempted by the agent, tagged with allowed/blocked status.
*   **State & Budgets (Redis):** Fast counters tracking token usage per `agent_id` to enforce hard cost limits (circuit breakers).
*   **Observability UI (React/TypeScript):** A real-time dashboard displaying the audit logs, token burn rate, and a "flight recorder" of agent actions.
*   **Agent Environment (Docker):** The target agent runs in an isolated Docker container with strict network egress rules to prevent proxy bypassing.

## Technical Implementation Details

### 1. Integration Method (Zero-Code Drop-In)
We are using the **API Gateway Pattern**. Agents point to the proxy via standard environment variables (e.g., `ANTHROPIC_BASE_URL="http://proxy:8080"`). We decided *against* transparent LAN MITM proxying to avoid TLS/Certificate Authority issues.

### 2. Secret Masking Engine (The Vault)
*   **Detection:** The Go proxy scans outbound requests using regex heuristics and Shannon Entropy to detect known secrets and high-entropy strings (e.g., from local `.env` files).
*   **Tokenization:** Secrets are replaced with context-aware placeholders (e.g., `{{VAULT_ENV_DB_PASS}}`) before reaching the internet.
*   **Rehydration:** The proxy buffers the LLM's incoming SSE streams, identifies vault tokens, and swaps the real secrets back in before passing chunks to the local agent.

### 3. Egress Security (Anti-Bypass)
The agent container will have a "Deny-by-Default" network policy. Direct internet access is blocked via Docker networking/iptables. The only open egress path is to the local Go proxy. If a prompt-injected agent tries to open a direct socket to an external server, it will fail at the OS/network level.

## The Pitch / Live Demo Strategy
We are building towards a specific "Magic Moment" on stage:
1.  **Normal Flow:** Agent reads benign data, UI shows token budgets ticking down.
2.  **The Attack ("EchoLeak"):** A malicious document instructs the agent to send `.env` variables to `evil@hacker.com`.
3.  **The Block:** The agent attempts to use a `send_email` tool. The Go proxy detects the `{{VAULT_ENV_...}}` token in the tool's outbound arguments and triggers a kill-switch. 
4.  **The Proof:** The React UI flashes red, showing the exact millisecond the circuit breaker fired and the deterministic data provenance graph.

## Directives for Assisting Agent
1. **Focus on Speed:** We have less than 22 hours. Prioritize working MVP code over extensive abstraction.
2. **Tech Stack:** Default to Go for the proxy, React (Vite/Tailwind) for the frontend, Docker Compose for orchestration.
3. **Next Steps:** Begin scaffolding the Go reverse proxy capable of parsing SSE streams and the Docker Compose file to establish the isolated network boundary.
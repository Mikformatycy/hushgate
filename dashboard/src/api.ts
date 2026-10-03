export type GateEvent = {
  id: number
  time: string
  agent: string
  kind: 'mask' | 'tool_call' | 'usage' | 'denied' | 'reset' | 'shadow_ai' | 'node_blocked' | 'suggestion' | 'review' | 'injection' | 'policy' | 'model_blocked' | 'signature'
  host?: string
  tool?: string
  tool_id?: string
  action?: string
  reason?: string
  tokens?: string[]
  usage?: number
}

export type Agent = {
  id: string
  used: number
  limit: number
  killed: boolean
  kill_reason?: string
}

export type Config = {
  vault: { name: string; tier: 'C0' | 'C1' | 'C2' | 'C3'; reason: string }[]
  detectors: { name: string; tier: string; check: string }[]
  mask_from: string
  policy: {
    default: string
    tools: Record<string, 'local' | 'network' | 'deny'>
    on_network_tool: Record<string, 'allow' | 'block' | 'kill'>
  }
  token_limit: number
  budgets: Record<string, number> | null
  models: string[] | null
  llm_hosts: string[] | null
  injection_threshold: number
  nodes: { id: string; owner: string }[] | null
  policy_file: { path: string; version: string; loaded_at: string; error?: string }
  bash_guard: { tools: string[]; network_commands: string[] }
}

export type Signatures = {
  feeds: { source: string; name: string; version: string; count: number; error?: string }[] | null
  signatures:
    | {
        id: string
        name: string
        category: string
        severity: 'low' | 'medium' | 'high' | 'critical'
        action: 'alert' | 'block' | 'kill'
        tools: string[]
        description: string
        reference?: string
        enabled: boolean
      }[]
    | null
}

export type Review = {
  id: string
  kind: 'tool' | 'variable'
  subject: string
  context: Record<string, unknown>
  current: string
  options: string[]
  first_seen: string
  seen_by?: string
  suggestion?: {
    value: string
    probabilities?: Record<string, number>
    confidence: number
    rationale?: string
    model: string
    at: string
  }
  status: 'pending' | 'applied' | 'dismissed'
  decision?: string
  decided_at?: string
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path)
  if (!res.ok) throw new Error(`${path}: ${res.status}`)
  return res.json()
}

export const api = {
  events: (after: number) => get<GateEvent[]>(`/api/events?after=${after}`),
  agents: () => get<Agent[]>('/api/agents'),
  config: () => get<Config>('/api/config'),
  reviews: () => get<Review[]>('/api/reviews'),
  signatures: () => get<Signatures>('/api/signatures'),
  decide: async (id: string, action: 'apply' | 'dismiss', value?: string) => {
    const res = await fetch(`/api/reviews/${encodeURIComponent(id)}/decision`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action, value }),
    })
    if (!res.ok) throw new Error(`decision: ${res.status} ${await res.text()}`)
  },
  reset: async (id: string) => {
    const res = await fetch(`/api/agents/${encodeURIComponent(id)}/reset`, { method: 'POST' })
    if (!res.ok) throw new Error(`reset: ${res.status}`)
  },
}

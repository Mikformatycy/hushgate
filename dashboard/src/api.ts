export type GateEvent = {
  id: number
  time: string
  agent: string
  kind: 'mask' | 'tool_call' | 'usage' | 'denied' | 'reset'
  tool?: string
  tool_id?: string
  action?: 'allow' | 'block' | 'kill'
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
  vault: { name: string; tier: 'C0' | 'C1' | 'C2' | 'C3' }[]
  mask_from: string
  policy: { default: string; tools: Record<string, 'local' | 'network' | 'deny'> }
  token_limit: number
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
  reset: async (id: string) => {
    const res = await fetch(`/api/agents/${encodeURIComponent(id)}/reset`, { method: 'POST' })
    if (!res.ok) throw new Error(`reset: ${res.status}`)
  },
}

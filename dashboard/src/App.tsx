import { useMemo, useState, type ReactNode } from 'react'
import { api, type Agent, type Config, type GateEvent, type Review } from './api'
import { EventStatus, Meter, Mono, Panel, Status, Table, Td, TierBadge, eventMessage, time } from './ui'
import { useGate } from './useGate'

const pages = ['Dashboard', 'Agents', 'Nodes', 'Review', 'Audit log', 'Vault', 'Policy'] as const
type Page = (typeof pages)[number]

const pageFromHash = (): Page =>
  pages.find((p) => p.toLowerCase().replace(' ', '-') === window.location.hash.slice(1)) ?? 'Dashboard'

export default function App() {
  const { events, agents, config, reviews, error, refreshAgents, refreshReviews } = useGate()
  const pending = reviews.filter((r) => r.status === 'pending').length
  const [page, setPageState] = useState<Page>(pageFromHash)
  const setPage = (p: Page) => {
    window.location.hash = p.toLowerCase().replace(' ', '-')
    setPageState(p)
  }
  const [dismissed, setDismissed] = useState<Set<number>>(new Set())

  // An injection warning stays until dismissed, the agent is reset, or a kill for it supersedes it.
  const warnings = events.filter(
    (e) =>
      e.kind === 'injection' &&
      !dismissed.has(e.id) &&
      !events.some((r) => r.kind === 'reset' && r.agent === e.agent && r.id > e.id),
  )
  // A kill stays on screen until dismissed or the agent is reset.
  const kills = events.filter(
    (e) =>
      e.kind === 'tool_call' &&
      e.action === 'kill' &&
      !dismissed.has(e.id) &&
      !events.some((r) => r.kind === 'reset' && r.agent === e.agent && r.id > e.id),
  )

  return (
    <div className="min-h-screen">
      <header className="flex h-10 items-center gap-3 bg-nav px-4 text-white">
        <svg viewBox="0 0 24 24" className="h-5 w-5" aria-hidden>
          <path fill="#ff9900" d="M12 2 4 5v6c0 5 3.4 9.4 8 11 4.6-1.6 8-6 8-11V5l-8-3z" />
        </svg>
        <span className="font-bold">Provenance Gate</span>
        <span className="text-gray-400">AI Control Layer</span>
        <span className="ml-auto flex items-center gap-2 text-xs text-gray-300">
          <span className={`h-2 w-2 rounded-full ${error ? 'bg-bad' : 'bg-ok animate-pulse'}`} />
          {error ? 'Gate unreachable' : 'Live'}
        </span>
      </header>

      <div className="flex">
        <nav className="min-h-[calc(100vh-2.5rem)] w-52 shrink-0 border-r border-gray-200 bg-white py-4">
          <p className="px-5 pb-2 text-xs font-bold tracking-wide text-gray-500 uppercase">Monitoring</p>
          {pages.map((p) => (
            <button
              key={p}
              onClick={() => setPage(p)}
              className={`flex w-full items-center justify-between px-5 py-1.5 text-left hover:text-link ${
                page === p ? 'border-l-4 border-link pl-4 font-bold text-link' : 'text-gray-700'
              }`}
            >
              {p}
              {p === 'Review' && pending > 0 && (
                <span className="rounded-full bg-link px-2 text-xs font-bold text-white">{pending}</span>
              )}
            </button>
          ))}
        </nav>

        <main className="min-w-0 flex-1 p-6">
          {warnings.map((w) => (
            <div key={w.id} className="mb-3 flex items-start gap-3 rounded-lg border-2 border-warn bg-[#fff8e6] px-4 py-3">
              <span className="text-lg leading-5 text-[#8d6605]">⚠</span>
              <div className="flex-1">
                <p className="font-bold">
                  Possible prompt injection reached agent "{w.agent}"
                </p>
                <p className="text-sm text-gray-700">
                  <Mono>{w.tool}</Mono> {w.reason}. Early warning from the AI advisor: nothing is blocked by this alert,
                  the policy still decides every action.
                </p>
              </div>
              <button
                aria-label="Dismiss"
                onClick={() => setDismissed(new Set(dismissed).add(w.id))}
                className="px-1 text-lg leading-5"
              >
                ×
              </button>
            </div>
          ))}
          {kills.map((k) => (
            <div key={k.id} className="kill-flash mb-3 flex items-start gap-3 rounded-lg bg-bad px-4 py-3 text-white">
              <span className="text-lg leading-5">⊗</span>
              <div className="flex-1">
                <p className="font-bold">
                  Kill switch fired for agent "{k.agent}" at {time(k.time)}
                </p>
                <p className="text-sm">
                  Tool <b>{k.tool}</b> tried to send {k.tokens?.join(', ')} off the machine. The call was dropped and
                  the agent is halted.
                </p>
              </div>
              <button
                aria-label="Dismiss"
                onClick={() => setDismissed(new Set(dismissed).add(k.id))}
                className="px-1 text-lg leading-5"
              >
                ×
              </button>
            </div>
          ))}
          {error && (
            <div className="mb-3 rounded-lg border-l-4 border-bad bg-white px-4 py-3">
              <b>Cannot reach the gate admin API.</b> <span className="text-gray-600">{error}</span>
            </div>
          )}

          <p className="mb-1 text-sm text-gray-500">
            Provenance Gate <span className="mx-1">›</span> <span className="text-gray-800">{page}</span>
          </p>
          <h1 className="mb-5 text-2xl font-bold">{page}</h1>

          {page === 'Dashboard' && <Dashboard events={events} agents={agents} onReset={refreshAgents} />}
          {page === 'Agents' && <Agents agents={agents} onReset={refreshAgents} />}
          {page === 'Nodes' && <Nodes config={config} events={events} />}
          {page === 'Review' && <ReviewPage reviews={reviews} onDecided={refreshReviews} />}
          {page === 'Audit log' && <AuditLog events={events} />}
          {page === 'Vault' && <Vault config={config} />}
          {page === 'Policy' && <Policy config={config} />}
        </main>
      </div>
    </div>
  )
}

function Dashboard({ events, agents, onReset }: { events: GateEvent[]; agents: Agent[]; onReset: () => void }) {
  const stats = useMemo(() => {
    const calls = events.filter((e) => e.kind === 'tool_call')
    return {
      allowed: calls.filter((e) => e.action === 'allow').length,
      blocked: calls.filter((e) => e.action === 'block').length,
      killed: calls.filter((e) => e.action === 'kill').length,
      masked: events.filter((e) => e.kind === 'mask').reduce((n, e) => n + (e.tokens?.length ?? 0), 0),
      shadow: events.filter((e) => e.kind === 'shadow_ai').length,
      unknown: events.filter((e) => e.kind === 'node_blocked').length,
    }
  }, [events])
  const halted = agents.filter((a) => a.killed).length
  const tokens = agents.reduce((n, a) => n + a.used, 0)
  const security = events.filter((e) => e.kind !== 'usage').slice(-10).reverse()

  return (
    <>
      <div className="mb-5 grid grid-cols-2 gap-5 lg:grid-cols-5">
        <Metric label="Agents" value={agents.length}>
          <Status tone="ok">{agents.length - halted} running</Status>
          {halted > 0 && <Status tone="bad">{halted} halted</Status>}
        </Metric>
        <Metric label="Tool calls" value={stats.allowed + stats.blocked + stats.killed}>
          <Status tone="ok">{stats.allowed} allowed</Status>
          <Status tone="warn">{stats.blocked} blocked</Status>
          <Status tone="bad">{stats.killed} killed</Status>
        </Metric>
        <Metric label="LLM traffic blocked" value={stats.shadow + stats.unknown}>
          <Status tone="bad">{stats.shadow} shadow AI</Status>
          <Status tone="bad">{stats.unknown} unknown device</Status>
        </Metric>
        <Metric label="Secrets masked" value={stats.masked}>
          <span className="text-gray-500">placeholders sent instead of real values</span>
        </Metric>
        <Metric label="Tokens used" value={tokens.toLocaleString()}>
          <span className="text-gray-500">across all agents</span>
        </Metric>
      </div>
      <Panel title="Recent security events">
        <EventTable events={security} empty="No events yet. Point an agent at the gate to get started." />
      </Panel>
      <Agents agents={agents} onReset={onReset} />
    </>
  )
}

function Metric({ label, value, children }: { label: string; value: number | string; children: ReactNode }) {
  return (
    <div className="rounded-lg bg-white p-5 shadow-[0_1px_1px_rgba(0,28,36,.3),1px_1px_1px_rgba(0,28,36,.15)]">
      <p className="text-sm text-gray-600">{label}</p>
      <p className="my-1 text-4xl font-light text-link">{value}</p>
      <div className="flex flex-col gap-0.5 text-xs">{children}</div>
    </div>
  )
}

function Agents({ agents, onReset }: { agents: Agent[]; onReset: () => void }) {
  const [busy, setBusy] = useState<string | null>(null)
  const reset = async (id: string) => {
    setBusy(id)
    try {
      await api.reset(id)
      onReset()
    } finally {
      setBusy(null)
    }
  }
  return (
    <Panel title={`Agents (${agents.length})`}>
      <Table head={['Agent ID', 'Status', 'Token budget', 'Halt reason', 'Actions']} empty="No agents have called the gate yet.">
        {agents.map((a) => (
          <tr key={a.id} className={a.killed ? 'bg-red-50' : ''}>
            <Td className="font-medium whitespace-nowrap">{a.id}</Td>
            <Td>{a.killed ? <Status tone="bad">Halted</Status> : <Status tone="ok">Running</Status>}</Td>
            <Td>
              <Meter used={a.used} limit={a.limit} />
            </Td>
            <Td className="text-gray-700">{a.kill_reason ?? '—'}</Td>
            <Td>
              <button
                disabled={busy === a.id}
                onClick={() => reset(a.id)}
                className="rounded-full border-2 border-link px-4 py-0.5 font-bold text-link hover:bg-blue-50 disabled:opacity-50"
              >
                Reset
              </button>
            </Td>
          </tr>
        ))}
      </Table>
    </Panel>
  )
}

const filters: Record<string, (e: GateEvent) => boolean> = {
  'All events': () => true,
  'Tool calls': (e) => e.kind === 'tool_call',
  'Blocked and killed': (e) =>
    (e.kind === 'tool_call' && e.action !== 'allow') || ['denied', 'shadow_ai', 'node_blocked'].includes(e.kind),
  'Network (shadow AI, devices)': (e) => e.kind === 'shadow_ai' || e.kind === 'node_blocked',
  Masking: (e) => e.kind === 'mask',
  Usage: (e) => e.kind === 'usage',
}

function AuditLog({ events }: { events: GateEvent[] }) {
  const [filter, setFilter] = useState('All events')
  const [query, setQuery] = useState('')
  const shown = events
    .filter(filters[filter])
    .filter((e) => !query || `${e.agent} ${e.tool ?? ''} ${e.reason ?? ''}`.toLowerCase().includes(query.toLowerCase()))
    .slice(-500)
    .reverse()
  return (
    <Panel
      title={`Events (${shown.length})`}
      actions={
        <div className="flex gap-2">
          <input
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder="Find by agent, tool or reason"
            className="w-64 rounded border border-gray-400 px-2 py-1"
          />
          <select value={filter} onChange={(e) => setFilter(e.target.value)} className="rounded border border-gray-400 px-2 py-1">
            {Object.keys(filters).map((f) => (
              <option key={f}>{f}</option>
            ))}
          </select>
        </div>
      }
    >
      <EventTable events={shown} empty="No matching events." />
    </Panel>
  )
}

function EventTable({ events, empty }: { events: GateEvent[]; empty: string }) {
  return (
    <Table head={['Time', 'Agent', 'Result', 'Details']} empty={empty}>
      {events.map((e) => (
        <tr key={e.id} className={e.action === 'kill' ? 'bg-red-50' : ''}>
          <Td className="font-mono text-xs whitespace-nowrap text-gray-600">{time(e.time)}</Td>
          <Td className="whitespace-nowrap">{e.agent}</Td>
          <Td>
            <EventStatus e={e} />
          </Td>
          <Td>{eventMessage(e)}</Td>
        </tr>
      ))}
    </Table>
  )
}

function Vault({ config }: { config: Config | null }) {
  if (!config) return null
  const masked = (tier: string) => tier >= config.mask_from
  return (
    <>
      <Panel title={`Variables (${config.vault.length})`}>
        <p className="px-5 py-3 text-gray-600">
          Values from the loaded <Mono>.env</Mono> files. Anything classified {config.mask_from} or higher is replaced
          with a placeholder before it reaches the LLM provider. Every tier comes from a fixed rule, shown below; values
          are never shown here.
        </p>
        <Table head={['Name', 'Classification', 'Decided by', 'Sent to LLM as']} empty="No .env files loaded.">
          {config.vault.map((v) => (
            <tr key={v.name}>
              <Td className="font-medium whitespace-nowrap">{v.name}</Td>
              <Td className="whitespace-nowrap">
                <TierBadge tier={v.tier} />
              </Td>
              <Td className="text-gray-700">{v.reason}</Td>
              <Td>
                {masked(v.tier) ? (
                  <Mono>{`{{VAULT_ENV_${v.name.toUpperCase()}}}`}</Mono>
                ) : (
                  <span className="text-gray-500">Real value</span>
                )}
              </Td>
            </tr>
          ))}
        </Table>
      </Panel>
      <Panel title={`Detectors (${config.detectors?.length ?? 0})`}>
        <p className="px-5 py-3 text-gray-600">
          Applied to every outgoing request, so values are caught even when they were never in a <Mono>.env</Mono> file.
          Personal data detectors only fire when the checksum is valid. Credentials in a network tool call trip the kill
          switch; personal data blocks the call.
        </p>
        <Table head={['Detector', 'Classification', 'Validation']}>
          {(config.detectors ?? []).map((d) => (
            <tr key={d.name}>
              <Td>
                <Mono>{d.name}</Mono>
              </Td>
              <Td className="whitespace-nowrap">
                <TierBadge tier={d.tier} />
              </Td>
              <Td className="text-gray-700">{d.check}</Td>
            </tr>
          ))}
        </Table>
      </Panel>
    </>
  )
}

const sinkInfo = {
  local: { tone: 'ok', label: 'Local', text: 'Allowed. Vaulted values are restored in its arguments.' },
  network: { tone: 'warn', label: 'Network', text: 'Allowed without secrets. A vault placeholder in its arguments fires the kill switch.' },
  deny: { tone: 'bad', label: 'Denied', text: 'Always blocked.' },
} as const

function Nodes({ config, events }: { config: Config | null; events: GateEvent[] }) {
  if (!config) return null
  const nodes = config.nodes ?? []
  const lastSeen = (id: string) => [...events].reverse().find((e) => e.agent === id)
  const blocked = new Map<string, GateEvent>()
  for (const e of events) if (e.kind === 'node_blocked') blocked.set(e.agent, e)
  return (
    <>
      <Panel title={`Allowlisted nodes (${nodes.length})`}>
        <p className="px-5 py-3 text-gray-600">
          Devices and workloads allowed to reach LLM providers. Identity comes from the device (MDM credentials or
          certificate), not from anything the agent says about itself.
        </p>
        <Table head={['Node', 'Owner', 'Status', 'Last activity']} empty="No allowlist configured.">
          {nodes.map((n) => {
            const e = lastSeen(n.id)
            return (
              <tr key={n.id}>
                <Td className="font-medium whitespace-nowrap">{n.id}</Td>
                <Td>{n.owner}</Td>
                <Td>
                  <Status tone="ok">Allowlisted</Status>
                </Td>
                <Td className="text-gray-600">{e ? time(e.time) : 'Not seen yet'}</Td>
              </tr>
            )
          })}
        </Table>
      </Panel>
      <Panel title={`Blocked devices (${blocked.size})`}>
        <p className="px-5 py-3 text-gray-600">
          Devices that sent LLM traffic without being on the allowlist. Ordinary web traffic from them is not affected.
        </p>
        <Table head={['Device', 'Last attempt', 'Destination', 'Detected by']} empty="No unknown devices have tried to use an LLM.">
          {[...blocked.values()].reverse().map((e) => (
            <tr key={e.agent} className="bg-red-50">
              <Td className="font-medium whitespace-nowrap">{e.agent.replace('unregistered: ', '')}</Td>
              <Td className="font-mono text-xs whitespace-nowrap text-gray-600">{time(e.time)}</Td>
              <Td>
                <Mono>{e.host}</Mono>
              </Td>
              <Td className="text-gray-700">{e.reason?.match(/\(([^)]*)\)$/)?.[1]}</Td>
            </tr>
          ))}
        </Table>
      </Panel>
    </>
  )
}

function ReviewPage({ reviews, onDecided }: { reviews: Review[]; onDecided: () => void }) {
  const pending = reviews.filter((r) => r.status === 'pending')
  const resolved = reviews.filter((r) => r.status !== 'pending').reverse()
  return (
    <>
      <div className="mb-5 rounded-lg border-l-4 border-[#7d4dc0] bg-white px-5 py-3 text-gray-700 shadow-[0_1px_1px_rgba(0,28,36,.3)]">
        <b>AI suggests, a person decides, the rules enforce.</b> These are cases the deterministic rules could not settle:
        tools with no policy entry and variables no rule recognized. Until someone decides, the safe default applies. The
        AI advisor (TypeSafe Jev, a classifier with calibrated probabilities) never sees secret values, only tool
        definitions and a value's shape, and its credentials can post suggestions but cannot apply them.
      </div>
      <Panel title={`Pending review (${pending.length})`}>
        {pending.length === 0 && <p className="px-5 py-8 text-center text-gray-500">Nothing waiting for review.</p>}
        {pending.map((r) => (
          <ReviewCard key={r.id} r={r} onDecided={onDecided} />
        ))}
      </Panel>
      <Panel title={`Decided (${resolved.length})`}>
        <Table head={['Item', 'Decision', 'AI suggested', 'Decided at']} empty="No decisions yet.">
          {resolved.map((r) => (
            <tr key={r.id}>
              <Td>
                <span className="mr-2 text-xs text-gray-500 uppercase">{r.kind}</span>
                <Mono>{r.subject}</Mono>
              </Td>
              <Td>{r.status === 'dismissed' ? <span className="text-gray-500">Kept: {r.current}</span> : <b>{r.decision}</b>}</Td>
              <Td>
                {r.suggestion ? (
                  r.suggestion.value === r.decision ? (
                    <Status tone="ok">{r.suggestion.value} (accepted)</Status>
                  ) : (
                    <Status tone="warn">{r.suggestion.value} (overridden)</Status>
                  )
                ) : (
                  <span className="text-gray-500">—</span>
                )}
              </Td>
              <Td className="font-mono text-xs text-gray-600">{r.decided_at ? time(r.decided_at) : ''}</Td>
            </tr>
          ))}
        </Table>
      </Panel>
    </>
  )
}

const optionLabel: Record<string, string> = {
  local: 'Local (secrets restored)',
  network: 'Network (secrets block or kill)',
  deny: 'Deny',
  C0: 'C0 · Public',
  C1: 'C1 · Internal',
  C2: 'C2 · Confidential',
  C3: 'C3 · Secret',
}

function ReviewCard({ r, onDecided }: { r: Review; onDecided: () => void }) {
  const [value, setValue] = useState(r.suggestion?.value ?? r.options[r.options.length - 1])
  const [busy, setBusy] = useState(false)
  const [err, setErr] = useState<string | null>(null)
  // Pick up a suggestion that arrives while the card is open, unless the user changed the value.
  const [touched, setTouched] = useState(false)
  if (!touched && r.suggestion && value !== r.suggestion.value) setValue(r.suggestion.value)

  const decide = async (action: 'apply' | 'dismiss') => {
    setBusy(true)
    setErr(null)
    try {
      await api.decide(r.id, action, value)
      onDecided()
    } catch (e) {
      setErr(String(e))
    } finally {
      setBusy(false)
    }
  }
  const ctx = r.context as Record<string, unknown>
  return (
    <div className="grid gap-5 border-b border-gray-200 px-5 py-4 last:border-b-0 lg:grid-cols-[1fr_1fr_auto]">
      <div className="min-w-0">
        <p className="text-xs font-bold text-gray-500 uppercase">
          {r.kind === 'tool' ? 'Tool with no policy rule' : 'Variable no rule recognized'}
        </p>
        <p className="my-1 text-lg">
          <Mono>{r.subject}</Mono>
        </p>
        {r.kind === 'tool' ? (
          <>
            <p className="text-gray-700">{String(ctx.description ?? '')}</p>
            <details className="mt-1 text-xs text-gray-600">
              <summary className="cursor-pointer text-link">Input schema</summary>
              <pre className="mt-1 overflow-x-auto rounded bg-gray-50 p-2">{JSON.stringify(ctx.input_schema, null, 2)}</pre>
            </details>
          </>
        ) : (
          <p className="text-gray-700">
            Value shape: <Mono>{String(ctx.pattern)}</Mono> · {String(ctx.length)} chars ·{' '}
            {(ctx.classes as string[] | undefined)?.join(', ')} · {String(ctx.entropy)} bits/char
          </p>
        )}
        <p className="mt-2 text-sm text-gray-600">
          Today: <b>{r.current}</b>
          {r.seen_by && <> · first used by {r.seen_by}</>} · since {time(r.first_seen)}
        </p>
      </div>

      <div className="rounded-lg border border-[#d9c8f0] bg-[#f8f4fd] p-3">
        <p className="mb-1 text-xs font-bold tracking-wide text-[#5a2d91] uppercase">✦ AI suggestion</p>
        {r.suggestion ? (
          <>
            <p className="font-bold">{optionLabel[r.suggestion.value] ?? r.suggestion.value}</p>
            {r.suggestion.rationale && <p className="text-gray-700">{r.suggestion.rationale}</p>}
            <div className="mt-2 space-y-1">
              {r.options.map((o) => {
                const p = r.suggestion?.probabilities?.[o] ?? 0
                return (
                  <div key={o} className="flex items-center gap-2 text-xs">
                    <span className="w-14 shrink-0 font-mono">{o}</span>
                    <div className="h-2 flex-1 rounded bg-white">
                      <div
                        className={`h-2 rounded ${o === r.suggestion?.value ? 'bg-[#7d4dc0]' : 'bg-[#cdb8ea]'}`}
                        style={{ width: `${p * 100}%` }}
                      />
                    </div>
                    <span className="w-10 text-right tabular-nums">{(p * 100).toFixed(0)}%</span>
                  </div>
                )
              })}
            </div>
            <p className="mt-2 text-xs text-gray-500">
              TypeSafe {r.suggestion.model} · {(r.suggestion.confidence * 100).toFixed(0)}% confidence (calibrated)
            </p>
          </>
        ) : (
          <p className="text-gray-500">Waiting for the advisor. You can decide without it.</p>
        )}
      </div>

      <div className="flex flex-col gap-2 lg:w-56">
        <select
          value={value}
          onChange={(e) => {
            setTouched(true)
            setValue(e.target.value)
          }}
          className="rounded border border-gray-400 px-2 py-1"
        >
          {r.options.map((o) => (
            <option key={o} value={o}>
              {optionLabel[o] ?? o}
              {o === r.suggestion?.value ? ' ✦' : ''}
            </option>
          ))}
        </select>
        <button
          disabled={busy}
          onClick={() => decide('apply')}
          className="rounded-full bg-warn px-4 py-1 font-bold text-[#16191f] hover:brightness-95 disabled:opacity-50"
        >
          Apply
        </button>
        <button
          disabled={busy}
          onClick={() => decide('dismiss')}
          className="rounded-full border-2 border-link px-4 py-0.5 font-bold text-link hover:bg-blue-50 disabled:opacity-50"
        >
          Keep current rule
        </button>
        {err && <p className="text-xs text-bad">{err}</p>}
      </div>
    </div>
  )
}

function Policy({ config }: { config: Config | null }) {
  if (!config) return null
  const tools = Object.entries(config.policy.tools).sort(([a], [b]) => a.localeCompare(b))
  const def = sinkInfo[config.policy.default as keyof typeof sinkInfo]
  return (
    <Panel title={`Tool rules (${tools.length})`}>
      <p className="px-5 py-3 text-gray-600">
        Every tool call the model requests is checked before the agent sees it. Tools not listed are treated as{' '}
        <b>{def?.label ?? config.policy.default}</b>.
      </p>
      <Table head={['Tool', 'Type', 'Behavior']}>
        {tools.map(([name, sink]) => {
          const info = sinkInfo[sink]
          return (
            <tr key={name}>
              <Td>
                <Mono>{name}</Mono>
              </Td>
              <Td>
                <Status tone={info.tone}>{info.label}</Status>
              </Td>
              <Td className="text-gray-700">{info.text}</Td>
            </tr>
          )
        })}
      </Table>
    </Panel>
  )
}

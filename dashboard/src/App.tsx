import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { api, type Agent, type Config, type GateEvent, type MetricsSummary, type Review, type Signatures } from './api'
import { EventStatus, Meter, Mono, Panel, Status, Table, Td, TierBadge, eventMessage, time } from './ui'
import { useGate } from './useGate'

const pages = ['Dashboard', 'Agents', 'Nodes', 'Review', 'Audit log', 'Vault', 'Signatures', 'Policy'] as const
type Page = (typeof pages)[number]

const pageFromHash = (): Page =>
  pages.find((p) => p.toLowerCase().replace(' ', '-') === window.location.hash.slice(1)) ?? 'Dashboard'

export default function App() {
  const { events, agents, config, reviews, perf, error, refreshAgents, refreshReviews } = useGate()
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
      <header className="flex h-16 items-center gap-3 border-b border-line bg-white px-6">
        <svg viewBox="0 0 24 24" className="h-7 w-7" aria-hidden>
          <path className="fill-gs" d="M12 2 4 5v6c0 5 3.4 9.4 8 11 4.6-1.6 8-6 8-11V5l-8-3z" />
        </svg>
        <span className="text-lg font-extrabold tracking-tight">HushGate</span>
        <span className="text-sub">AI Control Layer</span>
        <span
          className={`ml-auto flex items-center gap-2 rounded-full px-3 py-1 text-xs font-bold ${
            error ? 'bg-bad-tint text-bad' : 'bg-ok-tint text-ok'
          }`}
        >
          <span className={`h-2 w-2 rounded-full ${error ? 'bg-bad' : 'bg-ok animate-pulse'}`} />
          {error ? 'Gate unreachable' : 'Live'}
        </span>
      </header>

      <div className="flex">
        <nav className="min-h-[calc(100vh-4rem)] w-60 shrink-0 border-r border-line bg-white px-3 py-5">
          <p className="px-3 pb-2 text-xs font-bold tracking-wider text-sub uppercase">Monitoring</p>
          {pages.map((p) => (
            <button
              key={p}
              onClick={() => setPage(p)}
              className={`mb-0.5 flex w-full items-center gap-3 rounded-xl px-3 py-2 text-left ${
                page === p ? 'bg-gs-tint font-bold text-gs-deep' : 'text-sub hover:bg-page hover:text-ink'
              }`}
            >
              <NavIcon page={p} />
              <span className="flex-1">{p}</span>
              {p === 'Review' && pending > 0 && (
                <span className="rounded-full bg-gs px-2 text-xs font-bold text-white">{pending}</span>
              )}
            </button>
          ))}
        </nav>

        <main className="min-w-0 flex-1 p-8">
          {warnings.map((w) => (
            <div key={w.id} className="mb-3 flex items-start gap-3 rounded-2xl border border-warn/40 bg-warn-tint px-5 py-4">
              <span className="text-lg leading-5 text-warn">⚠</span>
              <div className="flex-1">
                <p className="font-bold">
                  Prompt injection in <Mono>{w.tool}</Mono>
                </p>
                <p className="text-sm text-sub">
                  Read by agent "{w.agent}": {w.reason}. The agent is the target of this attack, not its source. This
                  warning blocks nothing; the policy still decides every action.
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
            <div key={k.id} className="kill-flash mb-3 flex items-start gap-3 rounded-2xl bg-bad px-5 py-4 text-white">
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
          {config?.policy_file.error && (
            <div className="mb-3 flex items-start gap-3 rounded-2xl bg-bad px-5 py-4 text-white">
              <span className="text-lg leading-5">⊗</span>
              <div>
                <p className="font-bold">
                  Policy file rejected, previous policy still active (version {config.policy_file.version})
                </p>
                <p className="text-sm">
                  {config.policy_file.path}: {config.policy_file.error}
                </p>
              </div>
            </div>
          )}
          {error && (
            <div className="mb-3 rounded-2xl border border-bad/40 bg-bad-tint px-5 py-4">
              <b>Cannot reach the gate admin API.</b> <span className="text-sub">{error}</span>
            </div>
          )}

          <p className="mb-1 text-sm text-sub">
            HushGate <span className="mx-1">/</span> <span className="text-ink">{page}</span>
          </p>
          <h1 className="mb-6 text-3xl font-extrabold tracking-tight">{page}</h1>

          {page === 'Dashboard' && <Dashboard events={events} agents={agents} perf={perf} onReset={refreshAgents} />}
          {page === 'Agents' && <Agents agents={agents} onReset={refreshAgents} />}
          {page === 'Nodes' && <Nodes config={config} events={events} />}
          {page === 'Review' && <ReviewPage reviews={reviews} onDecided={refreshReviews} />}
          {page === 'Audit log' && <AuditLog events={events} />}
          {page === 'Vault' && <Vault config={config} />}
          {page === 'Signatures' && <SignaturesPage config={config} events={events} />}
          {page === 'Policy' && <Policy config={config} />}
        </main>
      </div>
    </div>
  )
}

function Dashboard({
  events,
  agents,
  perf,
  onReset,
}: {
  events: GateEvent[]
  agents: Agent[]
  perf: MetricsSummary | null
  onReset: () => void
}) {
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
      <div className="mb-6 grid grid-cols-2 gap-5 lg:grid-cols-3">
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
        <Metric label="Gate overhead (p50)" value={perf?.stages.preprocess.count ? ms(perf.stages.preprocess.p50_ms) : '—'}>
          <span className="text-gray-500">
            p95 {perf?.stages.preprocess.count ? ms(perf.stages.preprocess.p95_ms) : '—'} · tool check p95{' '}
            {perf?.stages.tool_decision.count ? ms(perf.stages.tool_decision.p95_ms) : '—'}
          </span>
          <span className="text-gray-500">
            LLM provider p50 {perf?.stages.upstream_first_byte.count ? ms(perf.stages.upstream_first_byte.p50_ms) : '—'}
          </span>
        </Metric>
        <Metric label="Tokens used" value={tokens.toLocaleString()}>
          <span className="text-gray-500">across all agents</span>
        </Metric>
      </div>
      <Performance perf={perf} />
      <Panel title="Recent security events">
        <EventTable events={security} empty="No events yet. Point an agent at the gate to get started." />
      </Panel>
      <Agents agents={agents} onReset={onReset} />
    </>
  )
}

function ms(v: number) {
  return v < 1 ? `${(v * 1000).toFixed(0)} µs` : v < 1000 ? `${v.toFixed(v < 10 ? 1 : 0)} ms` : `${(v / 1000).toFixed(2)} s`
}

const stageInfo = [
  { key: 'preprocess', label: 'Gate: request checks', note: 'budget, allowed model, masking, before forwarding' },
  { key: 'tool_decision', label: 'Gate: tool call decision', note: 'policy, Bash guard, attack signatures' },
  { key: 'upstream_first_byte', label: 'LLM provider', note: 'time to first byte; not caused by the gate' },
  { key: 'total', label: 'End to end', note: 'whole request, including the full streamed answer' },
] as const

function Performance({ perf }: { perf: MetricsSummary | null }) {
  return (
    <Panel
      title="Performance"
      actions={<span className="text-xs text-gray-500">last 1,000 samples · full histograms in Prometheus (localhost:9090)</span>}
    >
      <Table head={['Stage', 'p50', 'p95', 'p99', 'Max', 'Samples']}>
        {stageInfo.map((s) => {
          const v = perf?.stages[s.key]
          return (
            <tr key={s.key}>
              <Td>
                <p className="font-medium">{s.label}</p>
                <p className="text-xs text-gray-500">{s.note}</p>
              </Td>
              {v?.count ? (
                <>
                  <Td className="font-mono">{ms(v.p50_ms)}</Td>
                  <Td className="font-mono">{ms(v.p95_ms)}</Td>
                  <Td className="font-mono">{ms(v.p99_ms)}</Td>
                  <Td className="font-mono">{ms(v.max_ms)}</Td>
                  <Td>{v.count}</Td>
                </>
              ) : (
                [0, 1, 2, 3, 4].map((i) => (
                  <Td key={i} className="text-gray-400">
                    —
                  </Td>
                ))
              )}
            </tr>
          )
        })}
      </Table>
    </Panel>
  )
}

function Metric({ label, value, children }: { label: string; value: number | string; children: ReactNode }) {
  return (
    <div className="rounded-2xl border border-line bg-white p-6">
      <p className="text-sm font-semibold text-sub">{label}</p>
      <p className="my-1.5 text-4xl font-extrabold tracking-tight text-ink">{value}</p>
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
          <tr key={a.id} className={a.killed ? 'bg-bad-tint/60' : ''}>
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
                className="rounded-full border border-gs-deep px-4 py-1 font-bold text-gs-deep hover:bg-gs-tint disabled:opacity-50"
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

// Each filter has the same meaning on screen and in the server-side export.
const stopped = (e: GateEvent) =>
  ['denied', 'shadow_ai', 'node_blocked', 'model_blocked'].includes(e.kind) ||
  (['tool_call', 'signature'].includes(e.kind) && (e.action === 'block' || e.action === 'kill')) ||
  (e.kind === 'policy' && e.action === 'rejected')

const filters: Record<string, { match: (e: GateEvent) => boolean; query: string }> = {
  'All events': { match: () => true, query: '' },
  'Tool calls': { match: (e) => e.kind === 'tool_call', query: 'kind=tool_call' },
  'Blocked and killed': { match: stopped, query: 'blocked=1' },
  'Attack signatures': { match: (e) => e.kind === 'signature', query: 'kind=signature' },
  'Network (shadow AI, devices)': {
    match: (e) => e.kind === 'shadow_ai' || e.kind === 'node_blocked',
    query: 'kind=shadow_ai,node_blocked',
  },
  Masking: { match: (e) => e.kind === 'mask', query: 'kind=mask' },
  Usage: { match: (e) => e.kind === 'usage', query: 'kind=usage' },
}

function AuditLog({ events }: { events: GateEvent[] }) {
  const [filter, setFilter] = useState('All events')
  const [query, setQuery] = useState('')
  const shown = events
    .filter(filters[filter].match)
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
            className="w-64 rounded-lg border border-line bg-white px-3 py-1.5"
          />
          <select value={filter} onChange={(e) => setFilter(e.target.value)} className="rounded-lg border border-line bg-white px-3 py-1.5">
            {Object.keys(filters).map((f) => (
              <option key={f}>{f}</option>
            ))}
          </select>
          {(['csv', 'jsonl'] as const).map((fmt) => (
            <a
              key={fmt}
              href={`/api/audit/export?format=${fmt}&${filters[filter].query}${query ? `&q=${encodeURIComponent(query)}` : ''}`}
              className="rounded-full border border-gs-deep px-3 py-1 font-bold whitespace-nowrap text-gs-deep hover:bg-gs-tint"
              title="Full audit trail from the gate's log file, with the current filter"
            >
              Export {fmt === 'csv' ? 'CSV' : 'JSON'}
            </a>
          ))}
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
        <tr key={e.id} className={e.action === 'kill' ? 'bg-bad-tint/60' : ''}>
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
  network: { tone: 'warn', label: 'Network', text: 'Allowed without vaulted data; with it, the Controls above decide.' },
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
            <tr key={e.agent} className="bg-bad-tint/60">
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
      <div className="mb-6 rounded-2xl border border-warn/30 bg-white px-6 py-4 text-sub">
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

      <div className="rounded-xl border border-warn/25 bg-warn-tint p-3">
        <p className="mb-1 text-xs font-bold tracking-wide text-warn uppercase">✦ AI suggestion</p>
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
                        className={`h-2 rounded ${o === r.suggestion?.value ? 'bg-warn' : 'bg-warn/30'}`}
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
          className="rounded-lg border border-line bg-white px-3 py-1.5"
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
          className="rounded-full bg-gs px-4 py-1 font-bold text-white hover:bg-gs-deep disabled:opacity-50"
        >
          Apply
        </button>
        <button
          disabled={busy}
          onClick={() => decide('dismiss')}
          className="rounded-full border border-gs-deep px-4 py-1 font-bold text-gs-deep hover:bg-gs-tint disabled:opacity-50"
        >
          Keep current rule
        </button>
        {err && <p className="text-xs text-bad">{err}</p>}
      </div>
    </div>
  )
}

const severityTone = { low: 'muted', medium: 'warn', high: 'bad', critical: 'bad' } as const

function SignaturesPage({ config, events }: { config: Config | null; events: GateEvent[] }) {
  const [data, setData] = useState<Signatures | null>(null)
  useEffect(() => {
    let alive = true
    const load = () => api.signatures().then((d) => alive && setData(d)).catch(() => {})
    load()
    const t = setInterval(load, 3000)
    return () => {
      alive = false
      clearInterval(t)
    }
  }, [])
  const hits = useMemo(() => {
    const m = new Map<string, number>()
    for (const e of events) {
      if (e.kind !== 'signature') continue
      const id = e.reason?.split(' ')[0] ?? ''
      m.set(id, (m.get(id) ?? 0) + 1)
    }
    return m
  }, [events])
  if (!data) return null
  const sigs = data.signatures ?? []
  return (
    <>
      <Panel title={`Feeds (${data.feeds?.length ?? 0})`}>
        <p className="px-5 py-3 text-gray-600">
          Signatures of known attacks, loaded from files and URLs listed under <Mono>signatures.feeds</Mono> in the
          policy file. URL feeds are fetched on a schedule so one feed can protect every gate. A feed that fails to load
          keeps its last good copy.
        </p>
        <Table head={['Source', 'Feed', 'Version', 'Signatures', 'Status']}>
          {(data.feeds ?? []).map((f) => (
            <tr key={f.source}>
              <Td className="max-w-md break-all">
                <Mono>{f.source}</Mono>
              </Td>
              <Td>{f.name || '—'}</Td>
              <Td>{f.version || '—'}</Td>
              <Td>{f.count}</Td>
              <Td>{f.error ? <Status tone={f.count ? 'warn' : 'bad'}>{f.error}</Status> : <Status tone="ok">Loaded</Status>}</Td>
            </tr>
          ))}
        </Table>
      </Panel>

      <Panel title={`Signatures (${sigs.length})`}>
        <Table head={['ID', 'Signature', 'Category', 'Severity', 'Action', 'Applies to', 'Hits']}>
          {sigs.map((s) => (
            <tr key={s.id} className={s.enabled ? '' : 'opacity-50'}>
              <Td className="whitespace-nowrap">
                <Mono>{s.id}</Mono>
              </Td>
              <Td>
                <p className="font-medium">
                  {s.name}
                  {!s.enabled && <span className="ml-2 text-xs text-gray-500">(disabled in policy)</span>}
                </p>
                <p className="text-xs text-gray-600">{s.description}</p>
                {s.reference && <p className="text-xs text-gray-500">{s.reference}</p>}
              </Td>
              <Td className="whitespace-nowrap text-gray-700">{s.category.replaceAll('_', ' ')}</Td>
              <Td>
                <Status tone={severityTone[s.severity]}>{s.severity}</Status>
              </Td>
              <Td>
                <Status tone={s.action === 'alert' ? 'warn' : 'bad'}>{s.action}</Status>
              </Td>
              <Td className="text-xs">{s.tools.includes('*') ? 'all tools' : s.tools.join(', ')}</Td>
              <Td className="font-bold">{hits.get(s.id) ?? 0}</Td>
            </tr>
          ))}
        </Table>
      </Panel>

      {config?.bash_guard && (
        <Panel title="Bash guard">
          <p className="px-5 py-3 text-gray-600">
            Shell tools ({config.bash_guard.tools.map((t) => <Mono key={t}>{t}</Mono>)}) are local, but a command that runs
            one of these programs sends data off the machine. Such a call is treated as a network tool: vaulted values are
            not restored into it, and a secret in it fires the kill switch. Quoting tricks (<Mono>{"c''url"}</Mono>),{' '}
            <Mono>sudo</Mono>, <Mono>xargs</Mono> and <Mono>bash -c</Mono> are seen through.
          </p>
          <div className="flex flex-wrap gap-1 px-5 pb-4">
            {config.bash_guard.network_commands.map((c) => (
              <Mono key={c}>{c}</Mono>
            ))}
          </div>
        </Panel>
      )}
    </>
  )
}

const actionInfo = {
  allow: { tone: 'ok', label: 'Allow' },
  block: { tone: 'warn', label: 'Block the call' },
  kill: { tone: 'bad', label: 'Kill switch' },
} as const

function Policy({ config }: { config: Config | null }) {
  if (!config) return null
  const tools = Object.entries(config.policy.tools).sort(([a], [b]) => a.localeCompare(b))
  const def = sinkInfo[config.policy.default as keyof typeof sinkInfo]
  const budgets = Object.entries(config.budgets ?? {}).sort(([a], [b]) => a.localeCompare(b))
  const pf = config.policy_file
  return (
    <>
      <Panel title="Policy file">
        <p className="px-5 py-3 text-gray-600">
          Every setting on this page comes from <Mono>{pf.path}</Mono> (<Mono>config/hushgate.yaml</Mono> in the
          repository). Edit and save it: the gate applies the change within a second and lists it in the audit log. An
          invalid edit is rejected and the previous version stays active.
        </p>
        <Table head={['Version', 'Loaded at', 'Status']}>
          <tr>
            <Td>
              <Mono>{pf.version}</Mono>
            </Td>
            <Td className="font-mono text-xs text-gray-600">{time(pf.loaded_at)}</Td>
            <Td>{pf.error ? <Status tone="bad">Last edit rejected</Status> : <Status tone="ok">Active</Status>}</Td>
          </tr>
        </Table>
      </Panel>

      <Panel title="Controls">
        <Table head={['Setting', 'Value']}>
          <tr>
            <Td>Masked before reaching the LLM</Td>
            <Td>
              Tier <b>{config.mask_from}</b> and above
            </Td>
          </tr>
          {['C3', 'C2', 'C1'].map((t) => {
            const a = config.policy.on_network_tool[t] ?? 'allow'
            return (
              <tr key={t}>
                <Td>
                  <TierBadge tier={t} /> data in a network tool call
                </Td>
                <Td>
                  <Status tone={actionInfo[a].tone}>{actionInfo[a].label}</Status>
                </Td>
              </tr>
            )
          })}
          <tr>
            <Td>Allowed models</Td>
            <Td>{config.models?.length ? config.models.map((m) => <Mono key={m}>{m}</Mono>) : 'Any model'}</Td>
          </tr>
          <tr>
            <Td>Approved LLM providers (company network)</Td>
            <Td>{(config.llm_hosts ?? []).map((h) => <Mono key={h}>{h}</Mono>)}</Td>
          </tr>
          <tr>
            <Td>Prompt injection warning above</Td>
            <Td>{(config.injection_threshold * 100).toFixed(0)}% (AI advisor score)</Td>
          </tr>
          <tr>
            <Td>Token budget per agent</Td>
            <Td>{config.token_limit ? config.token_limit.toLocaleString() : 'Unlimited'}</Td>
          </tr>
          {budgets.map(([agent, n]) => (
            <tr key={agent}>
              <Td className="pl-10">
                Budget for <Mono>{agent}</Mono>
              </Td>
              <Td>{n ? n.toLocaleString() : 'Unlimited'}</Td>
            </tr>
          ))}
        </Table>
      </Panel>

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
    </>
  )
}

const navIcons: Record<Page, ReactNode> = {
  Dashboard: <><rect x="4" y="4" width="7" height="7" rx="1.5" /><rect x="13" y="4" width="7" height="7" rx="1.5" /><rect x="4" y="13" width="7" height="7" rx="1.5" /><rect x="13" y="13" width="7" height="7" rx="1.5" /></>,
  Agents: <><rect x="4" y="5" width="16" height="12" rx="2" /><path d="m8 10 2 1.5L8 13M12.5 13H16M8 20h8" /></>,
  Nodes: <><rect x="5" y="5" width="14" height="10" rx="1.5" /><path d="M3 19h18" /></>,
  Review: <path d="M12 3l1.8 4.6L18.5 9l-4.7 1.4L12 15l-1.8-4.6L5.5 9l4.7-1.4zM18 15l.9 2.1L21 18l-2.1.9L18 21l-.9-2.1L15 18l2.1-.9z" />,
  'Audit log': <><path d="M9 6h11M9 12h11M9 18h11" /><circle cx="4.5" cy="6" r="1" /><circle cx="4.5" cy="12" r="1" /><circle cx="4.5" cy="18" r="1" /></>,
  Vault: <><rect x="5" y="11" width="14" height="10" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></>,
  Signatures: <path d="M12 3 5 6v5c0 4.5 3 8.3 7 10 4-1.7 7-5.5 7-10V6z" />,
  Policy: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v4h4M9 12h6M9 16h6" /></>,
}

function NavIcon({ page }: { page: Page }) {
  return (
    <svg viewBox="0 0 24 24" className="h-5 w-5 shrink-0" fill="none" stroke="currentColor" strokeWidth={1.8} strokeLinecap="round" strokeLinejoin="round" aria-hidden>
      {navIcons[page]}
    </svg>
  )
}

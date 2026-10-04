import type { ReactNode } from 'react'
import type { GateEvent } from './api'

export function Panel({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <section className="mb-6 overflow-hidden rounded-2xl border border-line bg-white">
      <header className="flex items-center justify-between border-b border-line px-6 py-4">
        <h2 className="text-lg font-bold tracking-tight">{title}</h2>
        {actions}
      </header>
      <div>{children}</div>
    </section>
  )
}

export function Table({ head, children, empty }: { head: string[]; children: ReactNode; empty?: string }) {
  const rows = Array.isArray(children) ? children.length : children ? 1 : 0
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-left">
        <thead>
          <tr className="border-b border-line bg-page/60 text-xs font-bold text-sub">
            {head.map((h) => (
              <th key={h} className="px-6 py-3 whitespace-nowrap">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
      {rows === 0 && <p className="px-6 py-8 text-center text-sub">{empty ?? 'No data'}</p>}
    </div>
  )
}

export function Td({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <td className={`border-b border-line/70 px-6 py-3 align-top ${className}`}>{children}</td>
}

type Tone = 'ok' | 'bad' | 'warn' | 'info' | 'muted'
const toneClass: Record<Tone, string> = {
  ok: 'text-ok',
  bad: 'text-bad',
  warn: 'text-warn',
  info: 'text-link',
  muted: 'text-sub',
}
const toneIcon: Record<Tone, string> = { ok: '✓', bad: '⊗', warn: '⚠', info: 'ⓘ', muted: '○' }

export function Status({ tone, children }: { tone: Tone; children: ReactNode }) {
  return (
    <span className={`inline-flex items-center gap-1.5 font-medium whitespace-nowrap ${toneClass[tone]}`}>
      <span aria-hidden>{toneIcon[tone]}</span>
      {children}
    </span>
  )
}

const tierStyle: Record<string, string> = {
  C0: 'bg-page text-sub',
  C1: 'bg-[#eef1f5] text-[#4b5a6c]',
  C2: 'bg-gs-tint text-gs-deep',
  C3: 'bg-bad-tint text-bad',
}
export const tierLabel: Record<string, string> = {
  C0: 'Public',
  C1: 'Internal',
  C2: 'Confidential',
  C3: 'Secret',
}

export function TierBadge({ tier }: { tier: string }) {
  return (
    <span className={`rounded-md px-2 py-0.5 text-xs font-bold ${tierStyle[tier] ?? ''}`}>
      {tier} · {tierLabel[tier] ?? tier}
    </span>
  )
}

export function Meter({ used, limit }: { used: number; limit: number }) {
  if (!limit) return <span>{used.toLocaleString()} tokens</span>
  const pct = Math.min(100, (used / limit) * 100)
  const color = pct >= 100 ? 'bg-bad' : pct >= 80 ? 'bg-gs-deep' : 'bg-gs'
  return (
    <div className="min-w-48">
      <div className="mb-1 flex justify-between text-xs text-sub">
        <span>{used.toLocaleString()} / {limit.toLocaleString()}</span>
        <span>{pct.toFixed(0)}%</span>
      </div>
      <div className="h-2 rounded-full bg-line">
        <div className={`h-2 rounded-full ${color}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

export function Mono({ children }: { children: ReactNode }) {
  return <code className="rounded-md bg-page px-1.5 py-0.5 font-mono text-xs">{children}</code>
}

export function time(iso: string) {
  const d = new Date(iso)
  return d.toLocaleTimeString([], { hour12: false }) + '.' + String(d.getMilliseconds()).padStart(3, '0')
}

export function EventStatus({ e }: { e: GateEvent }) {
  switch (e.kind) {
    case 'tool_call':
      if (e.action === 'kill') return <Status tone="bad">Killed</Status>
      if (e.action === 'block') return <Status tone="warn">Blocked</Status>
      return <Status tone="ok">Allowed</Status>
    case 'mask':
      return <Status tone="info">Masked</Status>
    case 'denied':
      return <Status tone="bad">Denied</Status>
    case 'reset':
      return <Status tone="info">Reset</Status>
    case 'shadow_ai':
      return <Status tone="bad">Shadow AI</Status>
    case 'node_blocked':
      return <Status tone="bad">Unknown device</Status>
    case 'suggestion':
      return <Status tone="info">AI suggestion</Status>
    case 'injection':
      return <Status tone="warn">Prompt injection</Status>
    case 'policy':
      return e.action === 'rejected' ? <Status tone="bad">Policy rejected</Status> : <Status tone="info">Policy reloaded</Status>
    case 'model_blocked':
      return <Status tone="bad">Model blocked</Status>
    case 'signature':
      if (e.action === 'kill') return <Status tone="bad">Signature: kill</Status>
      if (e.action === 'block') return <Status tone="bad">Signature: block</Status>
      return <Status tone="warn">Signature: alert</Status>
    case 'review':
      return <Status tone="ok">Reviewed</Status>
    default:
      return <Status tone="muted">Usage</Status>
  }
}

export function eventMessage(e: GateEvent): ReactNode {
  const tokens = e.tokens?.length ? (
    <span className="ml-1 inline-flex flex-wrap gap-1">
      {e.tokens.map((t) => (
        <Mono key={t}>{t}</Mono>
      ))}
    </span>
  ) : null
  switch (e.kind) {
    case 'tool_call':
      return (
        <>
          <Mono>{e.tool}</Mono> — {e.reason}
          {tokens}
        </>
      )
    case 'mask':
      return (
        <>
          Replaced {e.tokens?.length ?? 0} secret(s) in outbound request {tokens}
        </>
      )
    case 'usage':
      return <>Total {e.usage?.toLocaleString()} tokens</>
    case 'denied':
      return <>Request refused — {e.reason}</>
    case 'suggestion':
      return (
        <>
          <Mono>{e.tool}</Mono> → <b>{e.action}</b>: {e.reason}
        </>
      )
    case 'review':
      return (
        <>
          <Mono>{e.tool}</Mono> {e.action === 'dismissed' ? 'dismissed' : <>set to <b>{e.action}</b></>} — {e.reason}
        </>
      )
    case 'injection':
      return (
        <>
          In <Mono>{e.tool}</Mono>: {e.reason}
        </>
      )
    case 'signature':
      return (
        <>
          <Mono>{e.tool}</Mono> call matched {e.reason}
        </>
      )
    case 'policy':
      return e.action === 'rejected' ? (
        <>
          Edit to <Mono>hushgate.yaml</Mono> rejected, previous policy still active: {e.reason}
        </>
      ) : (
        <>
          <Mono>hushgate.yaml</Mono> applied:{' '}
          {e.reason?.split('; ').map((c) => (
            <Mono key={c}>{c}</Mono>
          ))}
        </>
      )
    case 'shadow_ai':
    case 'node_blocked':
      return (
        <>
          <Mono>{e.host}</Mono> — {e.reason}
        </>
      )
    default:
      return <>{e.reason}</>
  }
}

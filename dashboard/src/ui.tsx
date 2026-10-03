import type { ReactNode } from 'react'
import type { GateEvent } from './api'

export function Panel({ title, actions, children }: { title: string; actions?: ReactNode; children: ReactNode }) {
  return (
    <section className="mb-5 rounded-lg bg-white shadow-[0_1px_1px_rgba(0,28,36,.3),1px_1px_1px_rgba(0,28,36,.15)]">
      <header className="flex items-center justify-between border-b border-gray-200 px-5 py-3">
        <h2 className="text-lg font-bold">{title}</h2>
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
          <tr className="border-b border-gray-200 bg-gray-50 text-xs font-bold text-gray-600">
            {head.map((h) => (
              <th key={h} className="px-5 py-2.5 whitespace-nowrap">{h}</th>
            ))}
          </tr>
        </thead>
        <tbody>{children}</tbody>
      </table>
      {rows === 0 && <p className="px-5 py-8 text-center text-gray-500">{empty ?? 'No data'}</p>}
    </div>
  )
}

export function Td({ children, className = '' }: { children: ReactNode; className?: string }) {
  return <td className={`border-b border-gray-100 px-5 py-2.5 align-top ${className}`}>{children}</td>
}

type Tone = 'ok' | 'bad' | 'warn' | 'info' | 'muted'
const toneClass: Record<Tone, string> = {
  ok: 'text-ok',
  bad: 'text-bad',
  warn: 'text-[#8d6605]',
  info: 'text-link',
  muted: 'text-gray-500',
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
  C0: 'bg-gray-100 text-gray-700',
  C1: 'bg-blue-50 text-blue-800',
  C2: 'bg-amber-50 text-amber-800',
  C3: 'bg-red-50 text-red-800',
}
export const tierLabel: Record<string, string> = {
  C0: 'Public',
  C1: 'Internal',
  C2: 'Confidential',
  C3: 'Secret',
}

export function TierBadge({ tier }: { tier: string }) {
  return (
    <span className={`rounded px-2 py-0.5 text-xs font-bold ${tierStyle[tier] ?? ''}`}>
      {tier} · {tierLabel[tier] ?? tier}
    </span>
  )
}

export function Meter({ used, limit }: { used: number; limit: number }) {
  if (!limit) return <span>{used.toLocaleString()} tokens</span>
  const pct = Math.min(100, (used / limit) * 100)
  const color = pct >= 100 ? 'bg-bad' : pct >= 80 ? 'bg-warn' : 'bg-link'
  return (
    <div className="min-w-48">
      <div className="mb-1 flex justify-between text-xs text-gray-600">
        <span>{used.toLocaleString()} / {limit.toLocaleString()}</span>
        <span>{pct.toFixed(0)}%</span>
      </div>
      <div className="h-1.5 rounded bg-gray-200">
        <div className={`h-1.5 rounded ${color}`} style={{ width: `${pct}%` }} />
      </div>
    </div>
  )
}

export function Mono({ children }: { children: ReactNode }) {
  return <code className="rounded bg-gray-100 px-1 py-0.5 font-mono text-xs">{children}</code>
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

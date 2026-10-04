import React from 'react';
import {AbsoluteFill, Easing, interpolate, spring, useCurrentFrame} from 'remotion';
import type {SpringConfig} from 'remotion';
import {C, MONO, SANS, SHADOW} from './theme';

export const FPS = 30;
const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;
export const easeOut = Easing.bezier(0.16, 1, 0.3, 1);
export const easeInOut = Easing.bezier(0.65, 0, 0.35, 1);

export const tw = (f: number, a: number, b: number, from = 0, to = 1, easing = easeOut) =>
  interpolate(f, [a, b], [from, to], {...clamp, easing});
export const sp = (f: number, at: number, config: Partial<SpringConfig> = {}) =>
  spring({frame: f - at, fps: FPS, config: {damping: 15, stiffness: 170, mass: 0.7, ...config}});
// 1 at frame `at`, fading to 0 over `len` frames: a flash.
export const bump = (f: number, at: number, len = 10) => (f < at ? 0 : Math.max(0, 1 - (f - at) / len));

type Kids = {children?: React.ReactNode};

export const Bg: React.FC<Kids & {color: string}> = ({color, children}) => (
  <AbsoluteFill style={{background: color, fontFamily: SANS, color: C.ink, overflow: 'hidden'}}>{children}</AbsoluteFill>
);

// Big kinetic headline: each word rises out of a mask.
export const Headline: React.FC<{text: string; at: number; out?: number; top: number; size?: number; color?: string; accent?: string; accentWords?: number[]}> = ({
  text, at, out, top, size = 96, color = C.ink, accent = C.orange, accentWords = [],
}) => {
  const f = useCurrentFrame();
  const o = out === undefined ? 1 : 1 - tw(f, out, out + 10);
  return (
    <div style={{position: 'absolute', left: 60, right: 60, top, textAlign: 'center', fontSize: size, fontWeight: 800, letterSpacing: '-0.03em', lineHeight: 1.12, color, opacity: o}}>
      {text.split(' ').map((w, i) => {
        const p = sp(f, at + i * 3, {damping: 20, stiffness: 210});
        return (
          <span key={i} style={{display: 'inline-block', overflow: 'hidden', verticalAlign: 'top', padding: '0 0.13em 0.14em'}}>
            <span style={{display: 'inline-block', transform: `translateY(${(1 - p) * 110}%)`, color: accentWords.includes(i) ? accent : undefined}}>{w}</span>
          </span>
        );
      })}
    </div>
  );
};

export const Pop: React.FC<Kids & {at: number; from?: number; y?: number; style?: React.CSSProperties; cfg?: Partial<SpringConfig>}> = ({
  at, from = 0.7, y = 30, style, cfg, children,
}) => {
  const p = sp(useCurrentFrame(), at, cfg);
  return <div style={{...style, opacity: Math.min(1, Math.max(0, p) * 1.6), transform: `translateY(${(1 - p) * y}px) scale(${from + (1 - from) * p})`}}>{children}</div>;
};

// A UI panel that swings in from a 3D tilt and settles flat.
export const Tilt: React.FC<Kids & {at: number; rx?: number; ry?: number; rz?: number; s?: number; style?: React.CSSProperties}> = ({
  at, rx = 24, ry = -18, rz = 3, s = 0.84, style, children,
}) => {
  const f = useCurrentFrame();
  const p = sp(f, at, {damping: 22, stiffness: 60, mass: 1});
  return (
    <div style={{...style, opacity: tw(f, at, at + 10), transform: `perspective(2600px) rotateX(${rx * (1 - p)}deg) rotateY(${ry * (1 - p)}deg) rotateZ(${rz * (1 - p)}deg) scale(${s + (1 - s) * p})`}}>
      {children}
    </div>
  );
};

// Camera moves: keyframes of [frame, scale, x, y].
export const Camera: React.FC<Kids & {keys: [number, number, number, number][]; origin?: string}> = ({keys, origin = '50% 50%', children}) => {
  const f = useCurrentFrame();
  const v = (i: 1 | 2 | 3) => (keys.length < 2 ? keys[0][i] : interpolate(f, keys.map((k) => k[0]), keys.map((k) => k[i]), {...clamp, easing: easeInOut}));
  return <AbsoluteFill style={{transformOrigin: origin, transform: `translate(${v(2)}px, ${v(3)}px) scale(${v(1)})`}}>{children}</AbsoluteFill>;
};

/* ---------- Dashboard pieces, scaled up for 1080p ---------- */

export const ShieldIcon: React.FC<{size?: number; color?: string}> = ({size = 40, color = C.orange}) => (
  <svg viewBox="0 0 24 24" width={size} height={size} style={{display: 'block'}}>
    <path fill={color} d="M12 2 4 5v6c0 5 3.4 9.4 8 11 4.6-1.6 8-6 8-11V5l-8-3z" />
  </svg>
);

export const Card: React.FC<Kids & {style?: React.CSSProperties}> = ({style, children}) => (
  <div style={{background: '#fff', borderRadius: 14, boxShadow: SHADOW, ...style}}>{children}</div>
);

export const Metric: React.FC<Kids & {label: string; value: React.ReactNode; style?: React.CSSProperties; valueSize?: number}> = ({label, value, style, valueSize = 76, children}) => (
  <Card style={{padding: '26px 32px', ...style}}>
    <div style={{fontSize: 24, color: C.sub}}>{label}</div>
    <div style={{fontSize: valueSize, fontWeight: 300, color: C.link, lineHeight: 1.15, margin: '4px 0 6px'}}>{value}</div>
    <div style={{display: 'flex', flexDirection: 'column', gap: 4, fontSize: 22}}>{children}</div>
  </Card>
);

export type Tone = 'ok' | 'bad' | 'warn' | 'info';
const TONE: Record<Tone, [string, string]> = {ok: [C.ok, '✓'], bad: [C.bad, '⊗'], warn: [C.warnInk, '⚠'], info: [C.link, 'ⓘ']};
export const Status: React.FC<Kids & {tone: Tone; size?: number; style?: React.CSSProperties}> = ({tone, size, style, children}) => (
  <span style={{display: 'inline-flex', alignItems: 'center', gap: '0.35em', fontWeight: 600, color: TONE[tone][0], fontSize: size, whiteSpace: 'nowrap', ...style}}>
    <span>{TONE[tone][1]}</span>{children}
  </span>
);

export const Panel: React.FC<Kids & {title: string; actions?: React.ReactNode; style?: React.CSSProperties}> = ({title, actions, style, children}) => (
  <Card style={{overflow: 'hidden', ...style}}>
    <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', height: 76, padding: '0 32px', borderBottom: `1px solid ${C.line}`}}>
      <div style={{fontSize: 30, fontWeight: 700}}>{title}</div>
      {actions}
    </div>
    {children}
  </Card>
);

export const Row: React.FC<Kids & {cols: string; head?: boolean; style?: React.CSSProperties}> = ({cols, head, style, children}) => (
  <div style={{display: 'grid', gridTemplateColumns: cols, alignItems: 'center', gap: 24, padding: '0 32px', height: head ? 54 : 78, fontSize: head ? 20 : 24, fontWeight: head ? 700 : 400, color: head ? C.sub : C.ink, background: head ? '#fafafa' : undefined, borderBottom: `1px solid ${C.line}`, ...style}}>
    {children}
  </div>
);

export const Code: React.FC<Kids & {size?: number}> = ({size = 21, children}) => (
  <span style={{fontFamily: MONO, fontSize: size, background: '#f2f3f3', borderRadius: 6, padding: '3px 10px', whiteSpace: 'nowrap'}}>{children}</span>
);

const TIER: Record<string, [string, string, string]> = {
  C1: ['#eff6ff', '#1e40af', 'Internal'],
  C2: ['#fffbeb', '#92400e', 'Confidential'],
  C3: ['#fef2f2', '#991b1b', 'Secret'],
};
export const TierBadge: React.FC<{tier: string; size?: number}> = ({tier, size = 24}) => (
  <span style={{background: TIER[tier][0], color: TIER[tier][1], fontWeight: 700, fontSize: size, padding: '5px 14px', borderRadius: 8, whiteSpace: 'nowrap'}}>
    {tier} · {TIER[tier][2]}
  </span>
);

export const Meter: React.FC<{used: number; limit: number}> = ({used, limit}) => {
  const pct = Math.min(100, (used / limit) * 100);
  const color = pct >= 100 ? C.bad : pct >= 80 ? C.orange : C.link;
  return (
    <div>
      <div style={{display: 'flex', justifyContent: 'space-between', fontSize: 20, color: C.sub, marginBottom: 8, fontVariantNumeric: 'tabular-nums'}}>
        <span>{Math.round(used).toLocaleString('en-US')} / {limit.toLocaleString('en-US')}</span>
        <span>{pct.toFixed(0)}%</span>
      </div>
      <div style={{height: 12, borderRadius: 6, background: C.line}}>
        <div style={{height: 12, borderRadius: 6, background: color, width: `${pct}%`}} />
      </div>
    </div>
  );
};

export type ChipKind = 'secret' | 'masked' | 'plain';
const CHIP: Record<ChipKind, [string, string, string]> = {
  secret: [C.bad, '#ffffff', C.bad],
  masked: [C.warnBg, C.warnInk, C.orange],
  plain: ['#10263a', '#a9d4f5', C.link],
};
// A piece of data in flight. Absolutely positioned by its center unless `inline`.
export const Chip: React.FC<Kids & {kind: ChipKind; x?: number; y?: number; inline?: boolean; style?: React.CSSProperties}> = ({kind, x = 0, y = 0, inline, style, children}) => (
  <div
    style={{
      ...(inline ? {display: 'inline-block'} : {position: 'absolute', left: x, top: y, transform: 'translate(-50%, -50%)'}),
      whiteSpace: 'nowrap', fontFamily: MONO, fontSize: 22, fontWeight: 600, padding: '7px 16px', borderRadius: 999,
      background: CHIP[kind][0], color: CHIP[kind][1], border: `2px solid ${CHIP[kind][2]}`, ...style,
    }}
  >
    {children}
  </div>
);

/* ---------- Network-shot icons ---------- */

export const AgentNode: React.FC<{x: number; y: number; size?: number; ring?: string; style?: React.CSSProperties}> = ({x, y, size = 92, ring = C.nightLine, style}) => (
  <div style={{position: 'absolute', left: x - size / 2, top: y - size / 2, width: size, height: size, borderRadius: size * 0.22, background: C.nightNode, border: `2px solid ${ring}`, display: 'flex', alignItems: 'center', justifyContent: 'center', fontFamily: MONO, fontSize: size * 0.32, fontWeight: 600, color: '#c8d3e0', ...style}}>
    &gt;_
  </div>
);

export const Cloud: React.FC<{x: number; y: number; w: number; h: number; stroke?: string; glow?: number}> = ({x, y, w, h, stroke = C.nightLine, glow = 0}) => (
  <svg viewBox="0 0 340 170" width={w} height={h} style={{position: 'absolute', left: x, top: y, overflow: 'visible', filter: glow ? `drop-shadow(0 0 ${8 + glow * 30}px ${stroke})` : undefined}}>
    <path d="M78 150 A42 42 0 0 1 66 68 A58 58 0 0 1 160 36 A66 66 0 0 1 270 62 A44 44 0 0 1 282 150 Z" fill={C.nightNode} stroke={stroke} strokeWidth={3} strokeLinejoin="round" />
  </svg>
);

export const Laptop: React.FC<{x: number; y: number; ring?: string}> = ({x, y, ring = C.nightLine}) => (
  <svg viewBox="0 0 150 104" width={150} height={104} style={{position: 'absolute', left: x - 75, top: y - 52, overflow: 'visible'}}>
    <rect x={18} y={4} width={114} height={76} rx={8} fill={C.nightNode} stroke={ring} strokeWidth={3} />
    <path d="M4 88 H146 L138 100 H12 Z" fill={C.nightNode} stroke={ring} strokeWidth={3} strokeLinejoin="round" />
  </svg>
);

export const Server: React.FC<{x: number; y: number; ring?: string}> = ({x, y, ring = C.nightLine}) => (
  <svg viewBox="0 0 160 150" width={160} height={150} style={{position: 'absolute', left: x - 80, top: y - 75, overflow: 'visible'}}>
    {[0, 50, 100].map((t) => (
      <g key={t}>
        <rect x={6} y={t + 4} width={148} height={42} rx={7} fill={C.nightNode} stroke={ring} strokeWidth={3} />
        <circle cx={28} cy={t + 25} r={5} fill={ring} />
      </g>
    ))}
  </svg>
);

export const Gate: React.FC<{x: number; y: number; w: number; h: number; at: number; flash?: number}> = ({x, y, w, h, at, flash = 0}) => {
  const p = sp(useCurrentFrame(), at, {damping: 14, stiffness: 120});
  return (
    <div style={{position: 'absolute', left: x, top: y, width: w, height: h, borderRadius: 22, background: C.nav, border: `3px solid ${C.orange}`, boxShadow: `0 0 ${30 + flash * 60}px rgba(255,153,0,${0.25 + flash * 0.45})`, transform: `scaleY(${p})`, opacity: Math.min(1, p * 2), display: 'flex', alignItems: 'center', justifyContent: 'center'}}>
      <ShieldIcon size={Math.min(96, w * 0.62)} />
    </div>
  );
};

export const Cursor: React.FC<{x: number; y: number; press?: number}> = ({x, y, press = 0}) => (
  <svg viewBox="0 0 24 24" width={48} height={48} style={{position: 'absolute', left: x - 6, top: y - 3, transform: `scale(${1 - press * 0.15})`, filter: 'drop-shadow(0 3px 6px rgba(0,0,0,.35))'}}>
    <path d="M4 2 L4 19 L8.5 15 L11.5 22 L14.5 20.7 L11.6 14 L18 14 Z" fill="#ffffff" stroke="#16191f" strokeWidth={1.4} strokeLinejoin="round" />
  </svg>
);

export const Wordmark: React.FC<{at: number; scale?: number; sub?: boolean}> = ({at, scale = 1, sub = true}) => {
  const f = useCurrentFrame();
  const s = sp(f, at, {damping: 10, stiffness: 150, mass: 0.8});
  const slide = tw(f, at + 22, at + 40, 1, 0, easeInOut);
  const textP = tw(f, at + 26, at + 44);
  return (
    <div style={{position: 'relative', display: 'flex', alignItems: 'center', gap: 46 * scale}}>
      {[0, 10].map((d) => {
        const r = tw(f, at + 8 + d, at + 40 + d, 0, 1, Easing.out(Easing.quad));
        return <div key={d} style={{position: 'absolute', left: 100 * scale + slide * 395 * scale - 90 * scale, top: '50%', width: 180 * scale, height: 180 * scale, marginTop: -90 * scale, borderRadius: '50%', border: `${3 * scale}px solid ${C.orange}`, opacity: (1 - r) * 0.7 * (f >= at + 8 + d ? 1 : 0), transform: `scale(${0.6 + r * 2.4})`}} />;
      })}
      <div style={{transform: `translateX(${slide * 395 * scale}px) scale(${s})`}}><ShieldIcon size={200 * scale} /></div>
      <div style={{opacity: textP, transform: `translateX(${(1 - textP) * -40}px)`}}>
        <div style={{fontSize: 150 * scale, fontWeight: 800, letterSpacing: '-0.035em', color: '#ffffff', lineHeight: 1}}>HushGate</div>
        {sub && <div style={{fontSize: 46 * scale, color: '#9ca3af', marginTop: 10 * scale, opacity: tw(f, at + 40, at + 54)}}>AI Control Layer</div>}
      </div>
    </div>
  );
};

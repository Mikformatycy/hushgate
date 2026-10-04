import React from 'react';
import {Easing, interpolate, interpolateColors, spring} from 'remotion';
import type {SpringConfig} from 'remotion';
import {C, MONO} from './theme';

export const FPS = 30;
export const easeInOut = Easing.bezier(0.65, 0, 0.35, 1);
export const easeOut = Easing.bezier(0.16, 1, 0.3, 1);
const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

export const tw = (f: number, a: number, b: number, from = 0, to = 1, easing = easeOut) =>
  interpolate(f, [a, b], [from, to], {...clamp, easing});
export const sp = (f: number, at: number, config: Partial<SpringConfig> = {}) =>
  spring({frame: f - at, fps: FPS, config: {damping: 16, stiffness: 170, mass: 0.7, ...config}});
export const bump = (f: number, at: number, len = 10) => (f < at ? 0 : Math.max(0, 1 - (f - at) / len));

// Keyframes: [frame, value] pairs, eased between neighbours.
export const kf = (f: number, keys: [number, number][], easing = easeInOut) => {
  if (f <= keys[0][0]) return keys[0][1];
  for (let i = 0; i < keys.length - 1; i++) {
    const [f0, v0] = keys[i], [f1, v1] = keys[i + 1];
    if (f <= f1) return v0 + (v1 - v0) * easing((f - f0) / (f1 - f0));
  }
  return keys[keys.length - 1][1];
};
export const kc = (f: number, keys: [number, string][]) =>
  keys.length < 2 ? keys[0][1] : interpolateColors(f, keys.map((k) => k[0]), keys.map((k) => k[1]));

export type Rect = {x: number; y: number; w: number; h: number; r: number};
// Morph a rectangle between keyframed shapes.
export const kr = (f: number, keys: [number, Rect][]): Rect => {
  const g = (p: keyof Rect) => kf(f, keys.map(([t, r]) => [t, r[p]] as [number, number]));
  return {x: g('x'), y: g('y'), w: g('w'), h: g('h'), r: g('r')};
};
export const box = (r: Rect): React.CSSProperties => ({position: 'absolute', left: r.x, top: r.y, width: r.w, height: r.h, borderRadius: r.r});

// Opacity window: fades in over [a, a+fi], out over [b-fo, b].
export const win = (f: number, a: number, b: number, fi = 10, fo = 10) => Math.min(tw(f, a, a + fi), 1 - tw(f, b - fo, b));

/* ---------- Morphing caption: one line that rewrites itself ---------- */

export type Cue = [number, number, string, string?];
export const Caption: React.FC<{f: number; cues: Cue[]; color: string; accent?: string; top?: number; size?: number}> = ({f, cues, color, accent = C.gs, top = 84, size = 64}) => (
  <>
    {cues.map(([a, b, text, acc]) => {
      if (f < a || f > b) return null;
      const i = tw(f, a, a + 12), o = tw(f, b - 8, b, 0, 1, Easing.in(Easing.quad));
      return (
        <div key={a} style={{position: 'absolute', left: 0, right: 0, top, textAlign: 'center', fontSize: size, fontWeight: 800, letterSpacing: '-0.03em', color, opacity: i * (1 - o), filter: `blur(${(1 - i) * 10 + o * 10}px)`, transform: `translateY(${(1 - i) * 26 - o * 22}px)`}}>
          {text.split(/(\*[^*]+\*)/).map((part, k) =>
            part.startsWith('*') ? <span key={k} style={{color: acc ?? accent}}>{part.slice(1, -1)}</span> : <span key={k}>{part}</span>,
          )}
        </div>
      );
    })}
  </>
);

/* ---------- Small pieces ---------- */

export const Shield: React.FC<{size: number; color?: string}> = ({size, color = C.gs}) => (
  <svg viewBox="0 0 24 24" width={size} height={size} style={{display: 'block'}}>
    <path fill={color} d="M12 2 4 5v6c0 5 3.4 9.4 8 11 4.6-1.6 8-6 8-11V5l-8-3z" />
  </svg>
);

export type ChipTone = 'secret' | 'masked' | 'plain' | 'killed';
const CHIP: Record<ChipTone, [string, string, string]> = {
  secret: [C.bad, '#FFFFFF', C.bad],
  masked: [C.gsTint, C.gsDeep, C.gs],
  plain: ['#1B3150', '#C9D8EA', C.navyLine],
  killed: [C.badTint, C.bad, C.bad],
};
export const Chip: React.FC<{x: number; y: number; tone: ChipTone; size?: number; style?: React.CSSProperties; children: React.ReactNode}> = ({x, y, tone, size = 22, style, children}) => (
  <div style={{position: 'absolute', left: x, top: y, transform: 'translate(-50%, -50%)', whiteSpace: 'nowrap', fontFamily: MONO, fontSize: size, fontWeight: 600, padding: '6px 14px', borderRadius: 999, background: CHIP[tone][0], color: CHIP[tone][1], border: `2px solid ${CHIP[tone][2]}`, ...style}}>
    {children}
  </div>
);

export const Dot: React.FC<{x: number; y: number; c: string; r?: number; o?: number}> = ({x, y, c, r = 10, o = 1}) => (
  <div style={{position: 'absolute', left: x - r, top: y - r, width: r * 2, height: r * 2, borderRadius: r, background: c, opacity: o, boxShadow: `0 0 ${r * 1.6}px ${c}`}} />
);

export const Ring: React.FC<{x: number; y: number; f: number; at: number; c: string; size?: number}> = ({x, y, f, at, c, size = 140}) => {
  if (f < at || f > at + 20) return null;
  const r = tw(f, at, at + 20);
  return <div style={{position: 'absolute', left: x - size / 2, top: y - size / 2, width: size, height: size, borderRadius: size, border: `4px solid ${c}`, opacity: 1 - r, transform: `scale(${0.3 + r * 1.1})`}} />;
};

// Line-art icons for the dashboard's rail and cards.
const P: Record<string, React.ReactNode> = {
  grid: <><rect x="4" y="4" width="7" height="7" rx="1.5" /><rect x="13" y="4" width="7" height="7" rx="1.5" /><rect x="4" y="13" width="7" height="7" rx="1.5" /><rect x="13" y="13" width="7" height="7" rx="1.5" /></>,
  shield: <path d="M12 3 5 6v5c0 4.5 3 8.3 7 10 4-1.7 7-5.5 7-10V6z" />,
  gauge: <><path d="M4 17a8 8 0 1 1 16 0" /><path d="m12 17 4-5" /></>,
  laptop: <><rect x="5" y="5" width="14" height="10" rx="1.5" /><path d="M3 19h18" /></>,
  file: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v4h4M9 12h6M9 16h6" /></>,
  list: <><path d="M9 6h11M9 12h11M9 18h11" /><circle cx="4.5" cy="6" r="1" /><circle cx="4.5" cy="12" r="1" /><circle cx="4.5" cy="18" r="1" /></>,
  doc: <><path d="M6 3h8l4 4v14H6z" /><path d="M14 3v4h4" /></>,
  mail: <><rect x="3" y="5" width="18" height="14" rx="2" /><path d="m4 7 8 6 8-6" /></>,
  server: <><rect x="4" y="4" width="16" height="7" rx="1.5" /><rect x="4" y="13" width="16" height="7" rx="1.5" /><path d="M8 7.5h.01M8 16.5h.01" /></>,
  download: <><path d="M6 3h8l4 4v14H6z" /><path d="M12 10v7M9 14l3 3 3-3" /></>,
};
export const Icon: React.FC<{name: keyof typeof P; size?: number; color?: string; stroke?: number}> = ({name, size = 28, color = C.sub, stroke = 1.8}) => (
  <svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke={color} strokeWidth={stroke} strokeLinecap="round" strokeLinejoin="round" style={{display: 'block'}}>
    {P[name]}
  </svg>
);

export const Toggle: React.FC<{on: number}> = ({on}) => (
  <div style={{width: 66, height: 36, borderRadius: 18, background: interpolateColors(on, [0, 1], ['#D3DBE6', C.gs]), position: 'relative', flex: 'none'}}>
    <div style={{position: 'absolute', top: 4, left: 4 + on * 30, width: 28, height: 28, borderRadius: 14, background: '#FFFFFF', boxShadow: '0 1px 3px rgba(12,26,43,.3)'}} />
  </div>
);

export const Pill: React.FC<{tone: 'ok' | 'bad' | 'warn' | 'gs'; size?: number; children: React.ReactNode; style?: React.CSSProperties}> = ({tone, size = 20, children, style}) => {
  const m = {ok: [C.okTint, C.ok], bad: [C.badTint, C.bad], warn: [C.warnTint, C.warn], gs: [C.gsTint, C.gsDeep]}[tone];
  return <span style={{display: 'inline-flex', alignItems: 'center', gap: 8, padding: '5px 14px', borderRadius: 999, background: m[0], color: m[1], fontSize: size, fontWeight: 700, whiteSpace: 'nowrap', ...style}}>{children}</span>;
};

export const Cursor: React.FC<{x: number; y: number; press?: number}> = ({x, y, press = 0}) => (
  <svg viewBox="0 0 24 24" width={46} height={46} style={{position: 'absolute', left: x - 6, top: y - 3, transform: `scale(${1 - press * 0.15})`, filter: 'drop-shadow(0 3px 6px rgba(0,0,0,.3))'}}>
    <path d="M4 2 L4 19 L8.5 15 L11.5 22 L14.5 20.7 L11.6 14 L18 14 Z" fill="#FFFFFF" stroke={C.ink} strokeWidth={1.4} strokeLinejoin="round" />
  </svg>
);

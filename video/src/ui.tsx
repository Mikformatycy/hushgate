import React from 'react';
import {AbsoluteFill, Easing, interpolate, useCurrentFrame} from 'remotion';
import {C, MONO} from './theme';

export const ease = Easing.bezier(0.37, 0, 0.63, 1);
const clamp = {extrapolateLeft: 'clamp', extrapolateRight: 'clamp'} as const;

export const prog = (f: number, start: number, dur = 24) =>
  interpolate(f, [start, start + dur], [0, 1], {...clamp, easing: ease});

// Fades the whole scene in and out so cuts are never hard.
export const Scene: React.FC<{dur: number; children: React.ReactNode}> = ({dur, children}) => {
  const f = useCurrentFrame();
  const o = Math.min(prog(f, 0, 15), 1 - prog(f, dur - 18, 18));
  return <AbsoluteFill style={{opacity: o}}>{children}</AbsoluteFill>;
};

export const Appear: React.FC<{at: number; dur?: number; y?: number; style?: React.CSSProperties; children: React.ReactNode}> = ({
  at, dur = 24, y = 14, style, children,
}) => {
  const p = prog(useCurrentFrame(), at, dur);
  return <div style={{opacity: p, transform: `translateY(${(1 - p) * y}px)`, ...style}}>{children}</div>;
};

export const Count: React.FC<{at: number; to: number; dur?: number; suffix?: string}> = ({at, to, dur = 45, suffix = ''}) => {
  const v = interpolate(useCurrentFrame(), [at, at + dur], [0, to], {...clamp, easing: Easing.out(Easing.cubic)});
  return <>{Math.round(v).toLocaleString('en-US')}{suffix}</>;
};

// Outlines that draw themselves, the way a stroke is drawn by hand.
export const DrawBox: React.FC<{x: number; y: number; w: number; h: number; at: number; dur?: number; color?: string}> = ({
  x, y, w, h, at, dur = 30, color = C.fg,
}) => {
  const p = prog(useCurrentFrame(), at, dur);
  return (
    <svg style={{position: 'absolute', left: x, top: y, overflow: 'visible'}} width={w} height={h}>
      <rect x={0} y={0} width={w} height={h} rx={8} fill="none" stroke={color} strokeWidth={2.5} pathLength={1} strokeDasharray="1 1" strokeDashoffset={1 - p} />
    </svg>
  );
};

export const DrawLine: React.FC<{x1: number; y1: number; x2: number; y2: number; at: number; dur?: number; color?: string}> = ({
  x1, y1, x2, y2, at, dur = 24, color = C.dim,
}) => {
  const p = prog(useCurrentFrame(), at, dur);
  return (
    <svg style={{position: 'absolute', left: 0, top: 0, overflow: 'visible'}} width={1920} height={1080}>
      <line x1={x1} y1={y1} x2={x2} y2={y2} stroke={color} strokeWidth={2.5} pathLength={1} strokeDasharray="1 1" strokeDashoffset={1 - p} />
    </svg>
  );
};

// A message travelling along a line: a dot that loops from (x1,y1) to (x2,y2).
export const Dot: React.FC<{x1: number; y1: number; x2: number; y2: number; at: number; period?: number; color?: string}> = ({
  x1, y1, x2, y2, at, period = 60, color = C.blue,
}) => {
  const f = useCurrentFrame();
  if (f < at) return null;
  const t = ease(((f - at) % period) / period);
  const fade = Math.min(t * 6, (1 - t) * 6, 1);
  return (
    <div style={{position: 'absolute', left: x1 + (x2 - x1) * t - 9, top: y1 + (y2 - y1) * t - 9, width: 18, height: 18, borderRadius: 9, background: color, opacity: fade}} />
  );
};

export const Title: React.FC<{eyebrow: string; children: React.ReactNode; at?: number}> = ({eyebrow, children, at = 0}) => (
  <div style={{position: 'absolute', left: 160, top: 110, right: 160}}>
    <Appear at={at}><div style={{fontSize: 32, fontStyle: 'italic', color: C.muted}}>{eyebrow}</div></Appear>
    <Appear at={at + 8}><div style={{fontSize: 72, fontWeight: 600, marginTop: 6, lineHeight: 1.15}}>{children}</div></Appear>
  </div>
);

export const Footnote: React.FC<{at: number; children: React.ReactNode}> = ({at, children}) => (
  <Appear at={at} style={{position: 'absolute', left: 160, right: 160, bottom: 56, fontSize: 24, fontStyle: 'italic', color: C.muted}}>
    {children}
  </Appear>
);

export const Mono: React.FC<{color?: string; size?: number; children: React.ReactNode}> = ({color = C.fg, size = 30, children}) => (
  <span style={{fontFamily: MONO, fontSize: size, color}}>{children}</span>
);

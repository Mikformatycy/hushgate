import React from 'react';
import {C} from '../theme';
import {Icon, Pill, Shield, box, bump, easeOut, kc, kf, kr, sp, tw, win} from '../kit';
import type {Rect} from '../kit';
import {DETAIL, DOT, FRAME, GATE, T, TILE, ZERO} from './layout';
import {gateFlash} from './Flow';
import {AttackDetail, AuditDetail, Chart, PolicyDetail, ShadowDetail, SpendDetail} from './Details';

const RAIL = ['grid', 'shield', 'gauge', 'laptop', 'file', 'list'] as const;
const ACTIVE: [number, number][] = [[480, 0], [T.attack[0], 0], [T.attack[0] + 12, 1], [T.spend[0], 1], [T.spend[0] + 12, 2], [T.shadow[0], 2], [T.shadow[0] + 12, 3], [T.policy[0], 3], [T.policy[0] + 12, 4], [T.audit[0], 4], [T.audit[0] + 12, 5]];

const Frame: React.FC<{f: number}> = ({f}) => {
  const o = tw(f, T.frameIn, T.frameIn + 28);
  if (o <= 0) return null;
  const s = kf(f, [[T.frameIn, 0.965], [T.frameIn + 44, 1]], easeOut);
  const a = kf(f, ACTIVE);
  return (
    <div style={{...box(FRAME), background: '#F7F9FC', border: `1.5px solid ${C.border}`, boxShadow: '0 30px 80px rgba(12,26,43,.08)', opacity: o, transform: `scale(${s})`, overflow: 'hidden'}}>
      <div style={{height: 76, background: '#FFFFFF', borderBottom: `1.5px solid ${C.border}`, display: 'flex', alignItems: 'center', gap: 14, padding: '0 30px 0 28px'}}>
        <Shield size={32} />
        <span style={{fontSize: 26, fontWeight: 800, color: C.ink, letterSpacing: '-0.02em'}}>HushGate</span>
        <span style={{fontSize: 20, color: C.sub, fontWeight: 500, marginLeft: 6}}>Overview</span>
        <span style={{marginLeft: 'auto'}}><Pill tone="ok"><span style={{width: 10, height: 10, borderRadius: 5, background: C.ok}} />Live</Pill></span>
      </div>
      <div style={{position: 'absolute', left: 0, top: 76, bottom: 0, width: 92, background: '#FFFFFF', borderRight: `1.5px solid ${C.border}`}}>
        <div style={{position: 'absolute', left: 18, top: 30 + a * 70, width: 56, height: 56, borderRadius: 16, background: C.gsTint}} />
        {RAIL.map((n, i) => (
          <div key={n} style={{position: 'absolute', left: 32, top: 44 + i * 70}}>
            <Icon name={n} color={Math.abs(a - i) < 0.5 ? C.gsDeep : '#8A98AB'} />
          </div>
        ))}
      </div>
    </div>
  );
};

type TileDef = {label: string; icon: 'shield' | 'gauge' | 'laptop' | 'grid'; value: (f: number) => number; changeAt: number; foot: React.ReactNode};
const count = (f: number, at: number, to: number) => Math.round(tw(f, at, at + 14, 0, to));
const TILES: TileDef[] = [
  {label: 'Leaks to the model', icon: 'grid', value: () => 0, changeAt: -1, foot: <Pill tone="ok" size={17}>✓ every secret masked</Pill>},
  {label: 'Attacks stopped', icon: 'shield', value: (f) => count(f, T.attack[1] - 4, 3), changeAt: T.attack[1] - 4, foot: '19 live signatures'},
  {label: 'Budget stops', icon: 'gauge', value: (f) => count(f, T.spend[1] - 4, 1), changeAt: T.spend[1] - 4, foot: 'limits per agent'},
  {label: 'Shadow AI blocked', icon: 'laptop', value: (f) => count(f, T.shadow[1] - 4, 1), changeAt: T.shadow[1] - 4, foot: 'unknown devices'},
];

export const TileBody: React.FC<{d: TileDef; f: number}> = ({d, f}) => (
  <div style={{position: 'absolute', inset: 0, padding: '24px 28px', display: 'flex', flexDirection: 'column', justifyContent: 'space-between'}}>
    <div style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center'}}>
      <span style={{fontSize: 21, color: C.sub, fontWeight: 700}}>{d.label}</span>
      <Icon name={d.icon} size={26} color={C.gs} />
    </div>
    <div style={{fontSize: 62, fontWeight: 800, color: C.ink, letterSpacing: '-0.03em', lineHeight: 1}}>{d.value(f)}</div>
    <div style={{fontSize: 18, color: C.sub, fontWeight: 600}}>{d.foot}</div>
  </div>
);

const EXPANDS: [number, readonly [number, number]][] = [[1, T.attack], [2, T.spend], [3, T.shadow]];

const Tiles: React.FC<{f: number}> = ({f}) => (
  <>
    {[1, 2, 3].map((i) => {
      const p = sp(f, 466 + i * 6, {damping: 16, stiffness: 150});
      if (p <= 0.001) return null;
      const [, w] = EXPANDS.find(([t]) => t === i)!;
      const hidden = f >= w[0] && f < w[1];
      const d = TILES[i];
      const flash = bump(f, d.changeAt, 24);
      return (
        <div key={i} style={{...box(TILE(i)), background: '#FFFFFF', border: `1.5px solid ${kc(flash, [[0, C.border], [1, C.gs]])}`, boxShadow: flash ? `0 0 0 ${flash * 6}px rgba(114,151,197,.25)` : undefined, opacity: hidden ? 0 : Math.min(1, p * 1.5), transform: `translateY(${(1 - p) * 30}px) scale(${0.92 + 0.08 * p})`}}>
          <TileBody d={d} f={f} />
        </div>
      );
    })}
  </>
);

// The gate itself becomes the "zero leaks" card, which becomes the first tile.
const Hero: React.FC<{f: number}> = ({f}) => {
  if (f < T.gateOn) return null;
  const r = kr(f, [[T.gateOn, DOT], [T.gateOn + 20, GATE], [T.zeroIn, GATE], [T.zeroDone, ZERO], [T.toTile, ZERO], [T.tileDone, TILE(0)]]);
  const flash = f < T.zeroIn ? gateFlash(f) : 0;
  const zeroO = kf(f, [[362, 0], [384, 1], [T.toTile, 1], [T.toTile + 12, 0]]);
  const zp = sp(f, 368, {damping: 9, stiffness: 140});
  const card = tw(f, T.zeroIn, T.zeroDone);
  return (
    <div
      style={{
        ...box(r),
        overflow: 'hidden',
        background: kc(f, [[T.zeroIn, C.navy2], [T.zeroIn + 12, '#FFFFFF']]),
        border: `${kf(f, [[T.toTile, 3], [T.tileDone, 1.5]])}px solid ${kc(f, [[T.toTile, C.gs], [T.tileDone, C.border]])}`,
        boxShadow: f < T.zeroIn + 10 ? `0 0 ${30 + flash * 60}px rgba(114,151,197,${0.3 + flash * 0.5})` : `0 30px 80px rgba(12,26,43,${0.14 * (1 - tw(f, T.toTile, T.tileDone))})`,
      }}
    >
      <div style={{position: 'absolute', left: '50%', top: '50%', transform: 'translate(-50%, -50%)', opacity: win(f, T.gateActive, T.zeroIn + 14, 8, 14)}}><Shield size={64} /></div>
      {card > 0 && (
        <div style={{position: 'absolute', left: 0, top: 0, width: ZERO.w, padding: '44px 54px', opacity: zeroO}}>
          <div style={{fontSize: 30, color: C.sub, fontWeight: 700}}>Plaintext secrets that reached the model</div>
          <div style={{fontSize: 220, fontWeight: 800, color: C.gsDeep, lineHeight: 1.05, letterSpacing: '-0.04em', transform: `scale(${zp})`, transformOrigin: '0 70%'}}>0</div>
          <Pill tone="ok" size={26}>✓ every secret masked in flight</Pill>
        </div>
      )}
      <div style={{position: 'absolute', left: 0, top: 0, width: TILE(0).w, height: TILE(0).h, opacity: tw(f, 466, 484)}}><TileBody d={TILES[0]} f={f} /></div>
    </div>
  );
};

const Expander: React.FC<{f: number; tile: number; w: readonly [number, number]; children: React.ReactNode}> = ({f, tile, w: [s, e], children}) => {
  if (f < s || f >= e) return null;
  const r: Rect = kr(f, [[s, TILE(tile)], [s + 22, DETAIL], [e - 22, DETAIL], [e, TILE(tile)]]);
  return (
    <div style={{...box(r), overflow: 'hidden', background: '#FFFFFF', border: `2px solid ${C.gs}`, boxShadow: '0 24px 60px rgba(12,26,43,.14)'}}>
      <div style={{position: 'absolute', left: 0, top: 0, width: DETAIL.w, height: DETAIL.h, opacity: win(f, s + 18, e - 16, 10, 8)}}>{children}</div>
    </div>
  );
};

const DetailBase: React.FC<{f: number}> = ({f}) => {
  const o = tw(f, 476, 494);
  if (o <= 0) return null;
  return (
    <div style={{...box(DETAIL), overflow: 'hidden', background: '#FFFFFF', border: `1.5px solid ${C.border}`, opacity: o}}>
      <Chart f={f} />
      <PolicyDetail f={f} />
      <AuditDetail f={f} />
    </div>
  );
};

export const Dash: React.FC<{f: number}> = ({f}) => {
  const c = kf(f, [[T.end, 0], [T.end + 40, 1]]);
  const o = 1 - tw(f, T.end + 20, T.end + 38);
  if (o <= 0) return null;
  return (
    <div style={{position: 'absolute', inset: 0, transformOrigin: '960px 600px', transform: `translateY(${-130 * c}px) scale(${1 - 0.88 * c})`, opacity: o, filter: c > 0 ? `blur(${c * 6}px)` : undefined}}>
      <Frame f={f} />
      <DetailBase f={f} />
      <Tiles f={f} />
      <Hero f={f} />
      <div style={{position: 'absolute', left: 0, right: 0, top: ZERO.y + ZERO.h + 26, textAlign: 'center', fontSize: 20, fontWeight: 600, color: C.sub, opacity: win(f, 388, T.toTile + 6, 10, 8)}}>Across HushGate's test scenarios</div>
      <Expander f={f} tile={1} w={T.attack}><AttackDetail f={f} /></Expander>
      <Expander f={f} tile={2} w={T.spend}><SpendDetail f={f} /></Expander>
      <Expander f={f} tile={3} w={T.shadow}><ShadowDetail f={f} /></Expander>
    </div>
  );
};

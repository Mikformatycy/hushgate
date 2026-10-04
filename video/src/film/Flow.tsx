import {C, MONO} from '../theme';
import {Chip, bump, easeInOut, sp, tw} from '../kit';
import type {ChipTone} from '../kit';
import {ENV, GATE, GATE_X, MODEL, T} from './layout';

type Tier = 'C3' | 'C2' | 'C1';
export const ENV_LINES: [string, string, Tier, string][] = [
  ['DB_PASSWORD', 'Pr0d-Adm1n-2026', 'C3', '{{VAULT_ENV_DB_PASSWORD}}'],
  ['API_KEY', 'sk_live_51Hx9Qa', 'C3', '{{VAULT_ENV_API_KEY}}'],
  ['CUSTOMER_IBAN', 'PL61 1090 1014', 'C2', '{{VAULT_IBAN_7f3a91}}'],
  ['SUPPORT_EMAIL', 'help@corp.example', 'C1', 'help@corp.example'],
];
const TIER_TONE: Record<Tier, [string, string]> = {C3: [C.bad, 'rgba(213,68,59,'], C2: [C.warn, 'rgba(201,138,26,'], C1: [C.gs, 'rgba(114,151,197,']};
const lineY = (i: number) => ENV.y + 126 + i * 76;
const KEY_X = ENV.x + 36;
const KEY_CH = 26 * 0.6, VAL_CH = 22 * 0.6;
const valX = (i: number) => {
  const [k, v] = ENV_LINES[i];
  return KEY_X + (k.length + 1) * KEY_CH + 10 + (v.length * VAL_CH + 32) / 2;
};
const DUR = 50;
// Frame at which a chip reaches the gate's centre line.
const solve = (u: number) => {
  let lo = 0, hi = 1;
  for (let n = 0; n < 24; n++) {
    const m = (lo + hi) / 2;
    if (easeInOut(m) < u) lo = m; else hi = m;
  }
  return (lo + hi) / 2;
};
// Chips stop at the model's edge, each line at its own height, so they never pile up.
const CHIPS = Array.from({length: 27}, (_, k) => {
  const L = k % 4;
  const s = T.leak + k * 8;
  const x0 = k < 4 ? valX(L) : ENV.x + ENV.w - 24;
  const tx = MODEL.x - MODEL.r * 0.55, ty = MODEL.y + (L - 1.5) * 30;
  return {k, L, s, x0, y0: lineY(L), tx, ty, cross: s + solve((GATE_X - x0) / (tx - x0)) * DUR};
});
const masks = (c: (typeof CHIPS)[number]) => ENV_LINES[c.L][2] !== 'C1' && c.cross >= T.gateActive;

const chipView = (c: (typeof CHIPS)[number], after: boolean): [ChipTone, string] => {
  const [, v, tier, ph] = ENV_LINES[c.L];
  if (tier === 'C1') return ['plain', v];
  return after && masks(c) ? ['masked', ph] : ['secret', v];
};

export const gateFlash = (f: number) => Math.max(0, ...CHIPS.filter(masks).map((c) => bump(f, c.cross, 12)));

export const Flow: React.FC<{f: number}> = ({f}) => {
  const fade = 1 - tw(f, T.zeroIn, T.zeroIn + 22);
  if (fade <= 0) return null;
  const card = sp(f, 4, {damping: 20, stiffness: 120});
  const model = sp(f, 16, {damping: 18, stiffness: 120});
  const arrived = CHIPS.filter((c) => f >= c.s + DUR);
  const last = arrived[arrived.length - 1];
  const ring = f < 214 ? (tw(f, 100, 150) > 0 ? mix(C.navyLine, C.bad, tw(f, 100, 150)) : C.navyLine) : mix(C.bad, C.gs, tw(f, 214, 262));
  const glow = Math.max(0, ...CHIPS.map((c) => bump(f, c.s + DUR, 10)));
  return (
    <div style={{position: 'absolute', inset: 0, opacity: fade}}>
      {/* agent workspace: the .env file */}
      <div style={{position: 'absolute', left: ENV.x, top: ENV.y, width: ENV.w, height: ENV.h, borderRadius: ENV.r, background: C.navy2, border: `1.5px solid ${C.navyLine}`, opacity: Math.min(1, card * 1.5), transform: `scale(${0.9 + 0.1 * card})`}}>
        <div style={{height: 58, display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 26px', borderBottom: `1.5px solid ${C.navyLine}`}}>
          <span style={{fontFamily: MONO, fontSize: 22, color: C.gs, fontWeight: 600}}>.env</span>
          <span style={{fontSize: 20, color: C.navyText, fontWeight: 600}}>agent workspace</span>
        </div>
      </div>
      {ENV_LINES.map(([k, v, tier], i) => {
        const h = tw(f, 40 + i * 12, 52 + i * 12);
        const [col, rgba] = TIER_TONE[tier];
        return (
          <div key={k} style={{opacity: tw(f, 14 + i * 5, 24 + i * 5)}}>
            <div style={{position: 'absolute', left: KEY_X, top: lineY(i) - 18, fontFamily: MONO, fontSize: 26, color: C.navyText}}>{k}=</div>
            <div style={{position: 'absolute', left: valX(i), top: lineY(i), transform: 'translate(-50%, -50%)', fontFamily: MONO, fontSize: 22, fontWeight: 600, padding: '6px 14px', borderRadius: 999, color: '#FFFFFF', background: `${rgba}${0.35 * h})`, border: `2px solid ${rgba}${h})`, whiteSpace: 'nowrap'}}>{v}</div>
            <div style={{position: 'absolute', left: ENV.x + ENV.w - 78, top: lineY(i) - 16, fontSize: 18, fontWeight: 800, color: col, opacity: h, border: `2px solid ${col}`, borderRadius: 8, padding: '1px 8px'}}>{tier}</div>
          </div>
        );
      })}
      {/* the model provider */}
      <div style={{position: 'absolute', left: MODEL.x - MODEL.r, top: MODEL.y - MODEL.r, width: MODEL.r * 2, height: MODEL.r * 2, borderRadius: MODEL.r, background: C.navy2, border: `3px solid ${ring}`, boxShadow: `0 0 ${20 + glow * 50}px ${ring}`, opacity: Math.min(1, model * 1.5), transform: `scale(${0.7 + 0.3 * model})`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#FFFFFF', fontSize: 40, fontWeight: 800}}>LLM</div>
      <div style={{position: 'absolute', left: MODEL.x - 200, width: 400, top: MODEL.y + MODEL.r + 18, textAlign: 'center', fontSize: 22, color: C.navyText, fontWeight: 600, opacity: model}}>model provider</div>
      {last && (
        <div style={{position: 'absolute', left: MODEL.x - 260, width: 520, top: 760, textAlign: 'center'}}>
          <div style={{fontSize: 20, color: C.navyText, marginBottom: 30}}>the model sees</div>
          <Chip x={260} y={50} tone={chipView(last, true)[0]} key={last.k} style={{scale: `${0.85 + 0.15 * sp(f, last.s + DUR, {damping: 12, stiffness: 240})}`}}>{chipView(last, true)[1]}</Chip>
        </div>
      )}
      {/* data in flight */}
      {CHIPS.map((c) => {
        if (f < c.s) return null;
        const t = Math.min(1, (f - c.s) / DUR);
        if (t >= 1) return null;
        const e = easeInOut(t);
        const x = c.x0 + (c.tx - c.x0) * e;
        const y = c.y0 + (c.ty - c.y0) * e;
        const after = f >= c.cross;
        const [tone, label] = chipView(c, after);
        const pop = after && masks(c) ? 0.8 + 0.2 * sp(f, c.cross, {damping: 12, stiffness: 260}) : 1;
        const o = (c.k < 4 ? 1 : Math.min(1, (f - c.s) / 6)) * (t > 0.8 ? (1 - t) / 0.2 : 1);
        return <Chip key={c.k} x={x} y={y} tone={tone} style={{opacity: o, scale: `${pop}`}}>{label}</Chip>;
      })}
      <div style={{position: 'absolute', left: GATE_X - 150, width: 300, top: GATE.y + GATE.h + 22, textAlign: 'center', fontSize: 28, fontWeight: 800, color: '#FFFFFF', opacity: Math.min(tw(f, 192, 204), 1 - tw(f, T.zeroIn, T.zeroIn + 12))}}>HushGate</div>
    </div>
  );
};

const mix = (a: string, b: string, t: number) => {
  const p = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  const [x, y] = [p(a), p(b)];
  return `rgb(${x.map((v, i) => Math.round(v + (y[i] - v) * t)).join(',')})`;
};

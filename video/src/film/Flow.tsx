import {C, MONO, SANS} from '../theme';
import {Chip, bump, easeInOut, kf, sp, tw} from '../kit';
import type {ChipTone} from '../kit';
import {ENV, ENV_WIDE, GATE, GATE_X, MODEL, T} from './layout';

type Tier = 'C3' | 'C2' | 'C1';
// .env values get a per-variable placeholder; values found by the detectors get kind + hash.
const ENV_LINES: [string, string, Tier, string][] = [
  ['DB_PASSWORD', 'Pr0d-Adm1n-2026', 'C3', '{{VAULT_ENV_DB_PASSWORD}}'],
  ['API_KEY', 'sk_live_51Hx9Qa', 'C3', '{{VAULT_ENV_API_KEY}}'],
  ['CUSTOMER_IBAN', 'PL61 1090 1014', 'C2', '{{VAULT_ENV_CUSTOMER_IBAN}}'],
  ['SUPPORT_EMAIL', 'help@corp.example', 'C1', 'help@corp.example'],
];
const PII_LINES: [string, string, string, string, string][] = [
  ['PESEL', '44051401359', '44051401359', '{{VAULT_PESEL_9F2C41A0}}', 'checksum ✓'],
  ['IBAN', 'PL61 1090 1014 0000 0712 1981 2874', 'PL61 1090 … 2874', '{{VAULT_IBAN_7F3A91C2}}', 'mod-97 ✓'],
  ['Card', '4111 1111 1111 1111', '4111 … 1111', '{{VAULT_CARD_C41E08B7}}', 'Luhn ✓'],
];
const TIER: Record<Tier, [string, string]> = {C3: ['#E8706A', 'rgba(213,68,59,'], C2: [C.gs, 'rgba(114,151,197,'], C1: [C.navyText, 'rgba(147,166,190,']};
const OK_ON_DARK = '#63C795';

const lineY = (i: number) => ENV.y + 126 + i * 76;
const KEY_X = ENV.x + 36;
const envValX = (i: number) => {
  const [k, v] = ENV_LINES[i];
  return KEY_X + (k.length + 1) * 15.6 + 10 + (v.length * 13.2 + 32) / 2;
};
const piiValX = (i: number) => KEY_X + 104 + (PII_LINES[i][1].length * 12 + 32) / 2;

const DUR = 50;
const solve = (u: number) => {
  let lo = 0, hi = 1;
  for (let n = 0; n < 24; n++) {
    const m = (lo + hi) / 2;
    if (easeInOut(m) < u) lo = m; else hi = m;
  }
  return (lo + hi) / 2;
};
type Flying = {id: string; s: number; x0: number; y0: number; tx: number; ty: number; raw: string; ph: string; plain: boolean; cross: number};
const fly = (id: string, s: number, x0: number, y0: number, lane: number, raw: string, ph: string, plain: boolean): Flying => {
  const tx = MODEL.x - MODEL.r * 0.55, ty = MODEL.y + lane * 30;
  return {id, s, x0, y0, tx, ty, raw, ph, plain, cross: s + solve((GATE_X - x0) / (tx - x0)) * DUR};
};
// Copies of the .env values stream out, then the values the detectors found in the prompt.
const CHIPS: Flying[] = [
  ...Array.from({length: 23}, (_, k) => {
    const L = k % 4, [, v, tier, ph] = ENV_LINES[L];
    return fly(`e${k}`, T.leak + k * 8, k < 4 ? envValX(L) : ENV.x + ENV.w - 24, lineY(L), L - 1.5, v, ph, tier === 'C1');
  }),
  ...Array.from({length: 6}, (_, j) => {
    const i = j % 3, [, , raw, ph] = PII_LINES[i];
    return fly(`p${j}`, T.piiStream + j * 11, j < 3 ? piiValX(i) : ENV.x + ENV_WIDE - 24, lineY(i + 1), i - 1, raw, ph, false);
  }),
];
const masks = (c: Flying) => !c.plain && c.cross >= T.gateActive;
const view = (c: Flying, after: boolean): [ChipTone, string] => (c.plain ? ['plain', c.raw] : after && masks(c) ? ['masked', c.ph] : ['secret', c.raw]);

export const gateFlash = (f: number) => Math.max(0, ...CHIPS.filter(masks).map((c) => bump(f, c.cross, 12)));

export const Flow: React.FC<{f: number}> = ({f}) => {
  const fade = 1 - tw(f, T.zeroIn, T.zeroIn + 22);
  if (fade <= 0) return null;
  const card = sp(f, 4, {damping: 20, stiffness: 120});
  const model = sp(f, 16, {damping: 18, stiffness: 120});
  const envO = 1 - tw(f, T.pii, T.pii + 12);
  const piiO = tw(f, T.pii + 10, T.pii + 24);
  const width = kf(f, [[T.pii, ENV.w], [T.pii + 16, ENV_WIDE]]);
  const arrived = CHIPS.filter((c) => f >= c.s + DUR);
  const last = arrived.sort((a, b) => a.s + DUR - (b.s + DUR))[arrived.length - 1];
  const ring = f < T.gateActive + 40 ? mix(C.navyLine, C.bad, tw(f, T.leak + 10, T.leak + 60)) : mix(C.bad, C.gs, tw(f, T.gateActive + 40, T.gateActive + 88));
  const glow = Math.max(0, ...CHIPS.map((c) => bump(f, c.s + DUR, 10)));
  return (
    <div style={{position: 'absolute', inset: 0, opacity: fade}}>
      {/* the agent's workspace: first its .env file, then the prompt it writes */}
      <div style={{position: 'absolute', left: ENV.x, top: ENV.y, width, height: ENV.h, borderRadius: ENV.r, background: C.navy2, border: `1.5px solid ${C.navyLine}`, opacity: Math.min(1, card * 1.5), transform: `scale(${0.9 + 0.1 * card})`, transformOrigin: '0 50%'}}>
        <div style={{position: 'relative', height: 58, borderBottom: `1.5px solid ${C.navyLine}`}}>
          {([[envO, '.env', 'agent workspace'], [piiO, 'prompt', 'agent → model']] as const).map(([o, l, r]) => (
            <div key={l} style={{position: 'absolute', inset: 0, display: 'flex', alignItems: 'center', justifyContent: 'space-between', padding: '0 26px', opacity: o}}>
              <span style={{fontFamily: MONO, fontSize: 22, color: C.gs, fontWeight: 600}}>{l}</span>
              <span style={{fontSize: 20, color: C.navyText, fontWeight: 600}}>{r}</span>
            </div>
          ))}
        </div>
      </div>
      {envO > 0 && ENV_LINES.map(([k, v, tier], i) => {
        const h = tw(f, 40 + i * 12, 52 + i * 12);
        const [col, rgba] = TIER[tier];
        return (
          <div key={k} style={{opacity: tw(f, 14 + i * 5, 24 + i * 5) * envO}}>
            <div style={{position: 'absolute', left: KEY_X, top: lineY(i) - 18, fontFamily: MONO, fontSize: 26, color: C.navyText}}>{k}=</div>
            <div style={{position: 'absolute', left: envValX(i), top: lineY(i), transform: 'translate(-50%, -50%)', fontFamily: MONO, fontSize: 22, fontWeight: 600, padding: '6px 14px', borderRadius: 999, color: '#FFFFFF', background: `${rgba}${0.35 * h})`, border: `2px solid ${rgba}${h})`, whiteSpace: 'nowrap'}}>{v}</div>
            <div style={{position: 'absolute', left: ENV.x + ENV.w - 78, top: lineY(i) - 16, fontSize: 18, fontWeight: 800, color: col, opacity: h, border: `2px solid ${col}`, borderRadius: 8, padding: '1px 8px'}}>{tier}</div>
          </div>
        );
      })}
      {piiO > 0 && (
        <div style={{opacity: piiO}}>
          <div style={{position: 'absolute', left: KEY_X, top: lineY(0) - 18, fontFamily: SANS, fontSize: 25, fontWeight: 600, color: '#E6EDF5'}}>Refund the customer and confirm:</div>
          {PII_LINES.map(([label, v, , , check], i) => {
            const at = T.piiDetect + i * 10;
            const h = tw(f, at, at + 12);
            const tag = sp(f, at + 4, {damping: 12, stiffness: 220});
            return (
              <div key={label}>
                <div style={{position: 'absolute', left: KEY_X, top: lineY(i + 1) - 16, fontSize: 22, fontWeight: 700, color: C.navyText}}>{label}</div>
                <div style={{position: 'absolute', left: piiValX(i), top: lineY(i + 1), transform: 'translate(-50%, -50%)', fontFamily: MONO, fontSize: 20, fontWeight: 600, padding: '6px 14px', borderRadius: 999, color: '#FFFFFF', background: `rgba(114,151,197,${0.35 * h})`, border: `2px solid rgba(114,151,197,${0.25 + 0.75 * h})`, whiteSpace: 'nowrap'}}>{v}</div>
                <div style={{position: 'absolute', right: 1920 - (ENV.x + ENV_WIDE - 24), top: lineY(i + 1) - 17, fontSize: 18, fontWeight: 800, color: OK_ON_DARK, border: `2px solid ${OK_ON_DARK}`, borderRadius: 8, padding: '2px 10px', opacity: Math.min(1, Math.max(0, tag) * 1.5), transform: `scale(${0.6 + 0.4 * tag})`, whiteSpace: 'nowrap'}}>{check}</div>
              </div>
            );
          })}
        </div>
      )}
      {/* the model provider */}
      <div style={{position: 'absolute', left: MODEL.x - MODEL.r, top: MODEL.y - MODEL.r, width: MODEL.r * 2, height: MODEL.r * 2, borderRadius: MODEL.r, background: C.navy2, border: `3px solid ${ring}`, boxShadow: `0 0 ${20 + glow * 50}px ${ring}`, opacity: Math.min(1, model * 1.5), transform: `scale(${0.7 + 0.3 * model})`, display: 'flex', alignItems: 'center', justifyContent: 'center', color: '#FFFFFF', fontSize: 40, fontWeight: 800}}>LLM</div>
      <div style={{position: 'absolute', left: MODEL.x - 200, width: 400, top: MODEL.y + MODEL.r + 18, textAlign: 'center', fontSize: 22, color: C.navyText, fontWeight: 600, opacity: model}}>model provider</div>
      {last && (
        <div style={{position: 'absolute', left: MODEL.x - 260, width: 520, top: 760, textAlign: 'center'}}>
          <div style={{fontSize: 20, color: C.navyText, marginBottom: 30}}>the model sees</div>
          <Chip key={last.id} x={260} y={50} tone={view(last, true)[0]} style={{scale: `${0.85 + 0.15 * sp(f, last.s + DUR, {damping: 12, stiffness: 240})}`}}>{view(last, true)[1]}</Chip>
        </div>
      )}
      {/* data in flight */}
      {CHIPS.map((c) => {
        if (f < c.s) return null;
        const t = Math.min(1, (f - c.s) / DUR);
        if (t >= 1) return null;
        const e = easeInOut(t);
        const after = f >= c.cross;
        const [tone, label] = view(c, after);
        const pop = after && masks(c) ? 0.8 + 0.2 * sp(f, c.cross, {damping: 12, stiffness: 260}) : 1;
        const detached = c.id === 'e0' || c.id === 'e1' || c.id === 'e2' || c.id === 'e3' || c.id === 'p0' || c.id === 'p1' || c.id === 'p2';
        const o = (detached ? 1 : Math.min(1, (f - c.s) / 6)) * (t > 0.8 ? (1 - t) / 0.2 : 1);
        return <Chip key={c.id} x={c.x0 + (c.tx - c.x0) * e} y={c.y0 + (c.ty - c.y0) * e} tone={tone} style={{opacity: o, scale: `${pop}`}}>{label}</Chip>;
      })}
      <div style={{position: 'absolute', left: GATE_X - 150, width: 300, top: GATE.y + GATE.h + 22, textAlign: 'center', fontSize: 28, fontWeight: 800, color: '#FFFFFF', opacity: Math.min(tw(f, T.gateOn + 20, T.gateOn + 32), 1 - tw(f, T.zeroIn, T.zeroIn + 12))}}>HushGate</div>
    </div>
  );
};

const mix = (a: string, b: string, t: number) => {
  const p = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16));
  const [x, y] = [p(a), p(b)];
  return `rgb(${x.map((v, i) => Math.round(v + (y[i] - v) * t)).join(',')})`;
};

import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Card, Cloud, Gate, Headline, Laptop, Pop, Server, Status, bump, easeInOut, tw} from '../kit';

const DEV = [{n: 'jdoe-macbook', y: 330}, {n: 'ci-runner-01', y: 530}, {n: 'guest-laptop', y: 760}];
const GX = 820, GW = 150;
const TRIES = [64, 116];
const HIT = 24;

const Dot: React.FC<{x: number; y: number; c: string; o?: number}> = ({x, y, c, o = 1}) => (
  <div style={{position: 'absolute', left: x - 11, top: y - 11, width: 22, height: 22, borderRadius: 11, background: c, opacity: o, boxShadow: `0 0 18px ${c}`}} />
);

export const ShadowAI = () => {
  const f = useCurrentFrame();
  const blocked = f >= TRIES[0] + HIT;
  const flash = Math.max(0, ...TRIES.map((t) => bump(f, t + HIT, 12)));
  return (
    <Bg color={C.night}>
      <Headline text="Shadow AI, blocked." at={6} top={64} color="#ffffff" accent={C.bad} accentWords={[2]} size={88} />
      {DEV.map((d, i) => {
        const bad = i === 2 && blocked;
        return (
          <div key={d.n}>
            <Laptop x={230} y={d.y} ring={bad ? C.bad : i < 2 ? C.ok : C.nightLine} />
            <div style={{position: 'absolute', left: 100, width: 260, top: d.y + 64, textAlign: 'center', fontFamily: MONO, fontSize: 20, color: C.nightText}}>{d.n}</div>
            {i < 2 && <Pop at={8 + i * 5} from={0.2} y={0} style={{position: 'absolute', left: 290, top: d.y - 66}}><Badge c={C.ok}>✓</Badge></Pop>}
            {i === 2 && <Pop at={TRIES[0] + HIT} from={0.2} y={0} style={{position: 'absolute', left: 290, top: d.y - 66}}><Badge c={C.bad}>⊗</Badge></Pop>}
          </div>
        );
      })}
      <Gate x={GX} y={250} w={GW} h={640} at={0} flash={0} />
      <Cloud x={1370} y={250} w={400} h={220} stroke={C.ok} glow={0.2} />
      <div style={{position: 'absolute', left: 1370, width: 400, top: 480, textAlign: 'center', fontFamily: MONO, fontSize: 21, color: '#c8d3e0'}}>api.anthropic.com</div>
      <Server x={1570} y={720} ring={blocked ? C.bad : C.nightLine} />
      <div style={{position: 'absolute', left: 1370, width: 400, top: 810, textAlign: 'center', fontFamily: MONO, fontSize: 21, color: C.nightText}}>llm.sketchy-vps.example</div>
      {Array.from({length: 12}, (_, k) => {
        const d = DEV[k % 2], s = 6 + k * 13;
        const t1 = tw(f, s, s + 22, 0, 1, easeInOut), t2 = tw(f, s + 28, s + 52, 0, 1, easeInOut);
        if (f < s || t2 >= 1) return null;
        if (t1 < 1) return <Dot key={k} x={300 + (GX - 300) * t1} y={d.y} c={C.link} />;
        if (f < s + 28) return null;
        return <Dot key={k} x={GX + GW + (1570 - GX - GW) * t2} y={d.y + (360 - d.y) * t2} c={C.link} o={t2 > 0.85 ? (1 - t2) / 0.15 : 1} />;
      })}
      {TRIES.map((s) => {
        const t = tw(f, s, s + HIT, 0, 1, easeInOut);
        const r = tw(f, s + HIT, s + HIT + 18);
        return (
          <div key={s}>
            {f >= s && t < 1 && <Dot x={300 + (GX - 10 - 300) * t} y={760} c={C.bad} />}
            {f >= s + HIT && r < 1 && <div style={{position: 'absolute', left: GX - 70, top: 690, width: 140, height: 140, borderRadius: 70, border: `4px solid ${C.bad}`, transform: `scale(${0.3 + r * 1.2})`, opacity: 1 - r}} />}
          </div>
        );
      })}
      <div style={{position: 'absolute', left: GX - 30, top: 735, fontSize: 48, color: C.bad, fontWeight: 700, opacity: blocked ? 1 : 0, transform: `scale(${1 + flash * 0.4})`}}>⊗</div>
      <Pop at={150} y={60} style={{position: 'absolute', left: 330, top: 938, width: 1260}}>
        <Card style={{height: 86, display: 'flex', alignItems: 'center', gap: 34, padding: '0 34px', fontSize: 24, background: '#fdf2f0'}}>
          <Status tone="bad">Blocked device</Status>
          <span style={{fontFamily: MONO, fontWeight: 600}}>guest-laptop</span>
          <span style={{fontFamily: MONO, color: C.sub}}>→ llm.sketchy-vps.example</span>
          <span style={{marginLeft: 'auto', color: C.sub}}>recognised by request shape</span>
        </Card>
      </Pop>
    </Bg>
  );
};

const Badge: React.FC<{c: string; children: React.ReactNode}> = ({c, children}) => (
  <div style={{width: 44, height: 44, borderRadius: 22, background: c, color: '#fff', fontSize: 26, fontWeight: 700, display: 'flex', alignItems: 'center', justifyContent: 'center', boxShadow: '0 0 0 4px #0f1722'}}>{children}</div>
);

import {random, useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {AgentNode, Bg, Chip, Cloud, Gate, Headline, Metric, Pop, Status, bump, easeInOut, sp, tw} from '../kit';

const AGENTS = [330, 480, 630, 780];
const SECRETS: [string, string][] = [
  ['Pr0d-Adm1n-2026', '{{VAULT_ENV_DB_PASSWORD}}'],
  ['wJalrXUtnFEMI…', '{{VAULT_ENV_AWS_SECRET}}'],
  ['PL61 1090 1014…', '{{VAULT_IBAN_7f3a91}}'],
  ['sk_live_51Hx…', '{{VAULT_ENV_STRIPE_KEY}}'],
];
const PROMPTS = ['refactor billing.ts', 'write the tests', 'fix the CI job'];
const GX = 850, GW = 160;
const CHIPS = Array.from({length: 18}, (_, i) => {
  const plain = i % 3 === 2;
  const s = SECRETS[i % 4];
  return {
    start: 26 + i * 11,
    y: AGENTS[i % 4],
    plain,
    before: plain ? PROMPTS[i % 3] : s[0],
    after: plain ? PROMPTS[i % 3] : s[1],
    x1: 1640 + (random(`gx${i}`) - 0.5) * 120,
    y1: 540 + (random(`gy${i}`) - 0.5) * 70,
  };
});
const IN = 30, HIDE = 8, OUT = 38;

export const GateFlow = () => {
  const f = useCurrentFrame();
  const flash = Math.max(0, ...CHIPS.map((c) => bump(f, c.start + IN, 14)));
  const masked = CHIPS.filter((c) => !c.plain && f >= c.start + IN + HIDE).length;
  const arrive = Math.max(0, ...CHIPS.map((c) => bump(f, c.start + IN + HIDE + OUT, 10)));
  return (
    <Bg color={C.night}>
      <Headline text="Secrets never leave." at={6} top={64} color="#ffffff" accentWords={[1]} size={88} />
      {AGENTS.map((y, i) => <AgentNode key={y} x={150} y={y} />)}
      <Gate x={GX} y={250} w={GW} h={640} at={0} flash={flash} />
      <div style={{position: 'absolute', left: 1240, top: 230, height: 760, borderLeft: `3px dashed ${C.nightLine}`}} />
      <Cloud x={1430} y={420} w={420} h={230} stroke={C.ok} glow={0.25 + arrive * 0.5} />
      <div style={{position: 'absolute', left: 1430, width: 420, top: 664, textAlign: 'center', fontSize: 26, color: '#c8d3e0'}}>LLM provider</div>
      {CHIPS.map((c, i) => {
        if (f < c.start) return null;
        const t1 = tw(f, c.start, c.start + IN, 0, 1, easeInOut);
        if (t1 < 1) return <Chip key={i} kind={c.plain ? 'plain' : 'secret'} x={330 + (GX - 330) * t1} y={c.y} style={{opacity: Math.min(1, t1 * 8)}}>{c.before}</Chip>;
        const s2 = c.start + IN + HIDE;
        if (f < s2) return null;
        const t2 = tw(f, s2, s2 + OUT, 0, 1, easeInOut);
        if (t2 >= 1) return null;
        const pop = sp(f, s2, {damping: 12, stiffness: 220});
        const x0 = GX + GW;
        return (
          <Chip key={i} kind={c.plain ? 'plain' : 'masked'} x={x0 + (c.x1 - x0) * t2} y={c.y + (c.y1 - c.y) * t2} style={{opacity: t2 > 0.85 ? (1 - t2) / 0.15 : 1, scale: `${0.6 + 0.4 * pop}`}}>
            {c.after}
          </Chip>
        );
      })}
      <Pop at={40} y={40} style={{position: 'absolute', left: 1440, top: 770, width: 400}}>
        <Metric label="Secrets masked" value={masked} valueSize={64}>
          <Status tone="info">placeholders sent instead</Status>
        </Metric>
      </Pop>
    </Bg>
  );
};

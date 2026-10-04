import {interpolateColors, random, useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {AgentNode, Bg, Chip, Cloud, Headline, Pop, bump, easeInOut, tw} from '../kit';

const AGENTS = [230, 390, 550, 710, 870];
const VALUES = ['Pr0d-Adm1n-2026', 'wJalrXUtnFEMI…', 'PL61 1090 1014…', 'help@payments…', 'sk_live_51Hx…'];
const DUR = 40;
const CHIPS = Array.from({length: 15}, (_, i) => {
  const y0 = AGENTS[(i * 3) % 5];
  const x1 = 1630 + (random(`x${i}`) - 0.5) * 140;
  const y1 = 540 + (random(`y${i}`) - 0.5) * 90;
  return {start: 22 + i * 7, y0, x1, y1, cy: Math.max(250, Math.min(y0, y1) - 70 - random(`c${i}`) * 110), v: VALUES[i % 5]};
});

export const Leak = () => {
  const f = useCurrentFrame();
  const arrived = CHIPS.filter((c) => f >= c.start + DUR);
  const red = Math.min(1, arrived.length / 8);
  const pulse = Math.max(0, ...CHIPS.map((c) => bump(f, c.start + DUR, 8)));
  return (
    <Bg color={C.night}>
      <Headline text="And send it out." at={8} top={64} color="#ffffff" accent={C.bad} accentWords={[3]} size={88} />
      {AGENTS.map((y, i) => <Pop key={y} at={2 + i * 3} from={0.3} y={0}><AgentNode x={190} y={y} /></Pop>)}
      <div style={{position: 'absolute', left: 1170, top: 200, height: 780, borderLeft: `3px dashed ${C.nightLine}`, opacity: tw(f, 6, 20)}} />
      <div style={{position: 'absolute', left: 900, width: 240, top: 990, textAlign: 'right', fontSize: 24, color: C.nightText}}>your company</div>
      <div style={{position: 'absolute', left: 1200, top: 990, fontSize: 24, color: C.nightText}}>the internet</div>
      <Pop at={6} from={0.6} y={0}>
        <Cloud x={1410} y={420} w={440} h={240} stroke={interpolateColors(red, [0, 1], [C.nightLine, C.bad])} glow={red * 0.6 + pulse * 0.4} />
        <div style={{position: 'absolute', left: 1410, width: 440, top: 680, textAlign: 'center', fontSize: 26, color: '#c8d3e0'}}>LLM provider</div>
      </Pop>
      {CHIPS.map((c, i) => {
        const t = tw(f, c.start, c.start + DUR, 0, 1, easeInOut);
        if (f < c.start || t >= 1) return null;
        const x0 = 330;
        const mx = (x0 + c.x1) / 2;
        const x = (1 - t) * (1 - t) * x0 + 2 * (1 - t) * t * mx + t * t * c.x1;
        const y = (1 - t) * (1 - t) * c.y0 + 2 * (1 - t) * t * c.cy + t * t * c.y1;
        return <Chip key={i} kind="secret" x={x} y={y} style={{opacity: t > 0.85 ? (1 - t) / 0.15 : Math.min(1, t * 8)}}>{c.v}</Chip>;
      })}
    </Bg>
  );
};

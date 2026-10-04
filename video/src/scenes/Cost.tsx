import {interpolate, useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {Appear, Mono, Title, ease} from '../ui';

const CAP = 200000;
const BENEFITS = [
  ['Approved models only', 'no surprise premium models on the bill'],
  ['No shadow AI spend', 'usage runs through the providers you chose'],
  ['0.3 ms overhead', 'no productivity cost for developers'],
  ['Self-hosted', 'one Go service and Redis, no per-seat fees'],
];

export const Cost = () => {
  const f = useCurrentFrame();
  const p = interpolate(f, [40, 200], [0, 1], {extrapolateLeft: 'clamp', extrapolateRight: 'clamp', easing: ease});
  const hit = f >= 200;
  const barColor = hit ? C.red : C.blue;
  return (
    <>
      <Title eyebrow="Result 2 · Cost">Spend stays predictable</Title>
      <div style={{position: 'absolute', left: 160, right: 160, top: 350}}>
        <Appear at={20} style={{display: 'flex', justifyContent: 'space-between', alignItems: 'baseline'}}>
          <div style={{fontSize: 32}}>An agent stuck in a loop <span style={{color: C.muted, fontStyle: 'italic'}}>(example)</span></div>
          <Mono color={barColor} size={32}>{Math.round(p * CAP).toLocaleString('en-US')} tokens</Mono>
        </Appear>
        <Appear at={30} style={{marginTop: 20, position: 'relative', height: 36, border: `2px solid ${C.dim}`, borderRadius: 4}}>
          <div style={{position: 'absolute', left: 0, top: 0, bottom: 0, width: `${p * 100}%`, background: barColor, opacity: .85}} />
        </Appear>
        <Appear at={30} style={{marginTop: 12, textAlign: 'right', fontSize: 26, color: C.muted}}>budget: 200,000 tokens</Appear>
        <Appear at={205} style={{marginTop: 10, fontSize: 34, color: C.red}}>Stopped at its budget, instead of running all weekend.</Appear>
      </div>
      <div style={{position: 'absolute', left: 160, right: 160, top: 690, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '40px 80px'}}>
        {BENEFITS.map(([n, d], i) => (
          <Appear key={n} at={280 + i * 25}>
            <div style={{fontSize: 38, color: C.green}}>{n}</div>
            <div style={{fontSize: 28, color: C.muted, marginTop: 6}}>{d}</div>
          </Appear>
        ))}
      </div>
    </>
  );
};

import {C} from '../theme';
import {Appear, Title} from '../ui';

const WHERE = [
  ['Laptops and CI', 'one setting'],
  ['Company network', 'automatic, through device management'],
  ['Servers and Kubernetes', 'the gate is the only way out'],
  ['Claude Code', 'works today, on a normal subscription'],
];

export const Rollout = () => (
  <>
    <Title eyebrow="Rollout">Nothing changes for developers</Title>
    <div style={{position: 'absolute', left: 160, right: 160, top: 400, display: 'flex', gap: 56}}>
      {WHERE.map(([n, d], i) => (
        <Appear key={n} at={30 + i * 20} style={{flex: 1, borderTop: `2px solid ${C.blue}`, paddingTop: 24}}>
          <div style={{fontSize: 38}}>{n}</div>
          <div style={{fontSize: 29, color: C.muted, marginTop: 10, lineHeight: 1.3}}>{d}</div>
        </Appear>
      ))}
    </div>
    <Appear at={150} style={{position: 'absolute', left: 160, right: 160, bottom: 120, fontSize: 36, fontStyle: 'italic', color: C.muted}}>
      57 automated tests · runs on your own infrastructure
    </Appear>
  </>
);

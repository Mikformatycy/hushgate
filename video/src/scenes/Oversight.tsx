import {C, MONO} from '../theme';
import {Appear, Title} from '../ui';

const LOG: [string, string, string, string, string][] = [
  ['04:48:12', 'demo-agent', 'masked', 'DB_PASSWORD sent as a placeholder', C.yellow],
  ['04:48:21', 'demo-agent', 'allowed', 'write_file, local tool', C.green],
  ['04:48:29', 'demo-agent', 'warned', 'prompt injection in read_file, 99%', C.yellow],
  ['04:48:46', 'demo-agent', 'halted', 'send_email carried a secret', C.red],
  ['04:50:25', 'guest-laptop', 'blocked', 'shadow AI at llm.sketchy-vps.example', C.red],
];
const POINTS = [
  'Export to CSV or JSON for auditors',
  'Live dashboard and Prometheus metrics',
  'AI suggests new rules; a person approves them',
  'A plain policy file you can keep in git',
];

export const Oversight = () => (
  <>
    <Title eyebrow="Result 3 · Oversight">Every AI action, on record</Title>
    <div style={{position: 'absolute', left: 160, right: 160, top: 340, fontFamily: MONO, fontSize: 27, display: 'flex', flexDirection: 'column', gap: 16}}>
      {LOG.map(([t, who, what, why, c], i) => (
        <Appear key={t} at={30 + i * 22} y={8} style={{display: 'flex', gap: 36}}>
          <span style={{color: C.muted, width: 150}}>{t}</span>
          <span style={{width: 250}}>{who}</span>
          <span style={{color: c, width: 160}}>{what}</span>
          <span style={{color: C.muted}}>{why}</span>
        </Appear>
      ))}
    </div>
    <div style={{position: 'absolute', left: 160, right: 160, top: 720, display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '30px 80px', fontSize: 36}}>
      {POINTS.map((p, i) => (
        <Appear key={p} at={170 + i * 20} style={{borderLeft: `3px solid ${C.blue}`, paddingLeft: 24}}>{p}</Appear>
      ))}
    </div>
  </>
);

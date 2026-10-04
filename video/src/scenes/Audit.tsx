import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Card, Code, Cursor, Headline, Panel, Pop, Row, Status, Tilt, bump, easeInOut, tw} from '../kit';
import type {Tone} from '../kit';

const COLS = '150px 250px 300px 1fr';
const EVENTS: [string, string, Tone, string, string][] = [
  ['04:48:12', 'demo-agent', 'info', 'Masked', '2 secrets replaced in outbound request'],
  ['04:48:15', 'demo-agent', 'ok', 'Allowed', 'write_file, local tool'],
  ['04:48:21', 'claude-code', 'info', 'AI suggestion', 'post_to_slack → network'],
  ['04:48:29', 'demo-agent', 'warn', 'Prompt injection', 'hidden instructions in read_file, 99%'],
  ['04:48:46', 'demo-agent', 'bad', 'Killed', 'send_email carried a secret'],
  ['04:49:02', 'claude-code', 'bad', 'Signature: block', 'HG-RCE-001, script piped into a shell'],
  ['04:50:25', 'guest-laptop', 'bad', 'Shadow AI', 'llm.sketchy-vps.example'],
  ['04:51:10', 'security-team', 'info', 'Policy reloaded', 'hushgate.yaml: Bash → deny'],
  ['04:51:40', 'ci-runner-01', 'bad', 'Model blocked', 'gpt-4o is not on the allowlist'],
];
const STEP = 11, ROW = 74;
const BTN: [number, number] = [1381, 268];
const CLICK = 162;

export const Audit = () => {
  const f = useCurrentFrame();
  const t = (e: number) => 8 + e * STEP;
  const move = tw(f, 124, CLICK - 2, 0, 1, easeInOut);
  const press = bump(f, CLICK, 8);
  return (
    <Bg color={C.page}>
      <Headline text="Every action, on record." at={6} top={80} size={88} accent={C.link} accentWords={[2, 3]} />
      <Tilt at={2} rx={-18} ry={12} rz={-1} style={{position: 'absolute', left: 200, top: 230, width: 1520}}>
        <Panel
          title="Audit log"
          actions={
            <div style={{display: 'flex', gap: 14}}>
              {['Export CSV', 'Export JSON'].map((b, i) => (
                <div key={b} style={{fontSize: 22, fontWeight: 700, color: C.link, border: `2px solid ${C.link}`, borderRadius: 999, padding: '7px 22px', background: i === 0 ? `rgba(0,115,187,${0.15 * press})` : '#fff', transform: `scale(${i === 0 ? 1 - press * 0.06 : 1})`}}>{b}</div>
              ))}
            </div>
          }
        >
          <Row cols={COLS} head><span>Time</span><span>Agent</span><span>Event</span><span>Details</span></Row>
          <div style={{position: 'relative', height: ROW * 9}}>
            {EVENTS.map(([time, agent, tone, kind, detail], e) => {
              if (f < t(e)) return null;
              const below = EVENTS.reduce((s, _, j) => (j > e ? s + tw(f, t(j), t(j) + 8, 0, 1, easeInOut) : s), 0);
              const fresh = 1 - tw(f, t(e), t(e) + 30);
              return (
                <div key={time} style={{position: 'absolute', left: 0, right: 0, top: below * ROW, opacity: tw(f, t(e), t(e) + 6)}}>
                  <Row cols={COLS} style={{height: ROW, background: `rgba(0,115,187,${0.1 * fresh})`}}>
                    <span style={{fontFamily: MONO, fontSize: 21, color: C.sub}}>{time}</span>
                    <span style={{fontFamily: MONO, fontSize: 21}}>{agent}</span>
                    <Status tone={tone}>{kind}</Status>
                    <span style={{color: '#414750', whiteSpace: 'nowrap', overflow: 'hidden'}}>{detail}</span>
                  </Row>
                </div>
              );
            })}
          </div>
        </Panel>
      </Tilt>
      {f >= 120 && <Cursor x={1250 + (BTN[0] - 1250) * move} y={900 + (BTN[1] - 900) * move} press={press} />}
      {f >= CLICK && (
        <div style={{position: 'absolute', left: BTN[0] - 60, top: BTN[1] - 60, width: 120, height: 120, borderRadius: 60, border: `3px solid ${C.link}`, opacity: 1 - tw(f, CLICK, CLICK + 16), transform: `scale(${0.3 + tw(f, CLICK, CLICK + 16)})`}} />
      )}
      <Pop at={CLICK + 8} from={0.5} y={-60} style={{position: 'absolute', left: 1330, top: 330, width: 420}} cfg={{damping: 12}}>
        <Card style={{padding: '22px 26px', display: 'flex', alignItems: 'center', gap: 20, boxShadow: '0 20px 50px rgba(0,28,36,.25)'}}>
          <svg viewBox="0 0 24 24" width={58} height={58} fill="none" stroke={C.ok} strokeWidth={1.8} strokeLinejoin="round"><path d="M6 2h8l5 5v15H6z" /><path d="M14 2v5h5" /><path d="M12.5 10v7M9.5 14l3 3 3-3" /></svg>
          <div>
            <Code size={22}>hushgate-audit.csv</Code>
            <div style={{marginTop: 8}}><Status tone="ok" size={22}>Ready for the auditors</Status></div>
          </div>
        </Card>
      </Pop>
    </Bg>
  );
};

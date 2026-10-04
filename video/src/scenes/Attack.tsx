import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Camera, Headline, Metric, ShieldIcon, Status, Tilt, bump, tw} from '../kit';

const NAV = ['Dashboard', 'Agents', 'Nodes', 'Review', 'Audit log', 'Vault', 'Signatures', 'Policy'];
const KILL = 108;

const Banner: React.FC<{at: number; h: number; children: React.ReactNode; style: React.CSSProperties}> = ({at, h, children, style}) => {
  const p = tw(useCurrentFrame(), at, at + 10);
  return (
    <div style={{height: p * h, marginBottom: p * 18, opacity: p, overflow: 'hidden'}}>
      <div style={{borderRadius: 12, padding: '18px 26px', fontSize: 23, lineHeight: 1.45, display: 'flex', gap: 16, ...style}}>{children}</div>
    </div>
  );
};

export const Attack = () => {
  const f = useCurrentFrame();
  const killed = f >= KILL;
  const glow = f >= KILL && f < KILL + 48 ? Math.abs(Math.sin(((f - KILL) / 12) * Math.PI)) * 12 * (1 - (f - KILL) / 48) : 0;
  const swap = (a: React.ReactNode, b: React.ReactNode) => <div style={{position: 'relative'}}><div style={{opacity: 1 - tw(f, KILL, KILL + 6)}}>{a}</div><div style={{position: 'absolute', top: 0, opacity: tw(f, KILL, KILL + 6)}}>{b}</div></div>;
  return (
    <Bg color={C.page}>
      <Camera keys={[[0, 1, 0, 0], [150, 1, 0, 0], [196, 1.42, 0, 60], [270, 1.48, 0, 60]]} origin="1090px 470px">
        <Tilt at={0} rx={26} ry={-12} rz={2} s={0.86} style={{position: 'absolute', left: 160, top: 196, width: 1600, height: 860}}>
          <div style={{width: '100%', height: '100%', borderRadius: 14, overflow: 'hidden', background: C.page, boxShadow: '0 30px 90px rgba(0,28,36,.25)', border: `1px solid ${C.line}`}}>
            <div style={{height: 60, background: C.nav, display: 'flex', alignItems: 'center', gap: 14, padding: '0 26px', color: '#fff'}}>
              <ShieldIcon size={30} />
              <b style={{fontSize: 24}}>HushGate</b>
              <span style={{fontSize: 22, color: '#9ca3af'}}>AI Control Layer</span>
              <span style={{marginLeft: 'auto', fontSize: 18, color: '#d1d5db', display: 'flex', alignItems: 'center', gap: 8}}><span style={{width: 10, height: 10, borderRadius: 5, background: C.ok}} />Live</span>
            </div>
            <div style={{display: 'flex', height: 800}}>
              <div style={{width: 270, background: '#fff', borderRight: `1px solid ${C.line}`, paddingTop: 22}}>
                <div style={{padding: '0 26px 10px', fontSize: 16, fontWeight: 700, letterSpacing: 1, color: C.sub}}>MONITORING</div>
                {NAV.map((n, i) => (
                  <div key={n} style={{padding: '9px 26px', fontSize: 22, color: i === 0 ? C.link : '#414750', fontWeight: i === 0 ? 700 : 400, borderLeft: i === 0 ? `5px solid ${C.link}` : '5px solid transparent'}}>{n}</div>
                ))}
              </div>
              <div style={{flex: 1, padding: '30px 36px'}}>
                <Banner at={40} h={126} style={{border: `3px solid ${C.orange}`, background: C.warnBg}}>
                  <span style={{color: C.warnInk, fontSize: 28}}>⚠</span>
                  <div><b>Prompt injection in read_file</b> <span style={{fontFamily: MONO, fontSize: 20, background: '#f6ecd2', borderRadius: 5, padding: '1px 8px'}}>quarterly_report.md</span><br />Hidden instructions for an AI (99%). The agent is the target of this attack, not its source.</div>
                </Banner>
                <Banner at={KILL} h={126} style={{background: C.bad, color: '#fff', boxShadow: `0 0 0 ${glow}px rgba(209,50,18,.35)`}}>
                  <span style={{fontSize: 28}}>⊗</span>
                  <div><b>Kill switch fired for agent "demo-agent"</b><br />send_email tried to send {'{{VAULT_ENV_DB_PASSWORD}}'} off the machine. The call was dropped and the agent is halted.</div>
                </Banner>
                <div style={{fontSize: 20, color: C.sub}}>HushGate › Dashboard</div>
                <div style={{fontSize: 40, fontWeight: 700, margin: '4px 0 22px'}}>Dashboard</div>
                <div style={{display: 'grid', gridTemplateColumns: '1fr 1fr 1fr', gap: 24}}>
                  <Metric label="Agents" value="1">
                    {swap(<Status tone="ok">1 running</Status>, <Status tone="bad">1 halted</Status>)}
                  </Metric>
                  <Metric label="Tool calls" value={killed ? '7' : '6'}>
                    <Status tone="ok">3 allowed</Status>
                    <Status tone="warn">3 blocked</Status>
                    <div style={{opacity: tw(f, KILL, KILL + 8)}}><Status tone="bad">1 killed</Status></div>
                  </Metric>
                  <Metric label="Secrets masked" value="4">
                    <Status tone="info">placeholders sent instead</Status>
                  </Metric>
                </div>
              </div>
            </div>
          </div>
        </Tilt>
      </Camera>
      <div style={{position: 'absolute', left: 0, right: 0, top: 0, height: 250, background: `linear-gradient(${C.page} 62%, rgba(242,243,243,0))`}} />
      <Headline text="Hijacked agent?" at={8} out={120} top={60} size={92} accent={C.orange} accentWords={[1]} />
      <Headline text="Stopped in 0.1 ms." at={134} top={60} size={92} accent={C.bad} accentWords={[0]} />
      <div style={{opacity: bump(f, KILL, 6) * 0.18, position: 'absolute', inset: 0, background: C.bad}} />
    </Bg>
  );
};

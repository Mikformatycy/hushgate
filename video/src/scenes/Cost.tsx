import {Easing, useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Card, Headline, Meter, Metric, Panel, Pop, Row, Status, Tilt, bump, tw} from '../kit';

const COLS = '440px 360px 1fr';
const CAP = 200000, HIT = 140;

export const Cost = () => {
  const f = useCurrentFrame();
  const loop = tw(f, 26, HIT, 0, CAP, Easing.in(Easing.cubic));
  const halted = f >= HIT;
  const rows: [string, number, number, boolean][] = [
    ['claude-code-sandbox', 1240000 + f * 950, 5000000, false],
    ['ci-runner-01', 38000 + f * 130, 200000, false],
    ['nightly-refactor', loop, CAP, halted],
  ];
  return (
    <Bg color={C.page}>
      <Headline text="Spend, capped." at={6} top={86} size={96} accent={C.link} accentWords={[1]} />
      <Tilt at={2} rx={22} ry={-14} rz={2} style={{position: 'absolute', left: 200, top: 250, width: 1520}}>
        <Panel title="Agents (3)">
          <Row cols={COLS} head><span>Agent</span><span>Status</span><span>Token budget</span></Row>
          {rows.map(([name, used, limit, stop]) => (
            <Row key={name} cols={COLS} style={{height: 96, background: stop ? `rgba(209,50,18,${0.05 + 0.12 * bump(f, HIT, 24)})` : undefined}}>
              <span style={{fontFamily: MONO, fontSize: 24, fontWeight: 600}}>{name}</span>
              <span>{stop ? <Status tone="bad">Budget reached · halted</Status> : <Status tone="ok">Running</Status>}</span>
              <Meter used={used} limit={limit} />
            </Row>
          ))}
        </Panel>
      </Tilt>
      <Pop at={162} y={50} style={{position: 'absolute', left: 200, top: 690, width: 740}}>
        <Card style={{padding: '24px 32px'}}>
          <div style={{fontSize: 24, color: C.sub, marginBottom: 12}}>Models</div>
          {([['claude-*', 'ok', 'Allowed', 168], ['gpt-4o', 'bad', 'Model blocked', 182]] as const).map(([m, tone, label, at]) => (
            <Pop key={m} at={at} from={0.9} y={10} style={{display: 'flex', justifyContent: 'space-between', alignItems: 'center', height: 58, fontSize: 26}}>
              <span style={{fontFamily: MONO, fontWeight: 600}}>{m}</span>
              <Status tone={tone}>{label}</Status>
            </Pop>
          ))}
        </Card>
      </Pop>
      <Pop at={196} y={50} style={{position: 'absolute', left: 980, top: 690, width: 740}}>
        <Metric label="Gate overhead (p50)" value="335 µs">
          <Status tone="ok">No slowdown for developers</Status>
        </Metric>
      </Pop>
    </Bg>
  );
};

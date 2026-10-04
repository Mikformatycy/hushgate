import {useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {Bg, Code, Headline, Panel, Pop, Row, Status, Tilt, bump, sp} from '../kit';

const COLS = '210px 1fr 170px 150px 90px';
const SIGS: {id: string; name: string; sev: 'critical' | 'high'; act: string; hits: number[]}[] = [
  {id: 'HG-RCE-001', name: 'Remote script piped into a shell', sev: 'critical', act: 'block', hits: [38, 72, 104]},
  {id: 'HG-CRED-002', name: 'Environment or .env piped to the network', sev: 'critical', act: 'kill', hits: [54]},
  {id: 'HG-CRED-001', name: 'SSH keys or cloud credentials accessed', sev: 'high', act: 'block', hits: [66, 114]},
  {id: 'HG-SUP-001', name: 'Remote code from a model repository', sev: 'high', act: 'block', hits: [84]},
  {id: 'HG-DES-001', name: 'Recursive delete of root or home', sev: 'high', act: 'block', hits: [96]},
  {id: 'HG-SUP-005', name: 'Known typosquatted package', sev: 'high', act: 'block', hits: [122]},
];

export const Signatures = () => {
  const f = useCurrentFrame();
  return (
    <Bg color={C.page}>
      <Headline text="Known attacks, blocked." at={6} top={86} size={92} accent={C.bad} accentWords={[2]} />
      <Tilt at={4} rx={20} ry={14} rz={-2} style={{position: 'absolute', left: 230, top: 270, width: 1460}}>
        <Panel title="Signatures (19)" actions={<Status tone="ok" size={22}>Live feed · loaded</Status>}>
          <Row cols={COLS} head><span>ID</span><span>Signature</span><span>Severity</span><span>Action</span><span>Hits</span></Row>
          {SIGS.map((s, i) => {
            const hits = s.hits.filter((h) => f >= h);
            const flash = Math.max(0, ...s.hits.map((h) => bump(f, h, 20)));
            const last = hits.length ? hits[hits.length - 1] : 0;
            const n = hits.length ? sp(f, last, {damping: 9, stiffness: 260}) : 1;
            return (
              <Pop key={s.id} at={12 + i * 4} from={1} y={16}>
                <Row cols={COLS} style={{background: `rgba(209,50,18,${0.12 * flash})`}}>
                  <span><Code>{s.id}</Code></span>
                  <span style={{fontWeight: 600}}>{s.name}</span>
                  <span><Status tone={s.sev === 'critical' ? 'bad' : 'warn'}>{s.sev}</Status></span>
                  <span><Status tone="bad">{s.act}</Status></span>
                  <span style={{fontWeight: 700, fontSize: 30, color: hits.length ? C.bad : C.sub, display: 'inline-block', transform: `scale(${hits.length ? 2 - n : 1})`}}>{hits.length}</span>
                </Row>
              </Pop>
            );
          })}
        </Panel>
      </Tilt>
    </Bg>
  );
};

import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Headline, Pop, Tilt, tw} from '../kit';

const L1 = '$ export ANTHROPIC_BASE_URL=https://hushgate.corp.internal';
const L2 = '$ claude';
const WHERE = ['Claude Code', 'CI pipelines', 'Company laptops', 'Kubernetes'];

export const DropIn = () => {
  const f = useCurrentFrame();
  const n1 = Math.round(tw(f, 14, 54, 0, L1.length, (x) => x));
  const n2 = Math.round(tw(f, 62, 70, 0, L2.length, (x) => x));
  const caret = Math.floor(f / 8) % 2 === 0;
  const cur = (on: boolean) => (on && caret ? <span style={{background: '#c8d3e0', color: C.night}}>&nbsp;</span> : null);
  return (
    <Bg color={C.night}>
      <Headline text="One line. No code changes." at={6} top={90} color="#ffffff" accentWords={[0, 1]} size={88} />
      <Tilt at={0} rx={24} ry={18} rz={-3} style={{position: 'absolute', left: 280, top: 290, width: 1360}}>
        <div style={{borderRadius: 16, overflow: 'hidden', background: '#121b27', border: `1px solid ${C.nightLine}`, boxShadow: '0 40px 120px rgba(0,0,0,.55)'}}>
          <div style={{height: 50, background: C.nav, display: 'flex', alignItems: 'center', gap: 10, padding: '0 20px'}}>
            {['#ff5f57', '#febc2e', '#28c840'].map((c) => <span key={c} style={{width: 14, height: 14, borderRadius: 7, background: c, opacity: 0.85}} />)}
          </div>
          <div style={{padding: '30px 40px 36px', fontFamily: MONO, fontSize: 31, lineHeight: 1.75, color: '#e6edf5', minHeight: 230}}>
            <div>{L1.slice(0, n1)}{cur(f < 58)}</div>
            {f >= 58 && <div>{L2.slice(0, n2)}{cur(f >= 58 && f < 80)}</div>}
            {f >= 80 && <Pop at={80} from={0.9} y={10} style={{color: '#7ee07a'}}>✓ Connected through HushGate</Pop>}
          </div>
        </div>
      </Tilt>
      <div style={{position: 'absolute', left: 0, right: 0, top: 760, display: 'flex', justifyContent: 'center', gap: 26}}>
        {WHERE.map((w, i) => (
          <Pop key={w} at={96 + i * 6} from={0.5} y={30}>
            <div style={{display: 'flex', alignItems: 'center', gap: 14, padding: '16px 30px', borderRadius: 999, background: C.nightNode, border: `2px solid ${C.nightLine}`, color: '#fff', fontSize: 30, fontWeight: 600}}>
              <span style={{width: 12, height: 12, borderRadius: 6, background: C.orange}} />{w}
            </div>
          </Pop>
        ))}
      </div>
    </Bg>
  );
};

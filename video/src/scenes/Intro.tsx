import {random, useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Headline, Tilt, tw} from '../kit';

const FILES = ['src/', '  api.ts', '  billing.ts', '  auth.ts', 'config/', '  .env', 'tests/', 'README.md'];
const HUES = ['#2f4a66', '#41607f', '#5b4d7a', '#2f5f4f', '#6b5a2e', '#3c4f63'];
const LINE_H = 36;
const LINES = Array.from({length: 24}, (_, i) => {
  const indent = Math.floor(random(`in${i}`) * 4) * 34;
  const segs = Array.from({length: 1 + Math.floor(random(`n${i}`) * 4)}, (_, j) => ({
    w: 50 + random(`w${i}-${j}`) * 190,
    c: HUES[Math.floor(random(`c${i}-${j}`) * HUES.length)],
  }));
  return {indent, segs};
});

export const Intro = () => {
  const f = useCurrentFrame();
  const env = tw(f, 92, 104);
  const scroll = (f * 5) % (LINE_H * LINES.length);
  return (
    <Bg color={C.night}>
      <Headline text="Your agents read everything." at={6} top={96} color="#ffffff" accentWords={[3]} size={92} />
      <Tilt at={0} rx={32} ry={-24} rz={5} s={0.78} style={{position: 'absolute', left: 360, top: 290, width: 1200, height: 660}}>
        <div style={{width: '100%', height: '100%', borderRadius: 16, overflow: 'hidden', background: '#121b27', border: `1px solid ${C.nightLine}`, boxShadow: '0 40px 120px rgba(0,0,0,.55)'}}>
          <div style={{height: 52, background: C.nav, display: 'flex', alignItems: 'center', gap: 10, padding: '0 20px'}}>
            {['#ff5f57', '#febc2e', '#28c840'].map((c) => <span key={c} style={{width: 14, height: 14, borderRadius: 7, background: c, opacity: 0.85}} />)}
            <span style={{marginLeft: 16, fontFamily: MONO, fontSize: 20, color: C.nightText}}>agent · payments-api</span>
          </div>
          <div style={{display: 'flex', height: 608}}>
            <div style={{width: 270, background: '#0d1520', padding: '18px 0', fontFamily: MONO, fontSize: 21, color: '#7f8c9c'}}>
              {FILES.map((n) => {
                const isEnv = n.trim() === '.env';
                return (
                  <div key={n} style={{padding: '7px 22px', whiteSpace: 'pre', background: isEnv ? `rgba(255,153,0,${0.22 * env})` : undefined, color: isEnv && env > 0.4 ? C.orange : undefined}}>{n}</div>
                );
              })}
            </div>
            <div style={{flex: 1, position: 'relative', overflow: 'hidden'}}>
              <div style={{position: 'absolute', left: 36, right: 36, top: 20 - scroll}}>
                {[...LINES, ...LINES].map((l, i) => (
                  <div key={i} style={{height: LINE_H, display: 'flex', alignItems: 'center', gap: 12, paddingLeft: l.indent}}>
                    {l.segs.map((s, j) => <span key={j} style={{width: s.w, height: 12, borderRadius: 6, background: s.c}} />)}
                  </div>
                ))}
              </div>
              <div style={{position: 'absolute', left: 0, right: 0, top: 150 + Math.sin(f / 9) * 120, height: 44, background: 'rgba(0,115,187,.22)', borderLeft: `4px solid ${C.link}`}} />
            </div>
          </div>
        </div>
      </Tilt>
    </Bg>
  );
};

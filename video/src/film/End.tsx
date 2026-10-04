import {AbsoluteFill} from 'remotion';
import {C, MONO} from '../theme';
import {Caption, Shield, kf, sp, tw} from '../kit';
import {T} from './layout';

export const End: React.FC<{f: number}> = ({f}) => {
  const at = T.end + 30;
  if (f < at - 6) return null;
  const s = sp(f, at, {damping: 11, stiffness: 150, mass: 0.8});
  const slide = kf(f, [[at + 20, 1], [at + 40, 0]]);
  const text = tw(f, at + 24, at + 42);
  return (
    <>
      <AbsoluteFill style={{alignItems: 'center', justifyContent: 'center', paddingBottom: 190}}>
        <div style={{position: 'relative', display: 'flex', alignItems: 'center', gap: 44}}>
          {[0, 9].map((d) => {
            const r = tw(f, at + 4 + d, at + 34 + d);
            return f >= at + 4 + d && r < 1 ? <div key={d} style={{position: 'absolute', left: 90 + slide * 369 - 90, top: '50%', width: 180, height: 180, marginTop: -90, borderRadius: 90, border: `3px solid ${C.gs}`, opacity: (1 - r) * 0.8, transform: `scale(${0.6 + r * 2.2})`}} /> : null;
          })}
          <div style={{transform: `translateX(${slide * 369}px) scale(${s})`}}><Shield size={180} /></div>
          <div style={{fontSize: 140, fontWeight: 800, letterSpacing: '-0.04em', color: '#FFFFFF', lineHeight: 1, opacity: text, transform: `translateX(${(1 - text) * -40}px)`}}>HushGate</div>
        </div>
      </AbsoluteFill>
      <Caption f={f} cues={[[at + 46, T.total + 40, 'Use AI. *Keep your secrets.*', C.gs]]} color="#FFFFFF" top={600} />
      <div style={{position: 'absolute', left: 0, right: 0, top: 740, textAlign: 'center', fontSize: 28, fontWeight: 600, color: C.navyText, opacity: tw(f, at + 62, at + 76)}}>
        57 tests · 19 attack signatures · 0.3 ms overhead
      </div>
      <div style={{position: 'absolute', left: 0, right: 0, top: 880, textAlign: 'center', fontSize: 22, color: C.navyText, opacity: tw(f, at + 74, at + 88)}}>
        Team Mikformatyka · HackYeah 2026
        <div style={{fontFamily: MONO, fontSize: 19, marginTop: 8}}>github.com/Mikformatycy/hushgate</div>
      </div>
    </>
  );
};

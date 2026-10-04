import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Headline, Pop, TierBadge, Tilt, tw} from '../kit';

const ENV: [string, string, string][] = [
  ['DB_PASSWORD', 'Pr0d-Adm1n-2026', 'C3'],
  ['AWS_SECRET_ACCESS_KEY', 'wJalrXUtnFEMI/K7MDENG', 'C3'],
  ['CUSTOMER_IBAN', 'PL61 1090 1014 0000 0712', 'C2'],
  ['SUPPORT_EMAIL', 'help@payments.example', 'C1'],
];
const HI: Record<string, string> = {C3: '209,50,18', C2: '255,153,0', C1: '0,115,187'};

export const EnvFile = () => {
  const f = useCurrentFrame();
  return (
    <Bg color={C.night}>
      <Headline text="Including this." at={4} top={120} color="#ffffff" accentWords={[1]} size={92} />
      <Tilt at={0} rx={-14} ry={14} rz={-2} s={0.9} style={{position: 'absolute', left: 290, top: 320, width: 1340}}>
        <div style={{borderRadius: 16, overflow: 'hidden', background: '#121b27', border: `1px solid ${C.nightLine}`, boxShadow: '0 40px 120px rgba(0,0,0,.55)'}}>
          <div style={{height: 52, background: C.nav, display: 'flex', alignItems: 'center', padding: '0 24px', fontFamily: MONO, fontSize: 22, color: C.orange}}>.env</div>
          <div style={{padding: '22px 36px 28px'}}>
            {ENV.map(([k, v, tier], i) => {
              const h = tw(f, 44 + i * 14, 56 + i * 14);
              return (
                <Pop key={k} at={10 + i * 7} from={1} y={18} style={{height: 92, display: 'flex', alignItems: 'center', justifyContent: 'space-between', fontFamily: MONO, fontSize: 34}}>
                  <span>
                    <span style={{color: C.nightText}}>{k}=</span>
                    <span style={{color: '#ffffff', background: `rgba(${HI[tier]},${0.38 * h})`, borderRadius: 6, padding: '2px 8px', boxShadow: `inset 0 -3px 0 rgba(${HI[tier]},${h})`}}>{v}</span>
                  </span>
                  <Pop at={46 + i * 14} from={0.4} y={0}><TierBadge tier={tier} /></Pop>
                </Pop>
              );
            })}
          </div>
        </div>
      </Tilt>
    </Bg>
  );
};

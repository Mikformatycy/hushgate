import {useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {Bg, Card, Chip, Headline, Pop, Status, Tilt, sp} from '../kit';

const SEEN = ['{{VAULT_ENV_DB_PASSWORD}}', '{{VAULT_ENV_AWS_SECRET}}', '{{VAULT_IBAN_7f3a91}}'];

export const ZeroLeaks = () => {
  const f = useCurrentFrame();
  const zero = sp(f, 26, {damping: 9, stiffness: 140});
  return (
    <Bg color={C.page}>
      <Headline text="Zero leaks." at={6} top={110} size={124} accent={C.ok} accentWords={[0]} />
      <Tilt at={10} rx={18} ry={16} rz={-2} style={{position: 'absolute', left: 460, top: 330, width: 1000}}>
        <Card style={{padding: '34px 44px 38px'}}>
          <div style={{fontSize: 30, color: C.sub}}>Plaintext secrets sent to model providers</div>
          <div style={{fontSize: 230, fontWeight: 300, color: C.link, lineHeight: 1.05, transform: `scale(${zero})`, transformOrigin: '0 60%'}}>0</div>
          <Pop at={44} from={0.9} y={10}><Status tone="ok" size={30}>Every secret replaced before it left the company</Status></Pop>
        </Card>
      </Tilt>
      <div style={{position: 'absolute', left: 0, right: 0, top: 830, display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 22}}>
        <Pop at={60} from={1} y={10} style={{fontSize: 26, color: C.sub, marginRight: 8}}>The model saw</Pop>
        {SEEN.map((s, i) => <Pop key={s} at={66 + i * 6} from={0.5} y={0}><Chip kind="masked" inline>{s}</Chip></Pop>)}
      </div>
      <div style={{position: 'absolute', left: 0, right: 0, bottom: 44, textAlign: 'center', fontSize: 20, color: C.sub}}>Across HushGate's test scenarios</div>
    </Bg>
  );
};

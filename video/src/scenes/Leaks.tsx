import {C} from '../theme';
import {Appear, Footnote, Mono, Title} from '../ui';

const STATS = [
  {big: '0', c: C.green, t: 'plaintext secrets sent to the model in our scenarios'},
  {big: '0.1 ms', c: C.green, t: 'to block an exfiltration attempt and halt the agent'},
  {big: '4 tiers', c: C.yellow, t: 'of sensitivity, including personal data such as IBANs and card numbers'},
];

export const Leaks = () => (
  <>
    <Title eyebrow="Result 1 · Data protection">Secrets stay inside the company</Title>
    <div style={{position: 'absolute', left: 160, right: 160, top: 340, display: 'flex', flexDirection: 'column', gap: 26}}>
      <Appear at={40} style={{fontSize: 30, fontStyle: 'italic', color: C.muted}}>What the model provider receives</Appear>
      <Appear at={60} style={{display: 'flex', alignItems: 'baseline', gap: 40}}>
        <div style={{width: 330, fontSize: 34, color: C.muted}}>Without HushGate</div>
        <Mono color={C.red} size={34}>DB_PASSWORD=Pr0d-Adm1n-2026</Mono>
      </Appear>
      <Appear at={120} style={{display: 'flex', alignItems: 'baseline', gap: 40}}>
        <div style={{width: 330, fontSize: 34}}>With HushGate</div>
        <Mono color={C.yellow} size={34}>{'DB_PASSWORD={{VAULT_ENV_DB_PASSWORD}}'}</Mono>
      </Appear>
      <Appear at={185} style={{fontSize: 32, color: C.green}}>
        The agent's own tools still get the real value, so the work gets done.
      </Appear>
    </div>
    <div style={{position: 'absolute', left: 160, right: 160, top: 690, display: 'flex', gap: 80}}>
      {STATS.map((s, i) => (
        <Appear key={s.big} at={270 + i * 40} style={{flex: 1}}>
          <div style={{fontSize: 92, color: s.c, lineHeight: 1}}>{s.big}</div>
          <div style={{fontSize: 30, color: C.muted, marginTop: 14, lineHeight: 1.3}}>{s.t}</div>
        </Appear>
      ))}
    </div>
    <Footnote at={400}>Measured in HushGate's test suite and demo runs. The password shown is an example.</Footnote>
  </>
);

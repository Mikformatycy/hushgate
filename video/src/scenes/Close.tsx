import {AbsoluteFill} from 'remotion';
import {C} from '../theme';
import {Appear} from '../ui';

export const Close = () => (
  <AbsoluteFill style={{alignItems: 'center', justifyContent: 'center', gap: 28, textAlign: 'center'}}>
    <Appear at={10} style={{fontSize: 128, fontWeight: 600, color: C.blue}}>HushGate</Appear>
    <div style={{display: 'flex', gap: 56, fontSize: 46, marginTop: 10}}>
      <Appear at={45} style={{color: C.yellow}}>Secrets stay inside.</Appear>
      <Appear at={70} style={{color: C.red}}>Attacks are stopped.</Appear>
      <Appear at={95} style={{color: C.green}}>Spend is capped.</Appear>
    </div>
    <Appear at={140} style={{fontSize: 30, color: C.muted, fontStyle: 'italic', marginTop: 40}}>Team Mikformatyka · HackYeah 2026</Appear>
  </AbsoluteFill>
);

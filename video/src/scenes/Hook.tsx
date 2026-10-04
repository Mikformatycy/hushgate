import {AbsoluteFill} from 'remotion';
import {C} from '../theme';
import {Appear} from '../ui';

export const Hook = () => (
  <AbsoluteFill style={{justifyContent: 'center', padding: '0 220px', gap: 36, fontSize: 64, lineHeight: 1.25}}>
    <Appear at={15}>Your engineers already work with AI agents.</Appear>
    <Appear at={85}>Those agents read your code, your configs and your secrets.</Appear>
    <Appear at={165} style={{color: C.yellow}}>Then they send it all to someone else's servers.</Appear>
  </AbsoluteFill>
);

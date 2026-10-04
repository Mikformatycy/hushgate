import {AbsoluteFill} from 'remotion';
import {C} from '../theme';
import {Bg, Wordmark} from '../kit';

export const Reveal = () => (
  <Bg color={C.nav}>
    <AbsoluteFill style={{alignItems: 'center', justifyContent: 'center'}}>
      <Wordmark at={4} />
    </AbsoluteFill>
  </Bg>
);

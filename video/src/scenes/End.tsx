import {AbsoluteFill} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Headline, Pop, Wordmark} from '../kit';

export const End = () => (
  <Bg color={C.nav}>
    <AbsoluteFill style={{alignItems: 'center', justifyContent: 'center', paddingBottom: 170}}>
      <Wordmark at={4} scale={0.9} />
    </AbsoluteFill>
    <Headline text="Use AI. Keep your secrets." at={66} top={640} size={68} color="#ffffff" accentWords={[2, 3, 4]} />
    <Pop at={104} from={1} y={14} style={{position: 'absolute', left: 0, right: 0, top: 860, textAlign: 'center', fontSize: 28, color: '#9ca3af'}}>
      Team Mikformatyka · HackYeah 2026
      <div style={{fontFamily: MONO, fontSize: 24, marginTop: 10}}>github.com/Mikformatycy/hushgate</div>
    </Pop>
  </Bg>
);

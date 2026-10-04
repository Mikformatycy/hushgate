import {AbsoluteFill, useCurrentFrame} from 'remotion';
import {C, SANS} from './theme';
import {Caption, kf} from './kit';
import {interpolateColors} from 'remotion';
import type {Cue} from './kit';
import {GATE_X, T} from './film/layout';
import {Flow} from './film/Flow';
import {Dash} from './film/Dash';
import {End} from './film/End';

export const TOTAL = T.total;

// One continuous shot: the caption rewrites itself while the picture morphs underneath.
const CUES: Cue[] = [
  [10, 86, 'Agents read your *secrets*.', C.bad],
  [90, 158, 'Then send them to the *model*.', C.bad],
  [164, 278, '*HushGate* masks them in flight.', C.gs],
  [284, 420, 'Personal data too: *IBAN, PESEL, cards*.', C.gs],
  [446, 546, '*Zero* leaks.', C.ok],
  [560, 636, 'One view for *every agent*.', C.gsDeep],
  [644, 838, 'Hijacked agents, *stopped*.', C.bad],
  [849, 1033, 'Spend, *capped*.', C.gsDeep],
  [1044, 1188, 'Shadow AI, *blocked*.', C.bad],
  [1199, 1313, 'Nine controls. *One policy file.*', C.gsDeep],
  [1324, 1458, 'Every action, *on record*.', C.gsDeep],
];

export const Video = () => {
  const f = useCurrentFrame();
  // The light world opens as a circle out of the gate; the dark one closes back in at the end.
  const r1 = kf(f, [[T.zeroIn, 0], [T.zeroIn + 40, 2400]]);
  const r2 = kf(f, [[T.end + 4, 0], [T.end + 40, 1200]]);
  const light = Math.min(1, Math.max(0, (r1 - 380) / 160)) * (1 - Math.min(1, Math.max(0, (r2 - 330) / 160)));
  const ink = interpolateColors(light, [0, 1], ['#FFFFFF', C.ink]);
  return (
    <AbsoluteFill style={{background: C.navy, fontFamily: SANS, color: C.ink, overflow: 'hidden'}}>
      {r1 > 0 && <AbsoluteFill style={{background: C.canvas, clipPath: `circle(${r1}px at ${GATE_X}px 530px)`}} />}
      {r2 > 0 && <AbsoluteFill style={{background: C.navy, clipPath: `circle(${r2}px at 960px 540px)`}} />}
      <Flow f={f} />
      <Dash f={f} />
      <Caption f={f} cues={CUES} color={ink} />
      <End f={f} />
    </AbsoluteFill>
  );
};

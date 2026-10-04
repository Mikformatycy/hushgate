import {AbsoluteFill, Series} from 'remotion';
import {C, SERIF} from './theme';
import {Scene} from './ui';
import {Hook} from './scenes/Hook';
import {Risk} from './scenes/Risk';
import {OneGate} from './scenes/OneGate';
import {Leaks} from './scenes/Leaks';
import {Filters} from './scenes/Filters';
import {Cost} from './scenes/Cost';
import {Oversight} from './scenes/Oversight';
import {Rollout} from './scenes/Rollout';
import {Close} from './scenes/Close';

// Durations in frames at 30 fps; they add up to exactly two minutes.
const SCENES: [React.FC, number][] = [
  [Hook, 300], [Risk, 420], [OneGate, 330], [Leaks, 570], [Filters, 540],
  [Cost, 540], [Oversight, 360], [Rollout, 270], [Close, 270],
];
export const TOTAL = SCENES.reduce((s, [, d]) => s + d, 0);

export const Video = () => (
  <AbsoluteFill style={{background: C.bg, color: C.fg, fontFamily: SERIF}}>
    <Series>
      {SCENES.map(([S, d], i) => (
        <Series.Sequence key={i} durationInFrames={d}>
          <Scene dur={d}><S /></Scene>
        </Series.Sequence>
      ))}
    </Series>
  </AbsoluteFill>
);

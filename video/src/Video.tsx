import React from 'react';
import {AbsoluteFill} from 'remotion';
import {TransitionSeries, linearTiming} from '@remotion/transitions';
import type {TransitionPresentation} from '@remotion/transitions';
import {fade} from '@remotion/transitions/fade';
import {slide} from '@remotion/transitions/slide';
import {wipe} from '@remotion/transitions/wipe';
import {C} from './theme';
import {Intro} from './scenes/Intro';
import {EnvFile} from './scenes/EnvFile';
import {Leak} from './scenes/Leak';
import {Reveal} from './scenes/Reveal';
import {GateFlow} from './scenes/GateFlow';
import {ZeroLeaks} from './scenes/ZeroLeaks';
import {Attack} from './scenes/Attack';
import {Signatures} from './scenes/Signatures';
import {Controls} from './scenes/Controls';
import {Cost} from './scenes/Cost';
import {ShadowAI} from './scenes/ShadowAI';
import {Audit} from './scenes/Audit';
import {DropIn} from './scenes/DropIn';
import {Proof} from './scenes/Proof';
import {End} from './scenes/End';

// Scenes and their length in frames (30 fps). Problem, reveal, results, proof.
const STEPS: [React.FC, number][] = [
  [Intro, 140], [EnvFile, 140], [Leak, 150], [Reveal, 105],
  [GateFlow, 285], [ZeroLeaks, 165], [Attack, 285], [Signatures, 150],
  [Controls, 240], [Cost, 270], [ShadowAI, 225], [Audit, 250],
  [DropIn, 165], [Proof, 135], [End, 240],
];
// Transition into the next scene, one per gap.
const T = 12;
const TRANS = [
  fade(), fade(), fade(), fade(),
  slide({direction: 'from-right'}), fade(), slide({direction: 'from-right'}), fade(),
  slide({direction: 'from-right'}), wipe({direction: 'from-left'}), wipe({direction: 'from-right'}), fade(),
  slide({direction: 'from-bottom'}), fade(),
] as TransitionPresentation<any>[];

export const TOTAL = STEPS.reduce((s, [, d]) => s + d, 0) - TRANS.length * T;

export const Video = () => {
  const items: React.ReactNode[] = [];
  STEPS.forEach(([Scene, d], i) => {
    items.push(<TransitionSeries.Sequence key={`s${i}`} durationInFrames={d}><Scene /></TransitionSeries.Sequence>);
    if (i < TRANS.length) items.push(<TransitionSeries.Transition key={`t${i}`} presentation={TRANS[i]} timing={linearTiming({durationInFrames: T})} />);
  });
  return (
    <AbsoluteFill style={{background: C.night}}>
      <TransitionSeries>{items}</TransitionSeries>
    </AbsoluteFill>
  );
};

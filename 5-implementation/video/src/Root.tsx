import {Composition} from 'remotion';
import {Video, TOTAL} from './Video';

export const Root = () => (
  <Composition id="HushGate" component={Video} durationInFrames={TOTAL} fps={30} width={1920} height={1080} defaultProps={{voice: 'tia' as const}} />
);

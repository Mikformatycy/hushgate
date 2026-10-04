import {Easing, useCurrentFrame} from 'remotion';
import {C} from '../theme';
import {Bg, Headline, Metric, Pop, tw} from '../kit';

const STATS: [number, string, string][] = [
  [57, '', 'Automated tests'],
  [19, '', 'Attack signatures'],
  [335, ' µs', 'Gate overhead (p50)'],
  [1, ' s', 'Policy reload'],
];

export const Proof = () => {
  const f = useCurrentFrame();
  return (
    <Bg color={C.page}>
      <Headline text="Measured, not promised." at={6} top={250} size={92} accent={C.link} accentWords={[0]} />
      <div style={{position: 'absolute', left: 175, right: 175, top: 500, display: 'flex', gap: 30}}>
        {STATS.map(([n, unit, label], i) => (
          <Pop key={label} at={18 + i * 7} y={60} style={{flex: 1}}>
            <Metric label={label} value={<>{Math.round(tw(f, 20 + i * 7, 50 + i * 7, 0, n, Easing.out(Easing.cubic)))}{unit}</>} valueSize={96} />
          </Pop>
        ))}
      </div>
    </Bg>
  );
};

import React from 'react';
import {useCurrentFrame} from 'remotion';
import {C, MONO} from '../theme';
import {Bg, Card, Headline, Pop, Status, easeInOut, tw} from '../kit';

const I = (d: React.ReactNode) => (c: string) => (
  <svg viewBox="0 0 24 24" width={46} height={46} fill="none" stroke={c} strokeWidth={2} strokeLinecap="round" strokeLinejoin="round">{d}</svg>
);
const ICONS = {
  lock: I(<><rect x="5" y="11" width="14" height="10" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></>),
  card: I(<><rect x="3" y="5" width="18" height="14" rx="2" /><path d="M3 10h18M7 15h4" /></>),
  route: I(<><circle cx="6" cy="6" r="2.5" /><circle cx="18" cy="18" r="2.5" /><path d="M8.5 6H15a3 3 0 0 1 0 6H9a3 3 0 0 0 0 6h6.5" /></>),
  shield: I(<path d="M12 3 5 6v5c0 4.5 3 8.3 7 10 4-1.7 7-5.5 7-10V6z" />),
  term: I(<><rect x="3" y="4" width="18" height="16" rx="2" /><path d="m7 10 3 2-3 2M13 15h4" /></>),
  warn: I(<><path d="M12 4 2.5 20h19z" /><path d="M12 10v4M12 17h.01" /></>),
  laptop: I(<><rect x="5" y="5" width="14" height="10" rx="1.5" /><path d="M3 19h18" /></>),
  chip: I(<><rect x="7" y="7" width="10" height="10" rx="1.5" /><path d="M10 3v4M14 3v4M10 17v4M14 17v4M3 10h4M3 14h4M17 10h4M17 14h4" /></>),
  gauge: I(<><path d="M4 17a8 8 0 1 1 16 0" /><path d="m12 17 4-5" /></>),
};
const GROUPS: {c: string; items: [keyof typeof ICONS, string, string][]}[] = [
  {c: C.orange, items: [['lock', 'Secret masking', 'four sensitivity tiers'], ['card', 'Personal data', 'IBAN · cards · PESEL · NIP'], ['route', 'Tool policy', 'local · network · deny']]},
  {c: C.bad, items: [['shield', 'Attack signatures', '19 patterns · live feed'], ['term', 'Shell guard', 'curl · scp · ssh in Bash'], ['warn', 'Injection warning', 'AI early warning']]},
  {c: C.link, items: [['laptop', 'Device allowlist', 'no shadow AI'], ['chip', 'Model allowlist', 'approved models only'], ['gauge', 'Budgets', 'per agent · kill switch']]},
];
const W = 500, H = 168, GAP = 30, X0 = 180, Y0 = 270;
const MERGE = 168;

export const Controls = () => {
  const f = useCurrentFrame();
  const k = tw(f, MERGE, MERGE + 22, 0, 1, easeInOut);
  return (
    <Bg color={C.page}>
      <Headline text="Nine controls. One policy." at={6} out={MERGE - 4} top={96} size={88} accentWords={[2, 3]} accent={C.link} />
      {GROUPS.flatMap((g, col) =>
        g.items.map(([icon, name, sub], row) => {
          const x = X0 + col * (W + GAP), y = Y0 + row * (H + GAP);
          const dx = (960 - (x + W / 2)) * k, dy = (580 - (y + H / 2)) * k;
          return (
            <div key={name} style={{position: 'absolute', left: x, top: y, width: W, transform: `translate(${dx}px, ${dy}px) scale(${1 - 0.75 * k})`, opacity: 1 - k}}>
              <Pop at={16 + (col * 3 + row) * 6} from={0.5} y={40}>
                <Card style={{height: H, display: 'flex', alignItems: 'center', gap: 26, padding: '0 32px', borderLeft: `8px solid ${g.c}`}}>
                  {ICONS[icon](g.c)}
                  <div>
                    <div style={{fontSize: 34, fontWeight: 700}}>{name}</div>
                    <div style={{fontSize: 23, color: C.sub, marginTop: 4}}>{sub}</div>
                  </div>
                </Card>
              </Pop>
            </div>
          );
        }),
      )}
      <Pop at={MERGE + 14} from={0.3} y={0} style={{position: 'absolute', left: 560, top: 430, width: 800}}>
        <Card style={{padding: '40px 50px', display: 'flex', alignItems: 'center', gap: 36}}>
          <svg viewBox="0 0 24 24" width={110} height={110} fill="none" stroke={C.link} strokeWidth={1.6} strokeLinejoin="round"><path d="M6 2h8l5 5v15H6z" /><path d="M14 2v5h5" /><path d="M9 12h7M9 15h7M9 18h4" /></svg>
          <div>
            <div style={{fontFamily: MONO, fontSize: 48, fontWeight: 600}}>hushgate.yaml</div>
            <Pop at={MERGE + 30} from={0.9} y={8} style={{marginTop: 10}}><Status tone="ok" size={30}>Live: changes apply in 1 s</Status></Pop>
          </div>
        </Card>
      </Pop>
    </Bg>
  );
};

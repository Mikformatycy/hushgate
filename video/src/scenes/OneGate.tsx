import {C} from '../theme';
import {Appear, DrawBox, DrawLine, Dot} from '../ui';

const AGENTS = ['Claude Code', 'CI pipelines', 'Laptops'];
const label = {position: 'absolute' as const, display: 'flex', alignItems: 'center', justifyContent: 'center', fontSize: 34};

export const OneGate = () => (
  <>
    <Appear at={0} style={{position: 'absolute', left: 160, top: 110, fontSize: 32, fontStyle: 'italic', color: C.muted}}>The answer</Appear>
    {AGENTS.map((a, i) => {
      const y = 330 + i * 150;
      return (
        <div key={a}>
          <DrawBox x={200} y={y} w={340} h={96} at={15 + i * 10} />
          <Appear at={30 + i * 10} style={{...label, left: 200, top: y, width: 340, height: 96}}>{a}</Appear>
          <DrawLine x1={540} y1={y + 48} x2={800} y2={528} at={70 + i * 6} />
          <Dot x1={540} y1={y + 48} x2={800} y2={528} at={110 + i * 17} period={66} />
        </div>
      );
    })}
    <DrawBox x={800} y={378} w={320} h={300} at={50} dur={40} color={C.blue} />
    <Appear at={75} style={{...label, left: 800, top: 378, width: 320, height: 300, fontSize: 52, fontWeight: 600, color: C.blue}}>HushGate</Appear>
    <DrawLine x1={1120} y1={528} x2={1380} y2={528} at={90} />
    <Dot x1={1120} y1={528} x2={1380} y2={528} at={125} period={44} />
    <DrawBox x={1380} y={468} w={340} h={120} at={95} />
    <Appear at={110} style={{...label, left: 1380, top: 468, width: 340, height: 120}}>Model providers</Appear>
    <Appear at={40} style={{position: 'absolute', left: 160, right: 160, top: 820, textAlign: 'center', fontSize: 44}}>
      One gate between every agent and every model.
    </Appear>
    <Appear at={170} style={{position: 'absolute', left: 160, right: 160, top: 900, textAlign: 'center', fontSize: 34, color: C.muted, fontStyle: 'italic'}}>
      No code changes: one setting, or automatic on the company network.
    </Appear>
  </>
);

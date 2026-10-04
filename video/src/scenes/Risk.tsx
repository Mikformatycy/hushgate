import {C} from '../theme';
import {Appear, Title} from '../ui';

const Q = [
  {q: 'What data left the company?', a: 'Secrets and customer data reach the model provider in plain text.', c: C.yellow},
  {q: 'Can an agent be turned against us?', a: 'One poisoned document can make an agent email credentials or run a remote script.', c: C.red},
  {q: 'What are we spending, and where?', a: 'Agents loop, pick expensive models, and run on services nobody approved.', c: C.blue},
];

export const Risk = () => (
  <>
    <Title eyebrow="The risk">Three questions a CTO has to answer</Title>
    <div style={{position: 'absolute', left: 160, right: 160, top: 360, display: 'flex', flexDirection: 'column', gap: 52}}>
      {Q.map((x, i) => (
        <Appear key={x.q} at={50 + i * 75} style={{borderLeft: `3px solid ${x.c}`, paddingLeft: 32}}>
          <div style={{fontSize: 46}}>{x.q}</div>
          <div style={{fontSize: 32, color: C.muted, marginTop: 8}}>{x.a}</div>
        </Appear>
      ))}
    </div>
    <Appear at={300} style={{position: 'absolute', left: 160, right: 160, bottom: 90, fontSize: 38, fontStyle: 'italic'}}>
      Without a control layer, there is no record to answer them from.
    </Appear>
  </>
);

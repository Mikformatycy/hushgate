import {C} from '../theme';
import {Appear, Title} from '../ui';

const GROUPS = [
  {name: 'Protect data', c: C.yellow, items: [
    ['Secret masking', 'four sensitivity tiers, C0 to C3'],
    ['Personal data detection', 'IBAN, cards, PESEL, NIP, checksum-verified'],
    ['Tool policy', 'each tool is local, network or denied'],
  ]},
  {name: 'Stop attacks', c: C.red, items: [
    ['Attack signatures', '19 known patterns, one central feed'],
    ['Shell command guard', 'curl, scp or ssh inside a command'],
    ['Prompt injection warning', 'an AI model reads what the agent reads'],
  ]},
  {name: 'Control use', c: C.blue, items: [
    ['Device allowlist', 'blocks shadow AI, even on private servers'],
    ['Model allowlist', 'only the models you approved'],
    ['Budgets and kill switch', 'token limits per agent, instant stop'],
  ]},
];

export const Filters = () => (
  <>
    <Title eyebrow="What it checks">Nine controls, one policy file</Title>
    <div style={{position: 'absolute', left: 160, right: 160, top: 350, display: 'flex', gap: 70}}>
      {GROUPS.map((g, gi) => (
        <div key={g.name} style={{flex: 1, display: 'flex', flexDirection: 'column', gap: 34}}>
          <Appear at={30 + gi * 70} style={{fontSize: 40, fontWeight: 600, color: g.c, borderBottom: `2px solid ${g.c}`, paddingBottom: 12}}>{g.name}</Appear>
          {g.items.map(([n, d], i) => (
            <Appear key={n} at={45 + gi * 70 + i * 18}>
              <div style={{fontSize: 34}}>{n}</div>
              <div style={{fontSize: 27, color: C.muted, marginTop: 6, lineHeight: 1.3}}>{d}</div>
            </Appear>
          ))}
        </div>
      ))}
    </div>
    <Appear at={330} style={{position: 'absolute', left: 160, right: 160, bottom: 90, fontSize: 36, fontStyle: 'italic'}}>
      Changes apply within a second. A broken edit is rejected and the old policy stays in force.
    </Appear>
  </>
);

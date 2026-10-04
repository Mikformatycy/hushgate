import type {Rect} from '../kit';

// One continuous timeline (frames at 30 fps). Every act starts from the state the previous one ended in,
// and holds long enough after its last element appears to be read.
export const T = {
  leak: 90,
  gateOn: 160,
  gateActive: 174,
  pii: 284,
  piiDetect: 306,
  piiStream: 340,
  zeroIn: 426,
  zeroDone: 472,
  toTile: 548,
  tileDone: 590,
  frameIn: 556,
  attack: [640, 840] as const,
  spend: [845, 1035] as const,
  shadow: [1040, 1190] as const,
  policy: [1195, 1315] as const,
  audit: [1320, 1460] as const,
  end: 1460,
  total: 1620,
};

export const ENV: Rect = {x: 190, y: 330, w: 650, h: 420, r: 22};
export const ENV_WIDE = 770;
export const MODEL = {x: 1640, y: 520, r: 116};
export const GATE_X = 1200;
export const DOT: Rect = {x: GATE_X - 6, y: 524, w: 12, h: 12, r: 6};
export const GATE: Rect = {x: GATE_X - 62, y: 270, w: 124, h: 520, r: 62};
export const ZERO: Rect = {x: 560, y: 290, w: 800, h: 450, r: 28};
export const FRAME: Rect = {x: 120, y: 190, w: 1680, h: 820, r: 28};
export const TILE = (i: number): Rect => ({x: 252 + i * 383, y: 296, w: 359, h: 178, r: 20});
export const DETAIL: Rect = {x: 252, y: 500, w: 1508, h: 474, r: 20};

import type {Rect} from '../kit';

// One continuous timeline (frames at 30 fps). Every act starts from the state the previous one ended in.
export const T = {
  leak: 96,
  gateOn: 172,
  gateActive: 186,
  zeroIn: 330,
  zeroDone: 376,
  toTile: 440,
  tileDone: 482,
  frameIn: 448,
  attack: [520, 700] as const,
  spend: [705, 860] as const,
  shadow: [865, 990] as const,
  policy: [995, 1110] as const,
  audit: [1115, 1230] as const,
  end: 1230,
  total: 1400,
};

export const ENV: Rect = {x: 190, y: 330, w: 650, h: 420, r: 22};
export const MODEL = {x: 1640, y: 520, r: 116};
export const GATE_X = 1200;
export const DOT: Rect = {x: GATE_X - 6, y: 524, w: 12, h: 12, r: 6};
export const GATE: Rect = {x: GATE_X - 62, y: 270, w: 124, h: 520, r: 62};
export const ZERO: Rect = {x: 560, y: 290, w: 800, h: 450, r: 28};
export const FRAME: Rect = {x: 120, y: 190, w: 1680, h: 820, r: 28};
export const TILE = (i: number): Rect => ({x: 252 + i * 383, y: 296, w: 359, h: 178, r: 20});
export const DETAIL: Rect = {x: 252, y: 500, w: 1508, h: 474, r: 20};

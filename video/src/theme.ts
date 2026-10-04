import {loadFont as loadSans} from '@remotion/google-fonts/OpenSans';
import {loadFont as loadMono} from '@remotion/google-fonts/JetBrainsMono';

export const SANS = loadSans('normal', {weights: ['300', '400', '600', '700', '800'], subsets: ['latin']}).fontFamily;
export const MONO = loadMono('normal', {weights: ['400', '600'], subsets: ['latin']}).fontFamily;

// The dashboard's palette (dashboard/src/index.css), plus a night tone for the network shots.
export const C = {
  nav: '#232f3e',
  night: '#0f1722',
  nightNode: '#1a2433',
  nightLine: '#34465c',
  nightText: '#9aa7b6',
  page: '#f2f3f3',
  ink: '#16191f',
  sub: '#5f6b7a',
  line: '#e9ebed',
  link: '#0073bb',
  ok: '#1d8102',
  bad: '#d13212',
  orange: '#ff9900',
  warnInk: '#8d6605',
  warnBg: '#fff8e6',
};
export const SHADOW = '0 2px 3px rgba(0,28,36,.22), 0 12px 32px rgba(0,28,36,.10)';

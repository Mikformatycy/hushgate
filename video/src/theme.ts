import {loadFont as loadSans} from '@remotion/google-fonts/Manrope';
import {loadFont as loadMono} from '@remotion/google-fonts/JetBrainsMono';

export const SANS = loadSans('normal', {weights: ['400', '500', '600', '700', '800'], subsets: ['latin']}).fontFamily;
export const MONO = loadMono('normal', {weights: ['400', '600'], subsets: ['latin']}).fontFamily;

// Primary is the challenge partner's blue, rgb(114, 151, 197). GS_DEEP is the same hue,
// darkened for text on light backgrounds. Semantic colors (green, red, violet) stay separate from it.
export const C = {
  gs: '#7297C5',
  gsDeep: '#3F6699',
  gsTint: '#EAF0F8',
  navy: '#0C1A2B',
  navy2: '#13243A',
  navyLine: '#29405C',
  navyText: '#93A6BE',
  canvas: '#F3F5F8',
  card: '#FFFFFF',
  border: '#E1E7EF',
  ink: '#0C1A2B',
  sub: '#5C6B7E',
  ok: '#2E9B6B',
  okTint: '#E6F4EC',
  bad: '#D5443B',
  badTint: '#FBEAE8',
  // Warnings raised by the AI (prompt injection) are violet, so nothing reads as orange or yellow.
  warn: '#7B5CC4',
  warnTint: '#F0EBFA',
};

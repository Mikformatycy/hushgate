import {loadFont as loadSerif} from '@remotion/google-fonts/SourceSerif4';
import {loadFont as loadMono} from '@remotion/google-fonts/JetBrainsMono';

export const SERIF = loadSerif('normal', {weights: ['400', '600'], subsets: ['latin']}).fontFamily;
loadSerif('italic', {weights: ['400'], subsets: ['latin']});
export const MONO = loadMono('normal', {weights: ['400'], subsets: ['latin']}).fontFamily;

// One meaning per color: blue = HushGate and normal traffic, yellow = sensitive data,
// green = safe or allowed, red = blocked or at risk.
export const C = {
  bg: '#111214',
  fg: '#E9E9E6',
  muted: '#8C9097',
  dim: '#3A3D43',
  blue: '#58C4DD',
  yellow: '#F2CF5B',
  green: '#83C167',
  red: '#FC6255',
};

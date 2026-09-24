// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Colours of the original WinMIPS64 (mytypes.h). They stay identical in light and dark themes.
export const C = {
  yellow: '#ffff00',
  cyan: '#00ffff',
  red: '#ff0000',
  green: '#00ff00',
  magenta: '#ff00ff',
  dgreen: '#008000',
  dcyan: '#008080',
  dyellow: '#808000',
  grey: '#808080',
  blue: '#0000ff',
} as const;

/** Pipeline diagram / Cycles grid colour for a stage name (HistCell.stage or a pipe unit). */
export function stageColor(stage: string): string | undefined {
  if (stage === 'IF') return C.yellow;
  if (stage === 'ID') return C.cyan;
  if (stage === 'EX') return C.red;
  if (stage === 'MEM') return C.green;
  if (stage === 'WB') return C.magenta;
  if (stage === 'DIV') return C.dyellow;
  if (/^A\d$/.test(stage)) return C.dgreen;
  if (/^M\d$/.test(stage)) return C.dcyan;
  return undefined;
}

/** Register text colour by `source` (CONTRACT §4). */
export function sourceColor(source: string): string | undefined {
  switch (source) {
    case 'id': return C.cyan;
    case 'ex': return C.red;
    case 'mem': return C.green;
    case 'add': return C.dgreen;
    case 'mul': return C.dcyan;
    case 'div': return C.dyellow;
    case 'na': return C.grey;
    default: return undefined;
  }
}

/** Light stage colours need dark text; dark ones need light text. */
export function textOn(bg: string): string {
  return bg === C.dgreen || bg === C.dcyan || bg === C.dyellow || bg === C.blue ? '#ffffff' : '#000000';
}

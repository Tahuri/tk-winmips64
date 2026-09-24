// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest';
import { es } from './es';
import { en } from './en';
import { ASM_ERROR_CODES, MESSAGE_CODES } from '../sim/types';
import { initI18n, translateAsmError, translateMessage } from './index';

function flatten(obj: object, prefix = ''): Record<string, string> {
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(obj)) {
    const key = prefix ? `${prefix}.${k}` : k;
    if (v && typeof v === 'object') Object.assign(out, flatten(v as object, key));
    else out[key] = String(v);
  }
  return out;
}

const interp = (s: string) => [...s.matchAll(/{{\s*(\w+)\s*}}/g)].map((m) => m[1]).sort();

describe('i18n resources', () => {
  const fes = flatten(es);
  const fen = flatten(en);

  it('every es key exists in en and vice versa', () => {
    expect(Object.keys(fes).filter((k) => !(k in fen))).toEqual([]);
    expect(Object.keys(fen).filter((k) => !(k in fes))).toEqual([]);
  });

  it('no empty strings and same interpolation variables', () => {
    for (const k of Object.keys(fes)) {
      expect(fes[k].trim(), `es:${k}`).not.toBe('');
      expect(fen[k].trim(), `en:${k}`).not.toBe('');
      expect(interp(fen[k]), k).toEqual(interp(fes[k]));
    }
  });

  it('covers every message code (CONTRACT §3) and assembler error code', () => {
    for (const c of MESSAGE_CODES) {
      expect(fes[`msg.${c}`], c).toBeDefined();
      expect(fen[`msg.${c}`], c).toBeDefined();
    }
    for (const c of ASM_ERROR_CODES) {
      expect(fes[`asm.${c}`], c).toBeDefined();
      expect(fen[`asm.${c}`], c).toBeDefined();
    }
  });

  it('English message texts match the canonical CONTRACT §3 texts', async () => {
    const i18n = initI18n('en');
    await i18n.changeLanguage('en');
    const t = i18n.t.bind(i18n) as (k: string, o?: Record<string, unknown>) => string;
    expect(translateMessage(t, { code: 'raw_stall', stage: 'ID', reg: 'R5' })).toBe('RAW Stall in ID (R5)');
    expect(translateMessage(t, { code: 'structural_stall', stage: 'MEM' })).toBe('Structural Stall in MEM');
    expect(translateMessage(t, { code: 'data_misaligned' })).toBe('Fatal Error - misaligned memory LOAD/STORE!');
    expect(translateMessage(t, { code: 'nope' })).toBe('Unknown message: nope');
    expect(translateAsmError(t, { line: 1, code: 'bad_register', text: '' })).toBe('Invalid register');
    await i18n.changeLanguage('es');
    expect(translateMessage(t, { code: 'raw_stall', stage: 'EX', reg: 'F2' })).toBe('Atasco RAW en EX (F2)');
  });
});

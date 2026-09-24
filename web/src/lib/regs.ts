// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

export const ABI_NAMES = [
  'zero', 'at', 'v0', 'v1', 'a0', 'a1', 'a2', 'a3',
  't0', 't1', 't2', 't3', 't4', 't5', 't6', 't7',
  's0', 's1', 's2', 's3', 's4', 's5', 's6', 's7',
  't8', 't9', 'k0', 'k1', 'gp', 'sp', 'fp', 'ra',
];

/** Display name for integer register i (presentation only, CONTRACT Config.registersAsNumbers). */
export function intRegName(i: number, asNumbers: boolean): string {
  return asNumbers ? `R${i}` : ABI_NAMES[i] ?? `R${i}`;
}

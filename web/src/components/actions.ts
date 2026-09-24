// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { useSim } from '../state/store';
import { openSourceFile, saveSourceFile, fetchExample } from '../lib/files';

export async function actionOpen() {
  const f = await openSourceFile();
  if (!f) return;
  useSim.getState().setSource(f.text, f.name);
  await useSim.getState().assemble();
}

export async function actionSave() {
  const { fileName, source, setSource } = useSim.getState();
  const saved = await saveSourceFile(fileName, source);
  if (saved && saved !== fileName) setSource(source, saved);
}

export async function actionLoadExample(name: string) {
  const text = await fetchExample(name);
  useSim.getState().setSource(text, name);
  await useSim.getState().assemble();
}

export function toggleConfigFlag(flag: 'forwarding' | 'delaySlot' | 'btb' | 'registersAsNumbers') {
  const { config, applyConfig } = useSim.getState();
  const next = { ...config, [flag]: !config[flag] };
  if (flag === 'delaySlot' && next.delaySlot) next.btb = false;
  if (flag === 'btb' && next.btb) next.delaySlot = false;
  void applyConfig(next);
}

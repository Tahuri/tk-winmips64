// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import type { Engine } from './types';

export interface BootOptions {
  mock: boolean;
  /** Base URL where `wasm/wmips.wasm` and `wasm/wasm_exec.js` live (usually "/"). */
  base: string;
  /** Config used for wmips.init(). */
  config: import('./types').Config;
}

export interface BootResult {
  kind: 'wasm' | 'mock';
  /** Why the WASM engine was not used (when kind === "mock" and mock was not requested). */
  fallbackReason?: string;
}

export type EngineApi = Omit<Engine, 'kind'>;

export interface WorkerApi extends EngineApi {
  boot(opts: BootOptions): BootResult;
}

export type Method = keyof WorkerApi;

export interface Request {
  id: number;
  method: Method;
  args: unknown[];
}

export type Response =
  | { id: number; ok: true; result: unknown }
  | { id: number; ok: false; error: string };

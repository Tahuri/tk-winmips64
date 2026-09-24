// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

import { create } from 'zustand';
import { SimClient } from '../sim/client';
import { DEFAULT_CONFIG, CONFIG_LIMITS, type AsmError, type Config, type Message, type Program, type Snapshot } from '../sim/types';
import { storage } from '../lib/storage';
import { DEFAULT_FILE_NAME, DEFAULT_PROGRAM } from '../lib/defaultProgram';

/** A status-bar entry: an i18n key with params, or the simulator messages of the last cycle. */
export type StatusLine =
  | { kind: 'key'; key: string; params?: Record<string, unknown> }
  | { kind: 'messages'; messages: Message[] };

export interface SimState {
  client: SimClient | null;
  engineKind: 'wasm' | 'mock' | null;
  booted: boolean;
  config: Config;
  source: string;
  fileName: string;
  /** Source text that produced the current program (to flag the editor as modified). */
  assembledSource: string | null;
  program: Program | null;
  snapshot: Snapshot | null;
  asmErrors: AsmError[];
  running: boolean;
  busy: boolean;
  multiCycles: number;
  status: StatusLine;

  boot(client: SimClient, mock: boolean): Promise<void>;
  setSource(src: string, fileName?: string): void;
  assemble(): Promise<boolean>;
  reload(): Promise<void>;
  step(): Promise<void>;
  stepN(): Promise<void>;
  runTo(): Promise<void>;
  stop(): void;
  reset(full: boolean): Promise<void>;
  applyConfig(cfg: Config): Promise<void>;
  setMultiCycles(n: number): void;
  toggleBreakpoint(addr: number): Promise<void>;
  setReg(i: number, hex: string): Promise<boolean>;
  setFReg(i: number, f: number): Promise<boolean>;
  setMem(addr: number, hex: string): Promise<boolean>;
  setMemDouble(addr: number, f: number): Promise<boolean>;
  sendInput(text: string): Promise<boolean>;
}

export function sanitizeConfig(c: Partial<Config> | undefined): Config {
  const cfg: Config = { ...DEFAULT_CONFIG, ...(c ?? {}) };
  for (const [k, [lo, hi]] of Object.entries(CONFIG_LIMITS)) {
    const key = k as keyof typeof CONFIG_LIMITS;
    const v = Number(cfg[key]);
    cfg[key] = Number.isInteger(v) ? Math.min(hi, Math.max(lo, v)) : DEFAULT_CONFIG[key];
  }
  if (cfg.delaySlot && cfg.btb) cfg.btb = false;
  return cfg;
}

let stopRequested = false;

function statusFromSnapshot(s: Snapshot): StatusLine {
  if (s.messages && s.messages.length > 0) return { kind: 'messages', messages: s.messages };
  if (s.status === 'halted') return { kind: 'key', key: 'status.halted' };
  if (s.status === 'waiting_input') return { kind: 'key', key: 'status.waitingInput' };
  return { kind: 'key', key: 'status.cycle', params: { n: s.cycles } };
}

export const useSim = create<SimState>((set, get) => {
  const persist = () => {
    const { source, config, fileName, multiCycles } = get();
    storage.set('source', source);
    storage.set('fileName', fileName);
    storage.set('config', config);
    storage.set('multi', multiCycles);
  };

  const refresh = async (withStatus = true) => {
    const client = get().client;
    if (!client) return;
    const snapshot = await client.call('snapshot');
    set(withStatus ? { snapshot, status: statusFromSnapshot(snapshot) } : { snapshot });
  };

  // Engine operations are serialised: key repeats (e.g. F7 pressed quickly) are queued, not dropped.
  let chain: Promise<unknown> = Promise.resolve();
  let pending = 0;
  const guard = <T>(fn: (c: SimClient) => Promise<T>): Promise<T | undefined> => {
    const client = get().client;
    if (!client || pending > 64) return Promise.resolve(undefined);
    pending++;
    set({ busy: true });
    const run = chain.then(async () => {
      try {
        return await fn(client);
      } catch (e) {
        set({ status: { kind: 'key', key: 'status.error', params: { error: e instanceof Error ? e.message : String(e) } } });
        return undefined;
      } finally {
        pending--;
        if (pending === 0) set({ busy: false });
      }
    });
    chain = run;
    return run;
  };

  const needProgram = (): boolean => {
    const s = get().snapshot;
    if (!s || !s.loaded) {
      set({ status: { kind: 'key', key: 'status.noProgram' } });
      return false;
    }
    return true;
  };

  return {
    client: null,
    engineKind: null,
    booted: false,
    config: sanitizeConfig(storage.get<Partial<Config>>('config')),
    source: storage.get<string>('source') ?? DEFAULT_PROGRAM,
    fileName: storage.get<string>('fileName') ?? DEFAULT_FILE_NAME,
    assembledSource: null,
    program: null,
    snapshot: null,
    asmErrors: [],
    running: false,
    busy: false,
    multiCycles: Math.min(10000, Math.max(1, storage.get<number>('multi') ?? 5)),
    status: { kind: 'key', key: 'status.booting' },

    async boot(client, mock) {
      set({ client });
      const r = await client.call('boot', { mock, base: import.meta.env?.BASE_URL ?? '/', config: get().config });
      set({ engineKind: r.kind, booted: true });
      set({
        status:
          r.kind === 'wasm'
            ? { kind: 'key', key: 'status.engineWasm' }
            : r.fallbackReason
              ? { kind: 'key', key: 'status.engineFallback', params: { reason: r.fallbackReason } }
              : { kind: 'key', key: 'status.engineMock' },
      });
      const status = get().status;
      if (await get().assemble()) set({ status });
    },

    setSource(src, fileName) {
      set(fileName ? { source: src, fileName } : { source: src });
      persist();
    },

    async assemble() {
      const ok = await guard(async (c) => {
        const src = get().source;
        const r = await c.call('load', src);
        const errors = r.errors ?? [];
        if (!r.ok || errors.length > 0) {
          set({ asmErrors: errors, program: null, assembledSource: null, status: { kind: 'key', key: 'status.asmFailed', params: { count: errors.length } } });
          await refresh(false);
          return false;
        }
        const program = await c.call('program');
        set({ asmErrors: [], program, assembledSource: src, status: { kind: 'key', key: 'status.assembled' } });
        await refresh(false);
        return true;
      });
      return ok ?? false;
    },

    async reload() {
      await get().assemble();
    },

    async step() {
      if (!needProgram() || get().running) return;
      await guard(async (c) => {
        await c.call('step');
        await refresh();
      });
    },

    async stepN() {
      if (!needProgram() || get().running) return;
      await guard(async (c) => {
        await c.call('stepN', get().multiCycles);
        await refresh();
      });
    },

    async runTo() {
      if (!needProgram() || get().running) return;
      stopRequested = false;
      set({ running: true, status: { kind: 'key', key: 'status.running', params: { cycles: 0 } } });
      await guard(async (c) => {
        const r = await c.run({
          shouldStop: () => stopRequested,
          onBatch: async () => {
            await refresh(false);
            set({ status: { kind: 'key', key: 'status.running', params: { cycles: get().snapshot?.cycles ?? 0 } } });
          },
        });
        await refresh();
        const snap = get().snapshot;
        if (r.stoppedBy === 'breakpoint') set({ status: { kind: 'key', key: 'status.breakpoint' } });
        else if (r.stoppedBy === 'limit' && stopRequested) set({ status: { kind: 'key', key: 'status.stopped' } });
        else if (r.stoppedBy === 'halted' && snap && (!snap.messages || snap.messages.length === 0)) set({ status: { kind: 'key', key: 'status.halted' } });
      });
      set({ running: false });
    },

    stop() {
      stopRequested = true;
    },

    async reset(full) {
      await guard(async (c) => {
        await c.call('reset', full);
        await refresh(false);
        set({ status: { kind: 'key', key: full ? 'status.fullReset' : 'status.reset' } });
      });
    },

    async applyConfig(cfg) {
      const next = sanitizeConfig(cfg);
      const prev = get().config;
      set({ config: next });
      persist();
      const onlyPresentation = (Object.keys(next) as (keyof Config)[]).every((k) => k === 'registersAsNumbers' || next[k] === prev[k]);
      if (onlyPresentation) {
        // Presentation-only change: no reset needed, but let the engine know (it may name registers).
        return;
      }
      const ok = await guard(async (c) => {
        const r = await c.call('setConfig', next);
        if (!r.ok) throw new Error(r.error ?? 'setConfig failed');
        return true;
      });
      if (ok) {
        await get().assemble();
        if (get().asmErrors.length === 0) set({ status: { kind: 'key', key: 'status.configApplied' } });
      }
    },

    setMultiCycles(n) {
      set({ multiCycles: Math.min(10000, Math.max(1, Math.floor(n) || 1)) });
      persist();
    },

    async toggleBreakpoint(addr) {
      await guard(async (c) => {
        await c.call('toggleBreakpoint', addr);
        await refresh(false);
      });
    },

    async setReg(i, hex) {
      return (await guard(async (c) => {
        const r = await c.call('setReg', i, hex);
        await refresh(false);
        return r.ok;
      })) ?? false;
    },

    async setFReg(i, f) {
      return (await guard(async (c) => {
        const r = await c.call('setFReg', i, f);
        await refresh(false);
        return r.ok;
      })) ?? false;
    },

    async setMem(addr, hex) {
      return (await guard(async (c) => {
        const r = await c.call('setMem', addr, hex);
        await refresh(false);
        return r.ok;
      })) ?? false;
    },

    async setMemDouble(addr, f) {
      return (await guard(async (c) => {
        const r = await c.call('setMemDouble', addr, f);
        await refresh(false);
        return r.ok;
      })) ?? false;
    },

    async sendInput(text) {
      return (await guard(async (c) => {
        const r = await c.call('sendInput', text);
        await refresh();
        if (!r.ok) set({ status: { kind: 'key', key: 'status.error', params: { error: r.error ?? '' } } });
        return r.ok;
      })) ?? false;
    },
  };
});

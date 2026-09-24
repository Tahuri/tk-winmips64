// Request dispatcher shared by the Web Worker and the in-process transport used in tests.
import { MockEngine } from './mockEngine';
import type { BootOptions, BootResult, Request, Response } from './protocol';
import { DEFAULT_CONFIG, type Engine } from './types';
import { loadWasmEngine } from './wasmEngine';

export type EngineLoader = (opts: BootOptions) => Promise<Engine>;

export function createHandler(loadWasm: EngineLoader = (o) => loadWasmEngine(o.base)) {
  let engine: Engine | null = null;

  async function boot(opts: BootOptions): Promise<BootResult> {
    let fallbackReason: string | undefined;
    if (!opts.mock) {
      try {
        engine = await loadWasm(opts);
      } catch (e) {
        fallbackReason = e instanceof Error ? e.message : String(e);
        engine = null;
      }
    }
    if (!engine) engine = new MockEngine();
    const r = engine.init(opts.config);
    if (!r.ok) engine.init(DEFAULT_CONFIG);
    return { kind: engine.kind, fallbackReason };
  }

  return async function handle(req: Request): Promise<Response> {
    try {
      if (req.method === 'boot') {
        return { id: req.id, ok: true, result: await boot(req.args[0] as BootOptions) };
      }
      if (!engine) throw new Error('engine not booted');
      const fn = (engine as unknown as Record<string, (...a: unknown[]) => unknown>)[req.method];
      if (typeof fn !== 'function') throw new Error(`unknown method ${req.method}`);
      return { id: req.id, ok: true, result: fn.apply(engine, req.args) };
    } catch (e) {
      return { id: req.id, ok: false, error: e instanceof Error ? e.message : String(e) };
    }
  };
}

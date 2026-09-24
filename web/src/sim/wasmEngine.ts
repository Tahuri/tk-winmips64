// Adapter over the Go WASM bridge (CONTRACT §5): global `wmips` whose functions return JSON strings.
import type {
  Config, Engine, LoadResult, OkResult, Program, RunResult, Snapshot, StepResult,
} from './types';

type Bridge = Record<string, (...args: unknown[]) => unknown>;

function parse<T>(v: unknown): T {
  return (typeof v === 'string' ? JSON.parse(v) : v) as T;
}

export class WasmEngine implements Engine {
  readonly kind = 'wasm' as const;
  constructor(private readonly w: Bridge) {}
  private call<T>(name: string, ...args: unknown[]): T {
    const fn = this.w[name];
    if (typeof fn !== 'function') throw new Error(`wmips.${name} is not available`);
    return parse<T>(fn(...args));
  }
  init(cfg: Config): OkResult { return this.call('init', JSON.stringify(cfg)); }
  setConfig(cfg: Config): OkResult { return this.call('setConfig', JSON.stringify(cfg)); }
  load(src: string): LoadResult { return this.call('load', src); }
  program(): Program { return this.call('program'); }
  snapshot(): Snapshot { return this.call('snapshot'); }
  step(): StepResult { return this.call('step'); }
  stepN(n: number): StepResult { return this.call('stepN', n); }
  runTo(maxCycles: number): RunResult { return this.call('runTo', maxCycles); }
  reset(full: boolean): OkResult { return this.call('reset', full); }
  sendInput(text: string): OkResult { return this.call('sendInput', text); }
  toggleBreakpoint(addr: number): OkResult { return this.call('toggleBreakpoint', addr); }
  setReg(i: number, hex: string): OkResult { return this.call('setReg', i, hex); }
  setFReg(i: number, f: number): OkResult { return this.call('setFReg', i, f); }
  setMem(addr: number, hex: string): OkResult { return this.call('setMem', addr, hex); }
  setMemDouble(addr: number, f: number): OkResult { return this.call('setMemDouble', addr, f); }
}

interface GoInstance {
  importObject: WebAssembly.Imports;
  run(instance: WebAssembly.Instance): Promise<void>;
}

/** Loads wasm_exec.js + wmips.wasm into the current (worker) global scope. Throws on any failure. */
export async function loadWasmEngine(base: string, timeoutMs = 5000): Promise<WasmEngine> {
  const g = globalThis as unknown as { Go?: new () => GoInstance; wmips?: Bridge };
  if (!g.Go) {
    // wasm_exec.js is a side-effect script that defines globalThis.Go. A dynamic
    // import works in module workers and keeps the CSP free of 'unsafe-eval'.
    const url = new URL(`${base}wasm/wasm_exec.js`, self.location.href).href;
    try {
      await import(/* @vite-ignore */ url);
    } catch (e) {
      throw new Error(`wasm_exec.js not loaded: ${e instanceof Error ? e.message : String(e)}`);
    }
    if (!g.Go) throw new Error('wasm_exec.js did not define Go');
  }
  const resp = await fetch(`${base}wasm/wmips.wasm`);
  const ct = resp.headers.get('content-type') ?? '';
  if (!resp.ok || ct.includes('text/html')) throw new Error(`wmips.wasm not found (${resp.status})`);
  const go = new g.Go();
  const bytes = await resp.arrayBuffer();
  const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
  void go.run(instance);
  const t0 = Date.now();
  while (!g.wmips) {
    if (Date.now() - t0 > timeoutMs) throw new Error('wmips global not registered');
    await new Promise((r) => setTimeout(r, 10));
  }
  return new WasmEngine(g.wmips);
}

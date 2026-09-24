// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Typed, promise-based client for the simulator worker.
import type { Method, Request, Response, WorkerApi } from './protocol';
import type { RunResult } from './types';

export interface Transport {
  send(req: Request): void;
  onMessage(cb: (res: Response) => void): void;
  terminate?(): void;
}

export const RUN_BATCH = 20000;

export class SimClient {
  private nextId = 1;
  private pending = new Map<number, { resolve: (v: unknown) => void; reject: (e: Error) => void }>();

  constructor(private readonly transport: Transport) {
    transport.onMessage((res) => {
      const p = this.pending.get(res.id);
      if (!p) return;
      this.pending.delete(res.id);
      if (res.ok) p.resolve(res.result);
      else p.reject(new Error(res.error));
    });
  }

  call<M extends Method>(method: M, ...args: Parameters<WorkerApi[M]>): Promise<ReturnType<WorkerApi[M]>> {
    const id = this.nextId++;
    return new Promise((resolve, reject) => {
      this.pending.set(id, { resolve: resolve as (v: unknown) => void, reject });
      this.transport.send({ id, method, args });
    });
  }

  /**
   * Run-to (F4): calls runTo in batches of RUN_BATCH cycles until the engine stops for a reason
   * other than "limit" or until shouldStop() returns true (Stop button).
   */
  async run(opts: { shouldStop: () => boolean; onBatch?: (r: RunResult) => void | Promise<void>; batch?: number }): Promise<RunResult> {
    const batch = opts.batch ?? RUN_BATCH;
    let total = 0;
    for (;;) {
      const r = await this.call('runTo', batch);
      total += r.cycles;
      await opts.onBatch?.(r);
      if (r.stoppedBy !== 'limit' || opts.shouldStop()) return { ...r, cycles: total };
      // Yield to the event loop so that Stop can be processed.
      await new Promise((res) => setTimeout(res, 0));
    }
  }

  terminate(): void {
    this.transport.terminate?.();
  }
}

export function workerTransport(worker: Worker): Transport {
  return {
    send: (req) => worker.postMessage(req),
    onMessage: (cb) => {
      worker.onmessage = (ev: MessageEvent<Response>) => cb(ev.data);
    },
    terminate: () => worker.terminate(),
  };
}

/** In-process transport (tests / environments without Worker support). */
export function inProcessTransport(handle: (req: Request) => Promise<Response>): Transport {
  let listener: ((res: Response) => void) | null = null;
  let queue = Promise.resolve();
  return {
    send: (req) => {
      queue = queue.then(async () => {
        const res = await handle(req);
        listener?.(res);
      });
    },
    onMessage: (cb) => {
      listener = cb;
    },
  };
}

export function createWorkerClient(): SimClient {
  const worker = new Worker(new URL('./worker.ts', import.meta.url), { type: 'module' });
  return new SimClient(workerTransport(worker));
}

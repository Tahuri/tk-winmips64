/// <reference lib="webworker" />
import { createHandler } from './handler';
import type { Request } from './protocol';

const handle = createHandler();
const scope = self as unknown as DedicatedWorkerGlobalScope;

// Requests are processed strictly in order so that e.g. step() and snapshot() never interleave.
let queue: Promise<void> = Promise.resolve();
scope.onmessage = (ev: MessageEvent<Request>) => {
  const req = ev.data;
  queue = queue.then(async () => {
    scope.postMessage(await handle(req));
  });
};

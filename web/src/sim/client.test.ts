import { describe, expect, it } from 'vitest';
import { SimClient, inProcessTransport } from './client';
import { createHandler } from './handler';
import { DEFAULT_CONFIG } from './types';
import { DEFAULT_PROGRAM } from '../lib/defaultProgram';

function mockClient() {
  const handler = createHandler(async () => {
    throw new Error('no wasm in tests');
  });
  return new SimClient(inProcessTransport(handler));
}

describe('SimClient + mock engine round trip', () => {
  it('boots falling back to the mock and reports why', async () => {
    const c = mockClient();
    const r = await c.call('boot', { mock: false, base: '/', config: DEFAULT_CONFIG });
    expect(r.kind).toBe('mock');
    expect(r.fallbackReason).toContain('no wasm');
  });

  it('assembles, steps and exposes contract-shaped snapshots', async () => {
    const c = mockClient();
    await c.call('boot', { mock: true, base: '/', config: DEFAULT_CONFIG });
    const load = await c.call('load', DEFAULT_PROGRAM);
    expect(load.ok).toBe(true);
    const prog = await c.call('program');
    expect(prog.code.length).toBe((1 << DEFAULT_CONFIG.codeBits) / 4);
    expect(prog.data.length).toBe((1 << DEFAULT_CONFIG.dataBits) / 8);
    expect(prog.code[0].word).toMatch(/^[0-9a-f]{8}$/);
    expect(prog.symbols.find((s) => s.name === 'loop')?.kind).toBe('code');

    let s = await c.call('snapshot');
    expect(s.loaded).toBe(true);
    expect(s.regs).toHaveLength(32);
    expect(s.fregs).toHaveLength(32);
    expect(s.regs[0].value).toMatch(/^[0-9a-f]{16}$/);
    expect(s.pipeline.add).toHaveLength(DEFAULT_CONFIG.addLatency);
    expect(s.pipeline.mul).toHaveLength(DEFAULT_CONFIG.mulLatency);

    for (let i = 0; i < 5; i++) await c.call('step');
    s = await c.call('snapshot');
    expect(s.cycles).toBe(5);
    expect(s.pipeline.wb.active).toBe(true);
    expect(s.pipeline.if.active).toBe(true);
    expect(s.history.length).toBeGreaterThanOrEqual(4);
    expect(s.history[0].cells.map((x) => x.stage)).toEqual(['IF', 'ID', 'EX', 'MEM', 'WB']);
    expect(s.screen.width * s.screen.height).toBe(2500);
  });

  it('runs to completion in batches, prints to the terminal and resets', async () => {
    const c = mockClient();
    await c.call('boot', { mock: true, base: '/', config: DEFAULT_CONFIG });
    await c.call('load', DEFAULT_PROGRAM);
    let batches = 0;
    const r = await c.run({ shouldStop: () => false, onBatch: () => { batches++; }, batch: 7 });
    expect(r.stoppedBy).toBe('halted');
    expect(batches).toBeGreaterThan(1);
    const s = await c.call('snapshot');
    expect(s.status).toBe('halted');
    expect(s.terminal).toBe('3\n');
    expect(s.stats.cpi).toBeGreaterThan(1);
    await c.call('reset', true);
    const s2 = await c.call('snapshot');
    expect(s2.cycles).toBe(0);
    expect(s2.terminal).toBe('');
  });

  it('reports assembler errors with codes and 1-based lines', async () => {
    const c = mockClient();
    await c.call('boot', { mock: true, base: '/', config: DEFAULT_CONFIG });
    const r = await c.call('load', '  .text\n  daddi r1, r0, 1\n  foo r1\n  j nowhere\n  .bogus\n');
    expect(r.ok).toBe(false);
    expect(r.errors?.map((e) => [e.line, e.code])).toEqual([
      [3, 'bad_instruction'],
      [4, 'undefined_symbol'],
      [5, 'bad_directive'],
    ]);
  });

  it('stops at breakpoints, edits state and waits for input', async () => {
    const c = mockClient();
    await c.call('boot', { mock: true, base: '/', config: DEFAULT_CONFIG });
    const src = `      .data
CONTROL: .word32 0x10000
DATA:    .word32 0x10008
      .text
      lwu r1, CONTROL(r0)
      daddi r2, r0, 8
      sd r2, 0(r1)
      daddi r3, r0, 5
      halt
`;
    expect((await c.call('load', src)).ok).toBe(true);
    await c.call('toggleBreakpoint', 12);
    let r = await c.call('runTo', 1000);
    expect(r.stoppedBy).toBe('breakpoint');
    r = await c.call('runTo', 1000);
    expect(r.stoppedBy).toBe('input');
    let s = await c.call('snapshot');
    expect(s.status).toBe('waiting_input');
    expect(s.inputKind).toBe('number');
    expect((await c.call('sendInput', '42')).ok).toBe(true);
    r = await c.call('runTo', 1000);
    expect(r.stoppedBy).toBe('halted');

    await c.call('setReg', 4, '00000000000000ff');
    await c.call('setFReg', 1, 2.5);
    await c.call('setMem', 0, 'ffffffffffffffff');
    s = await c.call('snapshot');
    expect(s.regs[4].value).toBe('00000000000000ff');
    expect(s.fregs[1].value).toBe(2.5);
    expect(s.data[0].value).toBe('ffffffffffffffff');
    expect(s.breakpoints).toEqual([12]);
  });

  it('rejects invalid configs and unknown methods', async () => {
    const c = mockClient();
    await c.call('boot', { mock: true, base: '/', config: DEFAULT_CONFIG });
    const bad = await c.call('setConfig', { ...DEFAULT_CONFIG, delaySlot: true, btb: true });
    expect(bad.ok).toBe(false);
    await expect((c.call as (m: string) => Promise<unknown>)('nope')).rejects.toThrow(/unknown method/);
  });
});

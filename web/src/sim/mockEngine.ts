// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// A deliberately trivial fake simulator that returns contract-shaped data (CONTRACT §4/§5).
// It is used only when /wasm/wmips.wasm cannot be loaded or when ?mock=1 is set.
//
// Simplifications (by design, this is NOT the reference behaviour):
// - every data directive value occupies its own 64-bit word;
// - branches and jumps are not taken (execution is linear until `halt`);
// - only a handful of instructions have semantics (daddi/dadd/dsub/and/or/ld/sd/l.d/s.d/add.d/…);
// - I/O: a store to 0x10000 (CONTROL) triggers the terminal functions using the word at 0x10008 (DATA).

import { bigToHex16, bytesToBase64, doubleToHex16, hex, hex16ToBig, hex16ToDouble } from '../lib/format';
import type {
  AsmError,
  CodeLine,
  Config,
  DataLine,
  DataWord,
  Engine,
  HistCell,
  HistEntry,
  LoadResult,
  Message,
  OkResult,
  PipeView,
  Program,
  RegView,
  FRegView,
  RunResult,
  SimStatus,
  Snapshot,
  StageSlot,
  StepResult,
  SymbolInfo,
} from './types';
import { DEFAULT_CONFIG, CONFIG_LIMITS } from './types';
import { ABI_NAMES } from '../lib/regs';

export { ABI_NAMES };

const KNOWN = new Set(
  (
    'lb lbu sb lh lhu sh lw lwu sw ld sd l.d s.d halt daddi daddui daddiu andi ori xori lui slti sltiu ' +
    'beq bne beqz bnez j jr jal jalr dsll dsrl dsra dsllv dsrlv dsrav movz movn nop and or xor slt sltu ' +
    'dadd daddu dsub dsubu dmul dmulu ddiv ddivu add.d sub.d mul.d div.d mov.d cvt.d.l cvt.l.d c.lt.d ' +
    'c.le.d c.eq.d bc1f bc1t mtc1 mfc1 dmult dmultu mflo mfhi dsub.d'
  ).split(/\s+/),
);

const DIRECTIVES = new Set([
  '.data', '.text', '.code', '.org', '.space', '.asciiz', '.ascii', '.align',
  '.word', '.byte', '.word32', '.word16', '.double',
]);

const CONTROL_ADDR = 0x10000;
const DATA_ADDR = 0x10008;
const SCREEN = 50;

interface Instr {
  addr: number;
  text: string; // instruction text without label/comment
  op: string;
  args: string[];
  line: number;
  raw: string;
}

interface InFlight {
  instr: Instr;
  stages: string[];
  idx: number; // index into stages of current stage
  entry: HistEntry;
}

function stripComment(line: string): string {
  const i = line.indexOf(';');
  return i >= 0 ? line.slice(0, i) : line;
}

function parseRegName(tok: string): { kind: 'r' | 'f'; n: number } | null {
  const t = tok.trim().toLowerCase().replace(/^\$/, '');
  let m = /^r(\d+)$/.exec(t);
  if (m) return +m[1] < 32 ? { kind: 'r', n: +m[1] } : null;
  m = /^f(\d+)$/.exec(t);
  if (m) return +m[1] < 32 ? { kind: 'f', n: +m[1] } : null;
  const abi = ABI_NAMES.indexOf(t);
  if (abi >= 0) return { kind: 'r', n: abi };
  return null;
}

function regKey(r: { kind: 'r' | 'f'; n: number }): string {
  return `${r.kind}${r.n}`;
}

const STORES = new Set(['sb', 'sh', 'sw', 'sd', 's.d']);
const LOADS = new Set(['lb', 'lbu', 'lh', 'lhu', 'lw', 'lwu', 'ld', 'l.d']);
const NO_DEST = new Set([...STORES, 'beq', 'bne', 'beqz', 'bnez', 'j', 'jr', 'halt', 'nop', 'bc1f', 'bc1t', 'c.lt.d', 'c.le.d', 'c.eq.d']);

function operandRegs(instr: Instr): { dest: string | null; srcs: string[] } {
  const regs = instr.args.map((a) => {
    const m = /\((.+)\)$/.exec(a);
    return parseRegName(m ? m[1] : a);
  });
  const present = regs.filter((r): r is { kind: 'r' | 'f'; n: number } => r !== null).map(regKey);
  if (NO_DEST.has(instr.op) || present.length === 0) return { dest: null, srcs: present };
  const first = regs[0];
  if (!first) return { dest: null, srcs: present };
  const dest = regKey(first);
  return { dest: dest === 'r0' ? null : dest, srcs: present.slice(1) };
}

function emptySlot(): StageSlot {
  return { active: false, addr: 0, mnemonic: '' };
}

export class MockEngine implements Engine {
  readonly kind = 'mock' as const;
  private cfg: Config = { ...DEFAULT_CONFIG };
  private loaded = false;
  private prog: Program = { codeSize: 0, dataSize: 0, code: [], data: [], symbols: [] };
  private instrs = new Map<number, Instr>();
  private symbols = new Map<string, SymbolInfo>();
  private dataImage: bigint[] = [];
  private dataWrittenImage: boolean[] = [];
  private mem: bigint[] = [];
  private written: boolean[] = [];
  private regs: bigint[] = new Array(32).fill(0n);
  private fregs: bigint[] = new Array(32).fill(0n);
  private breakpoints = new Set<number>();
  private status: SimStatus = 'idle';
  private inputKind: 'number' | 'char' | undefined;
  private cycles = 0;
  private instructions = 0;
  private loads = 0;
  private stores = 0;
  private rawStalls = 0;
  private structuralStalls = 0;
  private pc = 0;
  private fetchStopped = false;
  private inflight: InFlight[] = [];
  private history: HistEntry[] = [];
  private lastMessages: Message[] = [];
  private terminal = '';
  private screen = new Uint8Array(SCREEN * SCREEN * 3);
  private screenDrawn = false;

  init(cfg: Config): OkResult {
    return this.setConfig(cfg);
  }

  setConfig(cfg: Config): OkResult {
    for (const [k, [lo, hi]] of Object.entries(CONFIG_LIMITS)) {
      const v = cfg[k as keyof typeof CONFIG_LIMITS];
      if (typeof v !== 'number' || v < lo || v > hi) return { ok: false, error: `${k} out of range` };
    }
    if (cfg.delaySlot && cfg.btb) return { ok: false, error: 'delay slot and BTB are mutually exclusive' };
    this.cfg = { ...cfg };
    this.loaded = false;
    this.status = 'idle';
    return { ok: true };
  }

  load(src: string): LoadResult {
    const errors: AsmError[] = [];
    const codeWords = (1 << this.cfg.codeBits) / 4;
    const dataWords = (1 << this.cfg.dataBits) / 8;
    const codeText: string[] = new Array(codeWords).fill('');
    const dataText: string[] = new Array(dataWords).fill('');
    const codeLabels: string[] = new Array(codeWords).fill('');
    const dataLabels: string[] = new Array(dataWords).fill('');
    const image: bigint[] = new Array(dataWords).fill(0n);
    const imageWritten: boolean[] = new Array(dataWords).fill(false);
    const instrs = new Map<number, Instr>();
    const symbols = new Map<string, SymbolInfo>();
    let section: 'code' | 'data' = 'code';
    let codePtr = 0;
    let dataPtr = 0; // word index
    const pendingData: { line: number; text: string; tokens: string[]; index: number }[] = [];

    const lines = src.replace(/\r/g, '').split('\n');
    lines.forEach((raw, i) => {
      const lineNo = i + 1;
      let body = stripComment(raw).trim();
      if (!body) return;
      let label = '';
      const lm = /^([A-Za-z_.$][\w.$]*)\s*:\s*(.*)$/.exec(body);
      if (lm) {
        label = lm[1];
        body = lm[2].trim();
      }
      if (label) {
        if (symbols.has(label.toLowerCase())) {
          errors.push({ line: lineNo, code: 'duplicate_symbol', text: raw });
        } else {
          const addr = section === 'code' ? codePtr * 4 : dataPtr * 8;
          symbols.set(label.toLowerCase(), { name: label, addr, kind: section });
          if (section === 'code' && codePtr < codeWords) codeLabels[codePtr] = label;
          if (section === 'data' && dataPtr < dataWords) dataLabels[dataPtr] = label;
        }
      }
      if (!body) return;
      const sp = body.search(/\s/);
      const head = (sp < 0 ? body : body.slice(0, sp)).toLowerCase();
      const rest = sp < 0 ? '' : body.slice(sp + 1).trim();
      if (head.startsWith('.')) {
        if (!DIRECTIVES.has(head)) {
          errors.push({ line: lineNo, code: 'bad_directive', text: raw });
          return;
        }
        if (head === '.data') { section = 'data'; return; }
        if (head === '.text' || head === '.code') { section = 'code'; return; }
        if (head === '.org' || head === '.align') return;
        if (section !== 'data') {
          errors.push({ line: lineNo, code: 'syntax', text: raw });
          return;
        }
        let count = 1;
        let tokens: string[] = [];
        if (head === '.space') {
          const n = Number(rest);
          if (!Number.isInteger(n) || n < 0) { errors.push({ line: lineNo, code: 'bad_number', text: raw }); return; }
          count = Math.max(1, Math.ceil(n / 8));
        } else if (head === '.asciiz' || head === '.ascii') {
          const s = rest.replace(/^"|"$/g, '');
          count = Math.max(1, Math.ceil((s.length + (head === '.asciiz' ? 1 : 0)) / 8));
        } else {
          tokens = rest.split(',').map((t) => t.trim()).filter(Boolean);
          count = Math.max(1, tokens.length);
        }
        if (dataPtr + count > dataWords) { errors.push({ line: lineNo, code: 'out_of_memory', text: raw }); return; }
        dataText[dataPtr] = raw.trim();
        for (let k = 0; k < count; k++) {
          const tok = tokens[k];
          if (tok !== undefined) pendingData.push({ line: lineNo, text: raw, tokens: [head, tok], index: dataPtr + k });
          if (head === '.asciiz' || head === '.ascii') {
            const s = rest.replace(/^"|"$/g, '');
            let v = 0n;
            for (let b = 7; b >= 0; b--) v = (v << 8n) | BigInt(s.charCodeAt(k * 8 + b) || 0);
            image[dataPtr + k] = v;
          }
          imageWritten[dataPtr + k] = true;
        }
        dataPtr += count;
        return;
      }
      if (section !== 'code') {
        errors.push({ line: lineNo, code: 'syntax', text: raw });
        return;
      }
      if (!KNOWN.has(head)) {
        errors.push({ line: lineNo, code: 'bad_instruction', text: raw });
        return;
      }
      if (codePtr >= codeWords) { errors.push({ line: lineNo, code: 'out_of_memory', text: raw }); return; }
      const args = rest ? rest.split(',').map((a) => a.trim()) : [];
      for (const a of args) {
        const inner = /\((.+)\)$/.exec(a);
        if (inner && !parseRegName(inner[1])) {
          errors.push({ line: lineNo, code: 'bad_register', text: raw });
          return;
        }
      }
      instrs.set(codePtr * 4, { addr: codePtr * 4, text: body.replace(/\s+/g, ' '), op: head, args, line: lineNo, raw });
      codeText[codePtr] = raw.replace(/\t/g, '    ').trimEnd();
      codePtr++;
    });

    // second pass: resolve data values (labels allowed)
    for (const p of pendingData) {
      const [dir, tok] = p.tokens;
      if (dir === '.double') {
        const f = Number(tok);
        if (Number.isNaN(f)) { errors.push({ line: p.line, code: 'bad_number', text: p.text }); continue; }
        image[p.index] = hex16ToBig(doubleToHex16(f));
      } else {
        const v = this.parseValue(tok, symbols);
        if (v === null) { errors.push({ line: p.line, code: /^[A-Za-z_]/.test(tok) ? 'undefined_symbol' : 'bad_number', text: p.text }); continue; }
        image[p.index] = BigInt.asUintN(64, v);
      }
    }
    // check label references in code
    for (const ins of instrs.values()) {
      for (const a of ins.args) {
        const base = a.replace(/\(.+\)$/, '').trim();
        if (!base || parseRegName(base)) continue;
        if (this.parseValue(base, symbols) === null) {
          errors.push({ line: ins.line, code: 'undefined_symbol', text: ins.raw });
        }
      }
    }

    if (errors.length > 0) {
      errors.sort((a, b) => a.line - b.line);
      this.loaded = false;
      this.status = 'idle';
      return { ok: false, errors };
    }

    const code: CodeLine[] = [];
    for (let w = 0; w < codeWords; w++) {
      const ins = instrs.get(w * 4);
      code.push({
        addr: w * 4,
        word: ins ? hex(fakeEncode(ins.text), 8) : '00000000',
        text: codeText[w],
        label: codeLabels[w] || undefined,
        used: !!ins,
      });
    }
    const data: DataLine[] = [];
    for (let w = 0; w < dataWords; w++) {
      data.push({ addr: w * 8, text: dataText[w], label: dataLabels[w] || undefined });
    }
    this.prog = {
      codeSize: codePtr * 4,
      dataSize: dataPtr * 8,
      code,
      data,
      symbols: [...symbols.values()],
    };
    this.instrs = instrs;
    this.symbols = symbols;
    this.dataImage = image;
    this.dataWrittenImage = imageWritten;
    this.breakpoints.clear();
    this.loaded = true;
    this.reset(true);
    return { ok: true, errors: [] };
  }

  private parseValue(tok: string, symbols: Map<string, SymbolInfo>): bigint | null {
    const t = tok.trim();
    const m = /^([A-Za-z_.$][\w.$]*)\s*([+-]\s*\d+)?$/.exec(t);
    if (m) {
      const sym = symbols.get(m[1].toLowerCase());
      if (!sym) return null;
      return BigInt(sym.addr) + (m[2] ? BigInt(m[2].replace(/\s/g, '')) : 0n);
    }
    try {
      if (/^-?0x[0-9a-f]+$/i.test(t)) return t.startsWith('-') ? -BigInt(t.slice(1)) : BigInt(t);
      if (/^[-+]?\d+$/.test(t)) return BigInt(t);
    } catch {
      return null;
    }
    return null;
  }

  program(): Program {
    return this.prog;
  }

  reset(full: boolean): OkResult {
    this.regs = new Array(32).fill(0n);
    this.fregs = new Array(32).fill(0n);
    if (full || this.mem.length !== this.dataImage.length) {
      this.mem = [...this.dataImage];
      this.written = [...this.dataWrittenImage];
    }
    this.cycles = 0;
    this.instructions = 0;
    this.loads = 0;
    this.stores = 0;
    this.rawStalls = 0;
    this.structuralStalls = 0;
    this.pc = 0;
    this.fetchStopped = false;
    this.inflight = [];
    this.history = [];
    this.lastMessages = [];
    this.terminal = '';
    this.screen = new Uint8Array(SCREEN * SCREEN * 3);
    this.screenDrawn = false;
    this.inputKind = undefined;
    this.status = this.loaded ? 'ok' : 'idle';
    return { ok: true };
  }

  private stagesFor(ins: Instr): string[] {
    const ex: string[] = [];
    if (ins.op === 'add.d' || ins.op === 'sub.d') {
      for (let i = 0; i < this.cfg.addLatency; i++) ex.push(`A${i}`);
    } else if (ins.op === 'mul.d') {
      for (let i = 0; i < this.cfg.mulLatency; i++) ex.push(`M${i}`);
    } else if (ins.op === 'div.d') {
      for (let i = 0; i < this.cfg.divLatency; i++) ex.push('DIV');
    } else ex.push('EX');
    return ['IF', 'ID', ...ex, 'MEM', 'WB'];
  }

  private cycle(): void {
    if (!this.loaded || this.status === 'halted' || this.status === 'waiting_input') return;
    this.cycles++;
    const messages: Message[] = [];
    const occupied = new Map<string, InFlight>();
    const survivors: InFlight[] = [];
    let halted = false;

    // Oldest first: try to advance each in-flight instruction.
    for (let k = 0; k < this.inflight.length; k++) {
      const f = this.inflight[k];
      const cur = f.stages[f.idx];
      if (cur === 'WB') {
        this.retire(f);
        if (f.instr.op === 'halt') halted = true;
        continue;
      }
      const next = f.stages[f.idx + 1];
      let cause = '';
      if (cur === 'ID') {
        const hazard = this.rawHazard(f, k);
        if (hazard) {
          cause = 'raw';
          this.rawStalls++;
          messages.push({ code: 'raw_stall', stage: 'ID', reg: hazard.toUpperCase() });
        }
      }
      const unitBusy = next !== 'DIV' || cur !== 'DIV' ? occupied.has(next) : false;
      if (!cause && unitBusy) {
        cause = 'structural';
        this.structuralStalls++;
        messages.push({ code: 'structural_stall', stage: cur === 'ID' ? 'EX' : stageName(next) });
      }
      if (!cause) {
        f.idx++;
        f.entry.cells.push({ stage: f.stages[f.idx] });
      } else {
        f.entry.cells.push({ stage: cur, cause });
      }
      occupied.set(f.stages[f.idx], f);
      survivors.push(f);
    }
    this.inflight = survivors;

    // Fetch a new instruction when IF is free.
    if (!this.fetchStopped && !halted && !occupied.has('IF') && !this.isWaiting()) {
      const ins = this.instrs.get(this.pc);
      if (!ins) {
        this.fetchStopped = true;
        if (this.inflight.length === 0) messages.push({ code: 'no_such_code_memory' });
      } else {
        const entry: HistEntry = { addr: ins.addr, mnemonic: ins.text, startCycle: this.cycles, cells: [{ stage: 'IF' }] };
        this.history.push(entry);
        this.inflight.push({ instr: ins, stages: this.stagesFor(ins), idx: 0, entry });
        if (ins.op === 'halt') this.fetchStopped = true;
        this.pc += 4;
      }
    }

    if (halted) this.status = 'halted';
    else if (this.isWaiting()) messages.push({ code: 'waiting_input' });
    else if (this.inflight.length === 0 && this.fetchStopped) this.status = 'halted';
    this.lastMessages = messages;
  }

  private isWaiting(): boolean {
    return this.status === 'waiting_input';
  }

  private rawHazard(f: InFlight, k: number): string | null {
    const { srcs } = operandRegs(f.instr);
    for (let j = 0; j < k; j++) {
      const older = this.inflight[j];
      const { dest } = operandRegs(older.instr);
      if (!dest || !srcs.includes(dest)) continue;
      const st = older.stages[older.idx];
      const stillComputing = st === 'ID' || st === 'EX' || st.startsWith('A') || st.startsWith('M') && st !== 'MEM' || st === 'DIV';
      const loadInEx = LOADS.has(older.instr.op) && st === 'EX';
      if (stillComputing || loadInEx || (!this.cfg.forwarding && st === 'MEM')) return dest;
    }
    return null;
  }

  private effAddr(arg: string): number {
    const m = /^(.*)\((.+)\)$/.exec(arg.trim());
    const off = m ? m[1].trim() : arg.trim();
    const base = m ? parseRegName(m[2]) : null;
    const offVal = off ? this.parseValue(off, this.symbols) ?? 0n : 0n;
    const baseVal = base ? this.regs[base.n] : 0n;
    return Number(BigInt.asUintN(32, offVal + baseVal));
  }

  private reg(tok: string | undefined): bigint {
    const r = tok ? parseRegName(tok) : null;
    if (!r) return 0n;
    return r.kind === 'r' ? this.regs[r.n] : this.fregs[r.n];
  }

  private setDest(tok: string | undefined, v: bigint): void {
    const r = tok ? parseRegName(tok) : null;
    if (!r) return;
    if (r.kind === 'r') {
      if (r.n !== 0) this.regs[r.n] = BigInt.asUintN(64, v);
    } else this.fregs[r.n] = BigInt.asUintN(64, v);
  }

  private fval(tok: string | undefined): number {
    return hex16ToDouble(bigToHex16(this.reg(tok)));
  }

  private setF(tok: string | undefined, f: number): void {
    this.setDest(tok, hex16ToBig(doubleToHex16(f)));
  }

  private retire(f: InFlight): void {
    this.instructions++;
    const { op, args } = f.instr;
    const imm = (tok: string | undefined) => (tok ? this.parseValue(tok, this.symbols) ?? 0n : 0n);
    switch (op) {
      case 'daddi': case 'daddiu': case 'daddui':
        this.setDest(args[0], this.reg(args[1]) + imm(args[2])); break;
      case 'andi': this.setDest(args[0], this.reg(args[1]) & imm(args[2])); break;
      case 'ori': this.setDest(args[0], this.reg(args[1]) | imm(args[2])); break;
      case 'dadd': case 'daddu': this.setDest(args[0], this.reg(args[1]) + this.reg(args[2])); break;
      case 'dsub': case 'dsubu': this.setDest(args[0], this.reg(args[1]) - this.reg(args[2])); break;
      case 'and': this.setDest(args[0], this.reg(args[1]) & this.reg(args[2])); break;
      case 'or': this.setDest(args[0], this.reg(args[1]) | this.reg(args[2])); break;
      case 'xor': this.setDest(args[0], this.reg(args[1]) ^ this.reg(args[2])); break;
      case 'add.d': this.setF(args[0], this.fval(args[1]) + this.fval(args[2])); break;
      case 'sub.d': this.setF(args[0], this.fval(args[1]) - this.fval(args[2])); break;
      case 'mul.d': this.setF(args[0], this.fval(args[1]) * this.fval(args[2])); break;
      case 'div.d': this.setF(args[0], this.fval(args[1]) / this.fval(args[2])); break;
      case 'mov.d': this.setDest(args[0], this.reg(args[1])); break;
      default:
        if (LOADS.has(op)) {
          this.loads++;
          const a = this.effAddr(args[1] ?? '');
          const w = this.mem[a >> 3] ?? 0n;
          this.setDest(args[0], op === 'lwu' || op === 'lw' ? w & 0xffffffffn : w);
        } else if (STORES.has(op)) {
          this.stores++;
          const a = this.effAddr(args[1] ?? '');
          const v = this.reg(args[0]);
          if (a === CONTROL_ADDR) this.control(Number(v & 0xffn));
          else if (a === DATA_ADDR) this.ioData = v;
          else if ((a >> 3) < this.mem.length) {
            this.mem[a >> 3] = v;
            this.written[a >> 3] = true;
          }
        }
    }
  }

  private ioData = 0n;

  private control(fn: number): void {
    const v = this.ioData;
    switch (fn) {
      case 1: this.terminal += BigInt.asUintN(64, v).toString() + '\n'; break;
      case 2: this.terminal += BigInt.asIntN(64, v).toString() + '\n'; break;
      case 3: this.terminal += hex16ToDouble(bigToHex16(v)).toString() + '\n'; break;
      case 5: {
        const x = Number((v >> 40n) & 0xffn);
        const y = Number((v >> 32n) & 0xffn);
        if (x < SCREEN && y < SCREEN) {
          const p = ((SCREEN - 1 - y) * SCREEN + x) * 3;
          this.screen[p] = Number(v & 0xffn);
          this.screen[p + 1] = Number((v >> 8n) & 0xffn);
          this.screen[p + 2] = Number((v >> 16n) & 0xffn);
          this.screenDrawn = true;
        }
        break;
      }
      case 6: this.terminal = ''; break;
      case 7: this.screen.fill(0); this.screenDrawn = false; break;
      case 8: this.status = 'waiting_input'; this.inputKind = 'number'; break;
      case 9: this.status = 'waiting_input'; this.inputKind = 'char'; break;
      default: break;
    }
  }

  step(): StepResult {
    this.cycle();
    return this.stepResult();
  }

  stepN(n: number): StepResult {
    for (let i = 0; i < n; i++) {
      this.cycle();
      if (this.status !== 'ok') break;
    }
    return this.stepResult();
  }

  runTo(maxCycles: number): RunResult {
    const start = this.cycles;
    let stoppedBy: RunResult['stoppedBy'] = 'limit';
    if (!this.loaded) return { cycles: 0, status: 'ok', stoppedBy: 'error', messages: [] };
    for (let i = 0; i < maxCycles; i++) {
      this.cycle();
      if (this.status === 'halted') { stoppedBy = 'halted'; break; }
      if (this.status === 'waiting_input') { stoppedBy = 'input'; break; }
      const fetched = this.inflight.find((f) => f.idx === 0);
      if (i > 0 && fetched && this.breakpoints.has(fetched.instr.addr)) { stoppedBy = 'breakpoint'; break; }
    }
    const r = this.stepResult();
    return { cycles: this.cycles - start, status: r.status, stoppedBy, messages: r.messages };
  }

  private stepResult(): StepResult {
    const status = this.status === 'idle' ? 'ok' : this.status;
    return { status, messages: this.lastMessages };
  }

  sendInput(text: string): OkResult {
    if (this.status !== 'waiting_input') return { ok: false, error: 'not waiting for input' };
    if (this.inputKind === 'char') {
      this.ioData = BigInt(text.charCodeAt(0) || 0);
    } else {
      const t = text.trim();
      if (t.includes('.')) {
        const f = Number(t);
        if (Number.isNaN(f)) return { ok: false, error: 'bad number' };
        this.ioData = hex16ToBig(doubleToHex16(f));
      } else {
        if (!/^[-+]?\d+$/.test(t)) return { ok: false, error: 'bad number' };
        this.ioData = BigInt.asUintN(64, BigInt(t));
      }
    }
    this.terminal += text + '\n';
    this.status = 'ok';
    this.inputKind = undefined;
    return { ok: true };
  }

  toggleBreakpoint(addr: number): OkResult {
    if (this.breakpoints.has(addr)) this.breakpoints.delete(addr);
    else this.breakpoints.add(addr);
    return { ok: true };
  }

  setReg(i: number, h: string): OkResult {
    if (i < 1 || i > 31) return { ok: false };
    this.regs[i] = hex16ToBig(h);
    return { ok: true };
  }

  setFReg(i: number, f: number): OkResult {
    if (i < 0 || i > 31) return { ok: false };
    this.fregs[i] = hex16ToBig(doubleToHex16(f));
    return { ok: true };
  }

  setMem(addr: number, h: string): OkResult {
    const w = addr >> 3;
    if (addr % 8 !== 0 || w >= this.mem.length) return { ok: false };
    this.mem[w] = hex16ToBig(h);
    this.written[w] = true;
    return { ok: true };
  }

  setMemDouble(addr: number, f: number): OkResult {
    return this.setMem(addr, doubleToHex16(f));
  }

  private sourceOf(kind: 'r' | 'f', n: number): string {
    for (const f of this.inflight) {
      const { dest } = operandRegs(f.instr);
      if (dest !== `${kind}${n}`) continue;
      const st = f.stages[f.idx];
      if (st === 'EX') return 'ex';
      if (st === 'MEM') return 'mem';
      if (st.startsWith('A')) return 'add';
      if (st.startsWith('M')) return 'mul';
      if (st === 'DIV') return 'div';
      if (st === 'ID') return 'id';
    }
    return 'reg';
  }

  private pipeView(): PipeView {
    const view: PipeView = {
      if: emptySlot(), id: emptySlot(), ex: emptySlot(),
      add: Array.from({ length: this.cfg.addLatency }, emptySlot),
      mul: Array.from({ length: this.cfg.mulLatency }, emptySlot),
      div: emptySlot(), mem: emptySlot(), wb: emptySlot(),
    };
    for (const f of this.inflight) {
      const st = f.stages[f.idx];
      const slot: StageSlot = { active: true, addr: f.instr.addr, mnemonic: f.instr.op };
      if (st === 'IF') view.if = slot;
      else if (st === 'ID') view.id = slot;
      else if (st === 'EX') view.ex = slot;
      else if (st === 'MEM') view.mem = slot;
      else if (st === 'WB') view.wb = slot;
      else if (st === 'DIV') view.div = slot;
      else if (st.startsWith('A')) view.add[+st.slice(1)] = slot;
      else if (st.startsWith('M')) view.mul[+st.slice(1)] = slot;
    }
    return view;
  }

  snapshot(): Snapshot {
    const regs: RegView[] = this.regs.map((v, i) => ({
      name: this.cfg.registersAsNumbers ? `R${i}` : ABI_NAMES[i],
      value: bigToHex16(v),
      source: this.loaded ? this.sourceOf('r', i) : 'reg',
    }));
    const fregs: FRegView[] = this.fregs.map((v, i) => {
      const bits = bigToHex16(v);
      return { name: `F${i}`, bits, value: hex16ToDouble(bits), source: this.loaded ? this.sourceOf('f', i) : 'reg' };
    });
    const data: DataWord[] = this.mem.map((v, w) => {
      const value = bigToHex16(v);
      const line = this.prog.data[w];
      return {
        addr: w * 8,
        value,
        double: hex16ToDouble(value),
        written: this.written[w] ?? false,
        label: line?.label,
        text: line?.text ?? '',
      };
    });
    const predicted: number[] = [];
    return {
      config: { ...this.cfg },
      loaded: this.loaded,
      status: this.status,
      inputKind: this.inputKind,
      cycles: this.cycles,
      instructions: this.instructions,
      stats: {
        loads: this.loads,
        stores: this.stores,
        rawStalls: this.rawStalls,
        wawStalls: 0,
        warStalls: 0,
        structuralStalls: this.structuralStalls,
        branchTakenStalls: 0,
        branchMispredictionStalls: 0,
        codeSize: this.prog.codeSize,
        dataSize: this.prog.dataSize,
        cpi: this.instructions ? this.cycles / this.instructions : 0,
      },
      messages: this.lastMessages,
      pc: this.pc,
      regs,
      fregs,
      fpcc: false,
      breakpoints: [...this.breakpoints].sort((a, b) => a - b),
      predicted,
      pipeline: this.pipeView(),
      data,
      history: this.history.map((h) => ({ ...h, cells: h.cells.map((c: HistCell) => ({ ...c })) })),
      terminal: this.terminal,
      screen: { width: SCREEN, height: SCREEN, drawn: this.screenDrawn, pixels: bytesToBase64(this.screen) },
    };
  }
}

function stageName(s: string): string {
  if (s.startsWith('A')) return 'FP-ADD';
  if (s.startsWith('M') && s !== 'MEM') return 'FP-MUL';
  if (s === 'DIV') return 'FP-DIV';
  return s;
}

function fakeEncode(text: string): number {
  // FNV-1a of the instruction text: stable, looks like a machine word.
  let h = 0x811c9dc5;
  for (let i = 0; i < text.length; i++) {
    h ^= text.charCodeAt(i);
    h = Math.imul(h, 0x01000193);
  }
  return h >>> 0;
}

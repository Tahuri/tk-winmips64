// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Mirrors docs/CONTRACT.md §2–§5. 64-bit values travel as 16-hex-digit strings.

export interface Config {
  codeBits: number;
  dataBits: number;
  addLatency: number;
  mulLatency: number;
  divLatency: number;
  forwarding: boolean;
  delaySlot: boolean;
  btb: boolean;
  registersAsNumbers: boolean;
}

export const DEFAULT_CONFIG: Config = {
  codeBits: 10,
  dataBits: 10,
  addLatency: 4,
  mulLatency: 7,
  divLatency: 24,
  forwarding: true,
  delaySlot: false,
  btb: false,
  registersAsNumbers: false,
};

export const CONFIG_LIMITS = {
  codeBits: [8, 13],
  dataBits: [4, 11],
  addLatency: [2, 8],
  mulLatency: [2, 8],
  divLatency: [10, 30],
} as const;

export type MessageCode =
  | 'raw_stall'
  | 'waw_stall'
  | 'war_stall'
  | 'branch_taken_stall'
  | 'branch_mispredicted_stall'
  | 'structural_stall'
  | 'no_such_code_memory'
  | 'integer_overflow'
  | 'divide_by_zero'
  | 'uninitialized_memory'
  | 'no_such_data_memory'
  | 'data_misaligned'
  | 'waiting_input';

export const MESSAGE_CODES: readonly MessageCode[] = [
  'raw_stall',
  'waw_stall',
  'war_stall',
  'branch_taken_stall',
  'branch_mispredicted_stall',
  'structural_stall',
  'no_such_code_memory',
  'integer_overflow',
  'divide_by_zero',
  'uninitialized_memory',
  'no_such_data_memory',
  'data_misaligned',
  'waiting_input',
];

export interface Message {
  code: MessageCode | string;
  stage?: string;
  reg?: string;
}

export type AsmErrorCode =
  | 'undefined_symbol'
  | 'bad_instruction'
  | 'bad_directive'
  | 'out_of_memory'
  | 'bad_register'
  | 'bad_number'
  | 'duplicate_symbol'
  | 'syntax';

export const ASM_ERROR_CODES: readonly AsmErrorCode[] = [
  'undefined_symbol',
  'bad_instruction',
  'bad_directive',
  'out_of_memory',
  'bad_register',
  'bad_number',
  'duplicate_symbol',
  'syntax',
];

export interface AsmError {
  line: number; // 1-based
  code: AsmErrorCode | string;
  text: string;
}

export interface CodeLine {
  addr: number;
  word: string; // 8 hex
  text: string;
  label?: string;
  used: boolean;
}

export interface DataLine {
  addr: number;
  text: string;
  label?: string;
}

export interface SymbolInfo {
  name: string;
  addr: number;
  kind: 'code' | 'data';
}

export interface Program {
  codeSize: number;
  dataSize: number;
  code: CodeLine[];
  data: DataLine[];
  symbols: SymbolInfo[];
}

export type SimStatus = 'idle' | 'ok' | 'halted' | 'waiting_input';

export interface Stats {
  loads: number;
  stores: number;
  rawStalls: number;
  wawStalls: number;
  warStalls: number;
  structuralStalls: number;
  branchTakenStalls: number;
  branchMispredictionStalls: number;
  codeSize: number;
  dataSize: number;
  cpi: number;
}

export type RegSource = 'reg' | 'id' | 'ex' | 'mem' | 'add' | 'mul' | 'div' | 'na';

export interface RegView {
  name: string;
  value: string; // 16 hex
  source: RegSource | string;
}

export interface FRegView {
  name: string;
  bits: string; // 16 hex
  value: number;
  source: RegSource | string;
}

export interface StageSlot {
  active: boolean;
  addr: number;
  mnemonic: string;
}

export interface PipeView {
  if: StageSlot;
  id: StageSlot;
  ex: StageSlot;
  add: StageSlot[];
  mul: StageSlot[];
  div: StageSlot;
  mem: StageSlot;
  wb: StageSlot;
}

export interface DataWord {
  addr: number;
  value: string; // 16 hex
  double: number;
  written: boolean;
  label?: string;
  text: string;
}

export type HistCause = 'raw' | 'waw' | 'war' | 'structural' | 'branch_taken' | 'branch_mispredicted' | '';

export interface HistCell {
  stage: string; // "IF","ID","EX","A0".."A7","M0".."M7","DIV","MEM","WB",""
  cause?: HistCause | string;
}

export interface HistEntry {
  addr: number;
  mnemonic: string;
  startCycle: number;
  cells: HistCell[];
}

export interface ScreenView {
  width: number;
  height: number;
  drawn: boolean;
  pixels: string; // base64 of width*height*3 RGB bytes
}

export interface Snapshot {
  config: Config;
  loaded: boolean;
  status: SimStatus;
  inputKind?: 'number' | 'char';
  cycles: number;
  instructions: number;
  stats: Stats;
  messages: Message[];
  pc: number;
  regs: RegView[];
  fregs: FRegView[];
  fpcc: boolean;
  breakpoints: number[];
  predicted: number[];
  pipeline: PipeView;
  data: DataWord[];
  history: HistEntry[];
  terminal: string;
  screen: ScreenView;
}

export type StepStatus = 'ok' | 'halted' | 'waiting_input';
export type StoppedBy = 'breakpoint' | 'halted' | 'input' | 'limit' | 'error';

export interface StepResult {
  status: StepStatus;
  messages: Message[] | null;
}

export interface RunResult {
  cycles: number;
  status: StepStatus;
  stoppedBy: StoppedBy;
  messages: Message[] | null;
}

export interface OkResult {
  ok: boolean;
  error?: string;
}

export interface LoadResult {
  ok: boolean;
  errors: AsmError[] | null;
}

/** Engine abstraction implemented by the WASM bridge adapter and the TS mock. */
export interface Engine {
  readonly kind: 'wasm' | 'mock';
  init(cfg: Config): OkResult;
  setConfig(cfg: Config): OkResult;
  load(src: string): LoadResult;
  program(): Program;
  snapshot(): Snapshot;
  step(): StepResult;
  stepN(n: number): StepResult;
  runTo(maxCycles: number): RunResult;
  reset(full: boolean): OkResult;
  sendInput(text: string): OkResult;
  toggleBreakpoint(addr: number): OkResult;
  setReg(i: number, hex: string): OkResult;
  setFReg(i: number, f: number): OkResult;
  setMem(addr: number, hex: string): OkResult;
  setMemDouble(addr: number, f: number): OkResult;
}

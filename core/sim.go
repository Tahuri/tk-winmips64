// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0
//
// Derived from WinMIPS64 by Mike Scott (Apache-2.0,
// https://github.com/mcarrickscott/WinMIPS64) and the fork by Andoni
// Zubimendi (https://github.com/AndoniZubimendi/WinMIPS64).
// Modified: ported from C++/MFC to Go. See NOTICE.

package core

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Config holds the simulator architecture/configuration (winmips64.ini).
type Config struct {
	CodeBits           int  `json:"codeBits"`
	DataBits           int  `json:"dataBits"`
	AddLatency         int  `json:"addLatency"`
	MulLatency         int  `json:"mulLatency"`
	DivLatency         int  `json:"divLatency"`
	Forwarding         bool `json:"forwarding"`
	DelaySlot          bool `json:"delaySlot"`
	BTB                bool `json:"btb"`
	RegistersAsNumbers bool `json:"registersAsNumbers"`
}

// DefaultConfig returns the constructor defaults of CWinMIPS64Doc.
func DefaultConfig() Config {
	return Config{CodeBits: 10, DataBits: 10, AddLatency: 4, MulLatency: 7, DivLatency: 24, Forwarding: true}
}

// Validate checks the ranges enforced by the original dialogs.
func (c Config) Validate() error {
	switch {
	case c.CodeBits < minCodeBits || c.CodeBits > maxCodeBits:
		return fmt.Errorf("codeBits must be in %d..%d", minCodeBits, maxCodeBits)
	case c.DataBits < minDataBits || c.DataBits > maxDataBits:
		return fmt.Errorf("dataBits must be in %d..%d", minDataBits, maxDataBits)
	case c.AddLatency < minAddLatency || c.AddLatency > maxAddLatency:
		return fmt.Errorf("addLatency must be in %d..%d", minAddLatency, maxAddLatency)
	case c.MulLatency < minMulLatency || c.MulLatency > maxMulLatency:
		return fmt.Errorf("mulLatency must be in %d..%d", minMulLatency, maxMulLatency)
	case c.DivLatency < minDivLatency || c.DivLatency > maxDivLatency:
		return fmt.Errorf("divLatency must be in %d..%d", minDivLatency, maxDivLatency)
	case c.DelaySlot && c.BTB:
		return errors.New("delay slot and branch target buffer are mutually exclusive")
	}
	return nil
}

// sanitize mimics the constructor reading winmips64.ini: out-of-range values
// keep the defaults, and BTB is only enabled without delay slot.
func (c Config) sanitize() Config {
	d := DefaultConfig()
	if c.CodeBits >= minCodeBits && c.CodeBits <= maxCodeBits {
		d.CodeBits = c.CodeBits
	}
	if c.DataBits >= minDataBits && c.DataBits <= maxDataBits {
		d.DataBits = c.DataBits
	}
	if c.AddLatency >= minAddLatency && c.AddLatency <= maxAddLatency {
		d.AddLatency = c.AddLatency
	}
	if c.MulLatency >= minMulLatency && c.MulLatency <= maxMulLatency {
		d.MulLatency = c.MulLatency
	}
	if c.DivLatency >= minDivLatency && c.DivLatency <= maxDivLatency {
		d.DivLatency = c.DivLatency
	}
	d.DelaySlot = c.DelaySlot
	d.Forwarding = c.Forwarding
	d.BTB = c.BTB && !c.DelaySlot
	d.RegistersAsNumbers = c.RegistersAsNumbers
	return d
}

// DefaultHistoryCap is the default number of history entries kept for the UI.
const DefaultHistoryCap = 200

// Sim is one simulator instance (the CWinMIPS64Doc document). Instances are
// fully independent.
type Sim struct {
	cfg Config

	CODESIZE, DATASIZE                    int
	ADD_LATENCY, MUL_LATENCY, DIV_LATENCY int

	cpu  processor
	pipe pipeline

	codelines, datalines, assembly, mnemonic []string

	codeTable, dataTable []symbol
	codeptr, dataptr     uint32
	CODEORDATA           int

	cycles, instructions, loads, stores       uint32
	branchTakenStalls, branchMispredictStalls uint32
	rawStalls, wawStalls, warStalls           uint32
	structuralStalls                          uint32

	multi   int
	stalls  int
	restart bool

	hist    *uiHist
	histCap int
	trace   *flatHist // byte-exact 50 entry history, only while tracing

	ioLine   []byte
	loaded   bool
	source   string
	haveSrc  bool
	lastMsgs []Message
	lastRes  result
}

// New creates a simulator with the given configuration (invalid fields fall
// back to defaults, as when reading winmips64.ini).
func New(cfg Config) *Sim {
	s := &Sim{histCap: DefaultHistoryCap}
	s.setup(cfg.sanitize())
	return s
}

// SetHistoryCap sets the number of history entries kept for Snapshot (the
// trace always uses the original 50-entry window). Resets the history.
func (s *Sim) SetHistoryCap(n int) {
	if n < 2 {
		n = 2
	}
	s.histCap = n
	s.hist = newUIHist(n)
	histClear(s.hist)
}

// setup is the constructor / OnFileMemory: (re)allocate everything.
func (s *Sim) setup(cfg Config) {
	s.cfg = cfg
	s.CODESIZE = 1 << cfg.CodeBits
	s.DATASIZE = 1 << cfg.DataBits
	s.ADD_LATENCY = cfg.AddLatency
	s.MUL_LATENCY = cfg.MulLatency
	s.DIV_LATENCY = cfg.DivLatency

	initProcessor(&s.cpu, s.CODESIZE, s.DATASIZE)
	initPipeline(&s.pipe, s.ADD_LATENCY, s.MUL_LATENCY, s.DIV_LATENCY)
	s.cpu.code = make([]byte, s.CODESIZE)
	s.cpu.cstat = make([]byte, s.CODESIZE)
	s.cpu.data = make([]byte, s.DATASIZE)
	s.cpu.dstat = make([]byte, s.DATASIZE)
	s.cpu.screen = make([]uint32, gSXY*gSXY)
	s.codelines = make([]string, s.CODESIZE/4)
	s.assembly = make([]string, s.CODESIZE/4)
	s.mnemonic = make([]string, s.CODESIZE/4)
	s.datalines = make([]string, s.DATASIZE/8)
	s.cpu.mm = [16]byte{}
	for i := range s.cpu.screen {
		s.cpu.screen[i] = colWHITE
	}
	s.cpu.Terminal = s.cpu.Terminal[:0]
	s.cpu.nlines = 0
	s.cpu.drawit = false
	s.cpu.keyboard = 0
	s.cpu.status = cpuRUNNING
	if s.hist == nil {
		s.hist = newUIHist(s.histCap)
	}
	s.clear()
	s.loaded = false
	s.lastMsgs = nil
}

func (s *Sim) histStores(f func(h histStore)) {
	f(s.hist)
	if s.trace != nil {
		f(s.trace)
	}
}

// clear is CWinMIPS64Doc::clear.
func (s *Sim) clear() {
	s.cycles, s.instructions, s.loads, s.stores = 0, 0, 0, 0
	s.branchTakenStalls, s.branchMispredictStalls = 0, 0
	s.rawStalls, s.wawStalls, s.warStalls, s.structuralStalls = 0, 0, 0, 0
	s.cpu.PC = 0
	s.histStores(histClear)
	s.multi = 5
	s.stalls = 0
}

// onFileReset resets the processor (OnFileReset).
func (s *Sim) onFileReset() {
	initPipeline(&s.pipe, s.ADD_LATENCY, s.MUL_LATENCY, s.DIV_LATENCY)
	s.cpu.PC = 0
	for i := 0; i < 64; i++ {
		s.cpu.rreg[i].val, s.cpu.wreg[i].val = 0, 0
		s.cpu.rreg[i].source, s.cpu.wreg[i].source = fROM_REGISTER, fROM_REGISTER
	}
	for i := range s.cpu.cstat {
		s.cpu.cstat[i] = 0
	}
	s.cpu.status = cpuRUNNING
	s.clear()
	s.cpu.mm = [16]byte{}
	for i := range s.cpu.screen {
		s.cpu.screen[i] = colWHITE
	}
	s.cpu.Terminal = s.cpu.Terminal[:0]
	s.cpu.nlines = 0
	s.cpu.drawit = false
	s.cpu.keyboard = 0
	s.cpu.fp_cc = false
	s.lastMsgs = nil
	s.ioLine = s.ioLine[:0]
}

// onFullReset also clears data memory (OnFullReset).
func (s *Sim) onFullReset() {
	for i := range s.datalines {
		s.datalines[i] = ""
	}
	for i := range s.cpu.data {
		s.cpu.data[i], s.cpu.dstat[i] = 0, 0
	}
	s.onFileReset()
}

// openfile resets everything and assembles src (nil = no file).
func (s *Sim) openfile(src *string) (int, []AsmError) {
	s.onFileReset()
	for i := range s.cpu.data {
		s.cpu.data[i], s.cpu.dstat[i] = 0, 0
	}
	for i := range s.cpu.code {
		s.cpu.code[i], s.cpu.cstat[i] = 0, 0
	}
	s.cpu.mm = [16]byte{}
	for i := range s.cpu.screen {
		s.cpu.screen[i] = colWHITE
	}
	s.cpu.Terminal = s.cpu.Terminal[:0]
	s.cpu.nlines = 0
	s.cpu.drawit = false
	s.cpu.keyboard = 0
	for i := range s.codelines {
		s.codelines[i], s.assembly[i], s.mnemonic[i] = "", "", ""
	}
	for i := range s.datalines {
		s.datalines[i] = ""
	}
	if src == nil {
		return 1, nil
	}
	return s.openit(*src)
}

// SetConfig changes the configuration. Memory size or latency changes
// re-create the machine (OnFileMemory) and reload the last program; toggling
// forwarding / delay slot / BTB resets the processor (OnFile...); the
// register naming option only affects presentation.
func (s *Sim) SetConfig(cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	old := s.cfg
	if cfg.CodeBits != old.CodeBits || cfg.DataBits != old.DataBits || cfg.AddLatency != old.AddLatency ||
		cfg.MulLatency != old.MulLatency || cfg.DivLatency != old.DivLatency {
		s.setup(cfg)
		if s.haveSrc {
			src := s.source
			s.Load(src)
		}
		return nil
	}
	s.cfg = cfg
	if cfg.Forwarding != old.Forwarding || cfg.DelaySlot != old.DelaySlot || cfg.BTB != old.BTB {
		s.onFileReset()
	}
	return nil
}

// Config returns the current configuration.
func (s *Sim) Config() Config { return s.cfg }

// Load assembles src (full reset). nil/empty result means success.
func (s *Sim) Load(src string) []AsmError {
	s.source = src
	s.haveSrc = true
	res, errs := s.openfile(&src)
	s.loaded = res == 0
	if res != 0 && len(errs) == 0 {
		errs = []AsmError{{Line: 0, Code: "syntax"}}
	}
	return errs
}

// Reset resets the processor; full also clears data memory (OnFullReset).
func (s *Sim) Reset(full bool) {
	if full {
		s.onFullReset()
	} else {
		s.onFileReset()
	}
}

// ---------------------------------------------------------------------------
// Messages

// Message is a status-bar message produced by a cycle.
type Message struct {
	Code  string `json:"code"`
	Stage string `json:"stage,omitempty"`
	Reg   string `json:"reg,omitempty"`
}

// Text returns the canonical English text of the message.
func (m Message) Text() string {
	switch m.Code {
	case "raw_stall":
		return "RAW Stall in " + m.Stage + " (" + m.Reg + ")"
	case "waw_stall":
		return "WAW Stall in " + m.Stage + " (" + m.Reg + ")"
	case "war_stall":
		return "WAR Stall in " + m.Stage + " (" + m.Reg + ")"
	case "branch_taken_stall":
		return "Branch Taken Stall"
	case "branch_mispredicted_stall":
		return "Branch Misprediction Stall"
	case "structural_stall":
		return "Structural Stall in " + m.Stage
	case "no_such_code_memory":
		return "No such code memory!"
	case "integer_overflow":
		return "Integer overflow!"
	case "divide_by_zero":
		return "Division by Zero in DIV!"
	case "uninitialized_memory":
		return "Uninitialised memory in MEM!"
	case "no_such_data_memory":
		return "No such data memory!"
	case "data_misaligned":
		return "Fatal Error - misaligned memory LOAD/STORE!"
	case "waiting_input":
		return "Waiting for input"
	}
	return m.Code
}

func regLabel(r int) string {
	if r < 32 {
		return "R" + strconv.Itoa(r)
	}
	return "F" + strconv.Itoa(r-32)
}

func (s *Sim) checkStalls(status int, stage string, rawreg int, msgs []Message) []Message {
	switch status {
	case RAW:
		s.rawStalls++
		msgs = append(msgs, Message{Code: "raw_stall", Stage: stage, Reg: regLabel(rawreg)})
	case WAW:
		s.wawStalls++
		msgs = append(msgs, Message{Code: "waw_stall", Stage: stage, Reg: regLabel(rawreg)})
	case WAR:
		s.warStalls++
		msgs = append(msgs, Message{Code: "war_stall", Stage: stage, Reg: regLabel(rawreg)})
	}
	return msgs
}

// processResult is CWinMIPS64Doc::process_result.
func (s *Sim) processResult(r *result) []Message {
	msgs := []Message{}
	if r.WB == OK || r.WB == HALTED {
		s.instructions++
	}
	if !s.cfg.DelaySlot && r.ID == BRANCH_TAKEN_STALL {
		s.branchTakenStalls++
		msgs = append(msgs, Message{Code: "branch_taken_stall"})
	}
	if r.ID == BRANCH_MISPREDICTED_STALL {
		s.branchMispredictStalls++
		msgs = append(msgs, Message{Code: "branch_mispredicted_stall"})
	}
	if r.MEM == LOADS || r.MEM == DATA_ERR {
		s.loads++
	}
	if r.MEM == STORES {
		s.stores++
	}
	msgs = s.checkStalls(r.ID, "ID", r.idrr, msgs)
	msgs = s.checkStalls(r.EX, "EX", r.exrr, msgs)
	msgs = s.checkStalls(r.ADDER[0], "ADD", r.addrr, msgs)
	msgs = s.checkStalls(r.MULTIPLIER[0], "MUL", r.mulrr, msgs)
	msgs = s.checkStalls(r.DIVIDER, "DIV", r.divrr, msgs)
	msgs = s.checkStalls(r.MEM, "MEM", r.memrr, msgs)

	if r.MEM != RAW {
		st := func(v int, stage string) {
			if v == STALLED {
				s.structuralStalls++
				msgs = append(msgs, Message{Code: "structural_stall", Stage: stage})
			}
		}
		st(r.ID, "ID")
		st(r.EX, "EX")
		st(r.DIVIDER, "FP-DIV")
		st(r.MULTIPLIER[s.MUL_LATENCY-1], "FP-MUL")
		st(r.ADDER[s.ADD_LATENCY-1], "FP-ADD")
	}
	if r.IF == NO_SUCH_CODE_MEMORY {
		msgs = append(msgs, Message{Code: "no_such_code_memory"})
		s.cpu.status = HALTED
	}
	if r.EX == INTEGER_OVERFLOW {
		msgs = append(msgs, Message{Code: "integer_overflow"})
	}
	if r.DIVIDER == DIVIDE_BY_ZERO {
		msgs = append(msgs, Message{Code: "divide_by_zero"})
	}
	if r.MEM == DATA_ERR {
		msgs = append(msgs, Message{Code: "uninitialized_memory"})
	}
	if r.MEM == NO_SUCH_DATA_MEMORY {
		msgs = append(msgs, Message{Code: "no_such_data_memory"})
	}
	if r.MEM == DATA_MISALIGNED {
		msgs = append(msgs, Message{Code: "data_misaligned"})
	}
	return msgs
}

// formatCf formats like printf("%lf") in the MSVC Universal CRT (which spells
// non-finite values "inf", "nan", "-nan(ind)", "nan(snan)").
func formatCf(d float64) string {
	switch {
	case math.IsInf(d, 1):
		return "inf"
	case math.IsInf(d, -1):
		return "-inf"
	case math.IsNaN(d):
		u := math.Float64bits(d)
		neg := u>>63 != 0
		quiet := (u>>51)&1 != 0
		sign := ""
		if neg {
			sign = "-"
		}
		switch {
		case u == 0xFFF8000000000000:
			return "-nan(ind)"
		case quiet:
			return sign + "nan"
		default:
			return sign + "nan(snan)"
		}
	}
	return strconv.FormatFloat(d, 'f', 6, 64)
}

// updateIO is CWinMIPS64Doc::update_io (memory mapped I/O at 0x10000).
func (s *Sim) updateIO() int {
	cpu := &s.cpu
	fn := pack32(cpu.mm[0:4])
	status := 0
	if fn == 0 {
		return status
	}
	fp := pack(cpu.mm[8:16])
	switch fn {
	case 1:
		cpu.Terminal = append(cpu.Terminal, strconv.FormatUint(fp, 10)+"\n"...)
	case 2:
		cpu.Terminal = append(cpu.Terminal, strconv.FormatInt(int64(fp), 10)+"\n"...)
	case 3:
		cpu.Terminal = append(cpu.Terminal, formatCf(f64(fp))+"\n"...)
	case 4:
		if fp < uint64(cpu.datasize) {
			for i := fp; i < uint64(len(cpu.data)) && cpu.data[i] != 0; i++ {
				cpu.Terminal = append(cpu.Terminal, cpu.data[i])
			}
		}
	case 5:
		y := int((fp >> 32) & 255)
		x := int((fp >> 40) & 255)
		cpu.drawit = true
		if x < gSXY && y < gSXY {
			cpu.screen[gSXY*y+x] = uint32(fp)
		}
	case 6:
		cpu.Terminal = cpu.Terminal[:0]
		cpu.nlines = 0
	case 7:
		for i := range cpu.screen {
			cpu.screen[i] = colWHITE
		}
		cpu.drawit = false
	case 8:
		cpu.keyboard = 1
		status = 1
	case 9:
		cpu.keyboard = 2
		status = 1
	}
	var nlines, ncols uint32
	for _, c := range cpu.Terminal {
		if c == '\n' {
			nlines++
			ncols = 0
		} else {
			ncols++
		}
	}
	cpu.nlines = nlines
	cpu.ncols = ncols
	cpu.mm[0], cpu.mm[1], cpu.mm[2], cpu.mm[3] = 0, 0, 0, 0
	return status
}

// oneCycle is CWinMIPS64Doc::one_cycle.
func (s *Sim) oneCycle() int {
	if s.cpu.status == HALTED {
		return HALTED
	}
	var res result // RESULT is zeroed before each clock_tick
	status := clockTick(&s.pipe, &s.cpu, s.cfg.Forwarding, s.cfg.DelaySlot, s.cfg.BTB, &res)
	s.cycles++
	s.lastMsgs = s.processResult(&res)

	// update_history: rewrite STALLED as STRUCTURAL, then update the stores
	if res.MEM != RAW {
		if res.ID == STALLED {
			res.ID = STRUCTURAL
		}
		if res.EX == STALLED {
			res.EX = STRUCTURAL
		}
		if res.DIVIDER == STALLED {
			res.DIVIDER = STRUCTURAL
		}
		if res.MULTIPLIER[s.MUL_LATENCY-1] == STALLED {
			res.MULTIPLIER[s.MUL_LATENCY-1] = STRUCTURAL
		}
		if res.ADDER[s.ADD_LATENCY-1] == STALLED {
			res.ADDER[s.ADD_LATENCY-1] = STRUCTURAL
		}
	}
	s.histStores(func(h histStore) { updateHistory(h, &s.pipe, &s.cpu, &res, s.cycles) })
	s.lastRes = res

	if s.updateIO() != 0 {
		return WAITING_FOR_INPUT
	}
	if status == HALTED {
		s.cpu.status = HALTED
		return HALTED
	}
	return status
}

// ---------------------------------------------------------------------------
// Execution API

// StepResult is returned by Step / StepN.
type StepResult struct {
	Status   string    `json:"status"`
	Messages []Message `json:"messages"`
}

// RunResult is returned by RunTo.
type RunResult struct {
	Cycles    int       `json:"cycles"`
	Status    string    `json:"status"`
	StoppedBy string    `json:"stoppedBy"`
	Messages  []Message `json:"messages"`
}

func (s *Sim) runStatus() string {
	switch {
	case s.cpu.status == HALTED:
		return "halted"
	case s.cpu.keyboard != 0:
		return "waiting_input"
	}
	return "ok"
}

func (s *Sim) messages() []Message {
	m := append([]Message{}, s.lastMsgs...)
	if s.cpu.keyboard != 0 {
		m = append(m, Message{Code: "waiting_input"})
	}
	return m
}

func (s *Sim) cycle() int {
	ret := s.oneCycle()
	if ret == HALTED && s.cpu.status == HALTED && s.lastMsgs == nil {
		s.lastMsgs = []Message{}
	}
	return ret
}

// Step executes one clock cycle (OnExecuteSingle).
func (s *Sim) Step() StepResult {
	if !s.loaded || s.cpu.keyboard != 0 {
		return StepResult{Status: s.runStatus(), Messages: s.messages()}
	}
	if s.cpu.status == HALTED {
		s.lastMsgs = []Message{}
	}
	s.cycle()
	return StepResult{Status: s.runStatus(), Messages: s.messages()}
}

// StepN executes n cycles (OnExecuteMulticycle with multi = n).
func (s *Sim) StepN(n int) StepResult {
	if !s.loaded || s.cpu.keyboard != 0 {
		return StepResult{Status: s.runStatus(), Messages: s.messages()}
	}
	if n < 1 {
		n = 1
	}
	if s.cpu.status == HALTED {
		s.lastMsgs = []Message{}
	}
	status := 0
	for i := 0; i < n-1; i++ {
		status = s.cycle()
		if status != 0 {
			break
		}
	}
	if status == 0 {
		s.cycle()
	}
	return StepResult{Status: s.runStatus(), Messages: s.messages()}
}

// RunTo runs until a breakpoint, HALT, input request or maxCycles cycles
// (OnExecuteRunto, executed in batches).
func (s *Sim) RunTo(maxCycles int) RunResult {
	if !s.loaded {
		return RunResult{Status: s.runStatus(), StoppedBy: "error", Messages: s.messages()}
	}
	if s.cpu.keyboard != 0 {
		return RunResult{Status: s.runStatus(), StoppedBy: "input", Messages: s.messages()}
	}
	if maxCycles < 1 {
		maxCycles = 1
	}
	if s.cpu.status == HALTED {
		s.lastMsgs = []Message{}
	}
	lapsed := 0
	status := 0
	stopped := ""
	for {
		lapsed++
		status = s.cycle()
		if status != 0 {
			break
		}
		if !(s.stalls != 0 || ((s.cpu.cstat[s.cpu.PC]&1) == 0 && s.cpu.status != HALTED)) {
			break
		}
		if lapsed >= maxCycles {
			stopped = "limit"
			break
		}
	}
	s.restart = status == WAITING_FOR_INPUT
	if stopped == "" {
		switch {
		case status == WAITING_FOR_INPUT:
			stopped = "input"
		case status == HALTED || s.cpu.status == HALTED:
			stopped = "halted"
		default:
			stopped = "breakpoint"
		}
	}
	return RunResult{Cycles: lapsed, Status: s.runStatus(), StoppedBy: stopped, Messages: s.messages()}
}

// SendInput delivers a line typed in the terminal followed by Enter, with
// the semantics of CIOView::OnChar: for CONTROL=8 each character is echoed
// (backspace deletes) and on Enter the line is parsed as a double if it
// contains '.', otherwise as a 64-bit integer (saturating), into mm[8..15];
// for CONTROL=9 only mm[8] receives the first key (13 for an empty line).
func (s *Sim) SendInput(text string) error {
	if s.cpu.keyboard == 0 {
		return errors.New("not waiting for input")
	}
	keys := append([]byte(text), 13) // the line followed by Enter
	for _, nChar := range keys {
		if s.cpu.keyboard == 0 {
			break
		}
		if s.cpu.keyboard == 2 { // just take one character, do not echo it.
			s.cpu.mm[8] = nChar
			s.cpu.keyboard = 0
		}
		if s.cpu.keyboard == 1 {
			if nChar == 13 {
				var v uint64
				if strings.IndexByte(string(s.ioLine), '.') >= 0 {
					v = u64(cAtof(string(s.ioLine)))
				} else {
					v = uint64(cAtoi64(string(s.ioLine)))
				}
				unpack(v, s.cpu.mm[8:16])
				s.ioLine = s.ioLine[:0]
				s.cpu.Terminal = append(s.cpu.Terminal, '\n')
				s.cpu.keyboard = 0
			} else if nChar == 8 {
				if len(s.ioLine) > 0 {
					s.ioLine = s.ioLine[:len(s.ioLine)-1]
					if len(s.cpu.Terminal) > 0 {
						s.cpu.Terminal = s.cpu.Terminal[:len(s.cpu.Terminal)-1]
					}
					s.cpu.ncols--
				}
			} else {
				s.ioLine = append(s.ioLine, nChar)
				s.cpu.Terminal = append(s.cpu.Terminal, nChar)
				s.cpu.ncols++
			}
		}
	}
	return nil
}

// ToggleBreakpoint sets/clears a breakpoint on the instruction at addr.
func (s *Sim) ToggleBreakpoint(addr uint32) {
	addr &^= 3
	if int(addr) >= len(s.cpu.cstat) {
		return
	}
	if s.cpu.cstat[addr]&1 == 0 {
		s.cpu.cstat[addr] |= 1
	} else {
		s.cpu.cstat[addr] &= 0xfe
	}
}

// SetReg edits integer register i (1..31) when it holds a committed value.
func (s *Sim) SetReg(i int, v uint64) { s.SetRegChecked(i, v) }

// SetRegChecked is SetReg reporting whether the edit was applied.
func (s *Sim) SetRegChecked(i int, v uint64) bool {
	if i <= 0 || i > 31 || s.cpu.rreg[i].source != fROM_REGISTER {
		return false
	}
	s.cpu.rreg[i].val = v
	s.cpu.wreg[i].val = v
	return true
}

// SetFReg edits FP register i (0..31) when it holds a committed value.
func (s *Sim) SetFReg(i int, f float64) { s.SetFRegChecked(i, f) }

// SetFRegChecked is SetFReg reporting whether the edit was applied.
func (s *Sim) SetFRegChecked(i int, f float64) bool {
	if i < 0 || i > 31 || s.cpu.rreg[i+32].source != fROM_REGISTER {
		return false
	}
	s.cpu.rreg[i+32].val = u64(f)
	s.cpu.wreg[i+32].val = u64(f)
	return true
}

// SetMem writes an aligned 64-bit data word and marks it initialised.
func (s *Sim) SetMem(addr uint32, v uint64) { s.SetMemChecked(addr, v) }

// SetMemChecked is SetMem reporting whether the address was valid.
func (s *Sim) SetMemChecked(addr uint32, v uint64) bool {
	addr &^= 7
	if int(addr)+8 > len(s.cpu.data) {
		return false
	}
	unpack(v, s.cpu.data[addr:])
	for i := uint32(0); i < 8; i++ {
		s.cpu.dstat[addr+i] = wRITTEN
	}
	return true
}

// SetMemDouble writes a double into an aligned data word.
func (s *Sim) SetMemDouble(addr uint32, f float64) { s.SetMemChecked(addr, u64(f)) }

// SetMemDoubleChecked is SetMemDouble reporting success.
func (s *Sim) SetMemDoubleChecked(addr uint32, f float64) bool { return s.SetMemChecked(addr, u64(f)) }

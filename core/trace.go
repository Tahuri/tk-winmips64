// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0
//
// Derived from WinMIPS64 by Mike Scott (Apache-2.0,
// https://github.com/mcarrickscott/WinMIPS64) and the fork by Andoni
// Zubimendi (https://github.com/AndoniZubimendi/WinMIPS64).
// Modified: ported from C++/MFC to Go. See NOTICE.

package core

import (
	"bufio"
	"fmt"
	"hash/fnv"
	"io"
	"strings"
)

func escapeC(b []byte) string {
	var sb strings.Builder
	for _, c := range b {
		switch {
		case c == '\n':
			sb.WriteString(`\n`)
		case c == '\\':
			sb.WriteString(`\\`)
		case c == '"':
			sb.WriteString(`\"`)
		case c == '\t':
			sb.WriteString(`\t`)
		case c < 0x20 || c >= 0x7f:
			fmt.Fprintf(&sb, `\x%02x`, c)
		default:
			sb.WriteByte(c)
		}
	}
	return sb.String()
}

func fnv32(b []byte) uint32 {
	h := fnv.New32a()
	h.Write(b)
	return h.Sum32()
}

func b2i(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Trace runs the loaded program writing the golden trace format (CONTRACT
// §1). The simulator must have been freshly loaded; assembly errors must be
// reported by the caller (see TraceSource).
func (s *Sim) Trace(w io.Writer, input []string, maxCycles int) {
	bw := bufio.NewWriter(w)
	defer bw.Flush()
	s.trace = newFlatHist()
	histClear(s.trace)
	defer func() { s.trace = nil }()

	s.writeHeader(bw)
	for s.cpu.status != HALTED && int(s.cycles) < maxCycles {
		ret := s.oneCycle()
		s.writeCycle(bw, ret)
		if ret == WAITING_FOR_INPUT {
			if len(input) == 0 {
				fmt.Fprintln(bw, "INPUT EOF")
				break
			}
			_ = s.SendInput(input[0])
			input = input[1:]
		}
	}
	fmt.Fprintf(bw, "END %d %d\n", s.cycles, s.instructions)
}

// TraceSource assembles src and traces it (the `wmips trace` protocol).
func TraceSource(w io.Writer, src string, cfg Config, input []string, maxCycles int) {
	s := New(cfg)
	if errs := s.Load(src); len(errs) > 0 {
		fmt.Fprintln(w, "ASM ERR")
		for _, e := range errs {
			fmt.Fprintf(w, "E %d %s\n", e.Line, e.Text)
		}
		return
	}
	s.Trace(w, input, maxCycles)
}

func (s *Sim) writeHeader(w io.Writer) {
	cpu := &s.cpu
	fmt.Fprintln(w, "ASM OK")
	fmt.Fprintf(w, "CODE %x\n", cpu.code[:cpu.codesize])
	fmt.Fprintf(w, "CSTAT %x\n", cpu.cstat[:cpu.codesize])
	fmt.Fprintf(w, "DATA %x\n", cpu.data[:cpu.datasize])
	fmt.Fprintf(w, "DSTAT %x\n", cpu.dstat[:cpu.datasize])
	for i, l := range s.codelines {
		fmt.Fprintf(w, "CL %d %s\n", i, l)
	}
	for i, l := range s.datalines {
		fmt.Fprintf(w, "DL %d %s\n", i, l)
	}
}

func (s *Sim) writeCycle(w io.Writer, ret int) {
	cpu := &s.cpu
	p := &s.pipe
	r := &s.lastRes
	fmt.Fprintf(w, "C %d RET %d CPU %d PC %08x\n", s.cycles, ret, cpu.status, cpu.PC)
	regs := func(tag string, rr *[64]reg) {
		var sb strings.Builder
		sb.WriteString(tag)
		for i := 0; i < 64; i++ {
			fmt.Fprintf(&sb, " %016x:%d", rr[i].val, rr[i].source)
		}
		sb.WriteByte('\n')
		io.WriteString(w, sb.String())
	}
	regs("R", &cpu.rreg)
	regs("W", &cpu.wreg)
	fmt.Fprintf(w, "FCC %d\n", b2i(cpu.fp_cc))

	var sb strings.Builder
	st := func(a bool, ir uint32) string { return fmt.Sprintf("%d,%08x", b2i(a), ir) }
	fmt.Fprintf(&sb, "P IF %s ID %s EX %s A ", st(p.active, cpu.PC), st(p.if_id.active, p.if_id.IR), st(p.integer.active, p.integer.IR))
	for i := 0; i < s.ADD_LATENCY; i++ {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(st(p.a[i].active, p.a[i].IR))
	}
	sb.WriteString(" M ")
	for i := 0; i < s.MUL_LATENCY; i++ {
		if i > 0 {
			sb.WriteByte(';')
		}
		sb.WriteString(st(p.m[i].active, p.m[i].IR))
	}
	fmt.Fprintf(&sb, " DIV %s MEM %s WB %s\n", st(p.div.active, p.div.IR), st(p.ex_mem.active, p.ex_mem.IR), st(p.mem_wb.active, p.mem_wb.IR))
	io.WriteString(w, sb.String())

	sb.Reset()
	fmt.Fprintf(&sb, "X %d %d %d %d %d %d A", r.IF, r.ID, r.EX, r.MEM, r.WB, r.DIVIDER)
	for i := 0; i < s.ADD_LATENCY; i++ {
		fmt.Fprintf(&sb, " %d", r.ADDER[i])
	}
	sb.WriteString(" M")
	for i := 0; i < s.MUL_LATENCY; i++ {
		fmt.Fprintf(&sb, " %d", r.MULTIPLIER[i])
	}
	fmt.Fprintf(&sb, " RR %d %d %d %d %d %d\n", r.idrr, r.exrr, r.memrr, r.addrr, r.mulrr, r.divrr)
	io.WriteString(w, sb.String())

	fmt.Fprintf(w, "S %d %d %d %d %d %d %d %d %d\n", s.instructions, s.loads, s.stores, s.rawStalls, s.wawStalls,
		s.warStalls, s.structuralStalls, s.branchTakenStalls, s.branchMispredictStalls)
	sb.Reset()
	sb.WriteString("MSG ")
	for _, m := range s.lastMsgs {
		sb.WriteString("  ")
		sb.WriteString(m.Text())
	}
	sb.WriteByte('\n')
	io.WriteString(w, sb.String())
	fmt.Fprintf(w, "T %s\n", escapeC(cpu.Terminal))
	scr := make([]byte, 4*len(cpu.screen))
	for i, c := range cpu.screen {
		unpack32(c, scr[4*i:])
	}
	fmt.Fprintf(w, "MEMH %08x %08x %08x\n", fnv32(cpu.data), fnv32(cpu.dstat), fnv32(scr))

	h := s.trace
	sb.Reset()
	fmt.Fprintf(&sb, "H %d", h.n())
	for i := 0; i < h.n(); i++ {
		stc := h.start(i)
		fmt.Fprintf(&sb, " %08x@%d:", h.ir(i), stc)
		for j := 0; j <= int(s.cycles-stc) && j < recStatus; j++ {
			if j > 0 {
				sb.WriteByte('/')
			}
			a, b, c := h.get(i, j)
			fmt.Fprintf(&sb, "%d.%d.%d", a, b, c)
		}
	}
	sb.WriteByte('\n')
	io.WriteString(w, sb.String())
}

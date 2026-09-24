// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"encoding/binary"
	"strings"
	"testing"
)

func mustLoad(t *testing.T, cfg Config, src string) *Sim {
	t.Helper()
	s := New(cfg)
	if errs := s.Load(src); len(errs) > 0 {
		t.Fatalf("assembly failed: %+v", errs)
	}
	return s
}

func word(s *Sim, addr int) uint32  { return binary.LittleEndian.Uint32(s.cpu.code[addr:]) }
func dword(s *Sim, addr int) uint64 { return binary.LittleEndian.Uint64(s.cpu.data[addr:]) }

func TestEncodings(t *testing.T) {
	cases := []struct {
		line string
		want uint32
	}{
		{"daddi r1,r2,5", 0x60410005},
		{"daddi r1,r2,-1", 0x6041ffff},
		{"daddui r3,r4,0xffff", 0x6483ffff},
		{"andi r1,r2,0x0f", 0x3041000f},
		{"ori r1,r2,1", 0x34410001},
		{"xori r1,r2,1", 0x38410001},
		{"slti r1,r2,1", 0x28410001},
		{"sltiu r1,r2,1", 0x2c410001},
		{"lui r1,0x1234", 0x3c011234},
		{"ld r1,8(r2)", 0xdc410008},
		{"sd r1,-8(r2)", 0xfc41fff8},
		{"lb r1,0(r2)", 0x80410000},
		{"lbu r1,0(r2)", 0x90410000},
		{"sb r1,0(r2)", 0xa0410000},
		{"lh r1,0(r2)", 0x84410000},
		{"lhu r1,0(r2)", 0x94410000},
		{"sh r1,0(r2)", 0xa4410000},
		{"lw r1,0(r2)", 0x8c410000},
		{"lwu r1,0(r2)", 0x9c410000},
		{"sw r1,0(r2)", 0xac410000},
		{"l.d f1,16(r2)", 0xd4410010},
		{"s.d f1,16(r2)", 0xf4410010},
		{"ld r1,(r2)", 0xdc410000},
		{"halt", 0x04000000},
		{"nop", 0x00000000},
		{"dadd r3,r1,r2", 0x0022182c},
		{"daddu r3,r1,r2", 0x0022182d},
		{"dsub r3,r1,r2", 0x0022182e},
		{"dsubu r3,r1,r2", 0x0022182f},
		{"and r3,r1,r2", 0x00221824},
		{"or r3,r1,r2", 0x00221825},
		{"xor r3,r1,r2", 0x00221826},
		{"slt r3,r1,r2", 0x0022182a},
		{"sltu r3,r1,r2", 0x0022182b},
		{"dsllv r3,r1,r2", 0x00221814},
		{"dsrlv r3,r1,r2", 0x00221816},
		{"dsrav r3,r1,r2", 0x00221817},
		{"movz r3,r1,r2", 0x0022180a},
		{"movn r3,r1,r2", 0x0022180b},
		{"dmul r3,r1,r2", 0x0022181c},
		{"dmulu r3,r1,r2", 0x0022181d},
		{"ddiv r3,r1,r2", 0x0022181e},
		{"ddivu r3,r1,r2", 0x0022181f},
		{"dsll r1,r2,3", 0x004008f8},
		{"dsrl r1,r2,3", 0x004008fa},
		{"dsra r1,r2,3", 0x004008fb},
		{"jr r31", 0x001f0008},
		{"jalr r5", 0x00050009},
		{"add.d f1,f2,f3", 0x46231040},
		{"sub.d f1,f2,f3", 0x46231041},
		{"mul.d f1,f2,f3", 0x46231042},
		{"div.d f1,f2,f3", 0x46231043},
		{"mov.d f1,f2", 0x46201046},
		{"cvt.d.l f1,f2", 0x46201061},
		{"cvt.l.d f1,f2", 0x46201065},
		{"c.lt.d f1,f2", 0x4622083c},
		{"c.le.d f1,f2", 0x4622083e},
		{"c.eq.d f1,f2", 0x46220832},
		{"mtc1 r1,f2", 0x44811000},
		{"mfc1 r1,f2", 0x44011000},
		{"DADDI $t0,$zero,1", 0x60080001},
		{"daddi $s7,$sp,1", 0x63b70001},
		{"daddi $t8,$ra,1", 0x63f80001},
		{"daddi $a3,$v1,1", 0x60670001},
		{"daddi $k1,$gp,1", 0x639b0001},
	}
	for _, c := range cases {
		s := mustLoad(t, DefaultConfig(), ".text\n "+c.line+"\n")
		if got := word(s, 0); got != c.want {
			t.Errorf("%-20s = %08x, want %08x", c.line, got, c.want)
		}
	}
}

// Every opcode of the table decodes back to its own subtype.
func TestOpcodeTableRoundTrip(t *testing.T) {
	for _, c := range codes {
		var ins instruction
		want := c.subtype
		if strings.HasPrefix(c.name, "dmul") || strings.HasPrefix(c.name, "ddiv") {
			want = tREG3X // assembled as REG3, executed by the FP-style units
		}
		if got := parse(c.opCode, &ins); got != want {
			t.Errorf("%s: decoded type %d, want %d", c.name, got, want)
		}
	}
}

func TestBranchOffsets(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), `
	.text
start:	j fwd          ; 0
	beq r1,r2,fwd  ; 4
	bnez r3,start  ; 8
	bc1t fwd       ; c
	bc1f start     ; 10
	jal start      ; 14
fwd:	halt           ; 18
`)
	want := []uint32{0x08000005, 0x10410004, 0x1c03fffd, 0x45010002, 0x4500fffb, 0x0ffffffa}
	for i, w := range want {
		if got := word(s, 4*i); got != w {
			t.Errorf("word %d = %08x, want %08x", i, got, w)
		}
	}
}

func TestDirectives(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), `
	.data
a:	.word 1, -1
b:	.byte 1, 2, 255
c:	.word16 0x1234, -2
d:	.word32 a, b+4
e:	.double 1.5
f:	.asciiz "ab\n"
g:	.ascii "xy"
	.space 16
h:	.align 8
	.word 0x10
	.org 0x100
i:	.word 7
	.text
	halt
`)
	checks := []struct {
		addr int
		want uint64
	}{
		{0, 1}, {8, 0xffffffffffffffff}, {16, 0xff0201}, {24, 0xfffe1234},
		{32, 0x0000001400000000}, {40, 0x3ff8000000000000}, {48, 0x000a6261}, {56, 0x7978},
		{0x100, 7},
	}
	for _, c := range checks {
		if got := dword(s, c.addr); got != c.want {
			t.Errorf("data[%#x] = %#x, want %#x", c.addr, got, c.want)
		}
	}
	p := s.Program()
	syms := map[string]uint32{}
	for _, sy := range p.Symbols {
		syms[sy.Name] = sy.Addr
	}
	for name, addr := range map[string]uint32{"a": 0, "b": 16, "c": 24, "d": 32, "e": 40, "f": 48, "g": 56, "h": 80, "i": 0x100} {
		if syms[name] != addr {
			t.Errorf("symbol %s = %#x, want %#x", name, syms[name], addr)
		}
	}
	if p.Data[0].Label != "a" || !strings.Contains(p.Data[0].Text, ".word 1, -1") {
		t.Errorf("data line 0 = %+v", p.Data[0])
	}
	if len(s.datalines[0]) != maxLine {
		t.Errorf("source lines must be padded to %d chars, got %d", maxLine, len(s.datalines[0]))
	}
}

// The .ascii size computation tests the first character of the string for a
// backslash (bug of the original): "\hello" is sized as if every character
// were escaped.
func TestAsciiSizeQuirk(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".data\nx: .asciiz \"\\hello\"\ny: .word 1\n.text\nhalt\n")
	if s.dataTable[1].value != 8 {
		t.Errorf("y at %d, want 8", s.dataTable[1].value)
	}
}

func TestAsmErrors(t *testing.T) {
	cases := []struct {
		src  string
		line int
		code string
	}{
		{".text\n daddi r1,r0,70000\n", 2, "bad_number"},
		{".text\n daddi r1,r0,nosuch\n", 2, "undefined_symbol"},
		{".text\n daddi r32,r0,1\n", 2, "bad_register"},
		{".text\n dadd r1,r2\n", 2, "syntax"},
		{".text\n j nowhere\n", 2, "undefined_symbol"},
		{".text\n bogus r1\n", 2, "bad_instruction"},
		{".data\n .bogus 1\n", 2, "bad_directive"},
		{"x: .word 1\n", 1, "syntax"},              // label before any segment
		{".data\n daddi r1,r0,1\n", 2, "syntax"},   // instruction in data
		{".text\n .word 1\n", 2, "bad_directive"},  // data directive in code
		{".data\n.byte 256\n", 2, "bad_directive"}, // pass 2 range check
		{".text\n halt\n" + strings.Repeat(" nop\n", 300), 258, "out_of_memory"},
	}
	for _, c := range cases {
		s := New(DefaultConfig())
		errs := s.Load(c.src)
		if len(errs) == 0 {
			t.Errorf("%q: expected an error", c.src)
			continue
		}
		if errs[0].Line != c.line || errs[0].Code != c.code {
			t.Errorf("%q: got %+v, want line %d code %s", c.src, errs[0], c.line, c.code)
		}
		if s.Snapshot().Loaded {
			t.Errorf("%q: loaded after errors", c.src)
		}
	}
	// pass 2 reports every failing line
	s := New(DefaultConfig())
	errs := s.Load(".text\n daddi r1,r0,70000\n nop\n j nowhere\n halt\n")
	if len(errs) != 2 || errs[0].Line != 2 || errs[1].Line != 4 || errs[1].Text != " j nowhere" {
		t.Errorf("pass 2 errors = %+v", errs)
	}
	if s.cpu.cstat[0] != 4 || s.cpu.cstat[8] != 4 {
		t.Errorf("failing lines must be marked in cstat")
	}
}

func TestLongLine(t *testing.T) {
	// lines of 197 characters or more are rejected (mygets buffer)
	l := " daddi r1,r0,1 ;" + strings.Repeat("x", 197)
	errs := New(DefaultConfig()).Load(".text\n" + l + "\n halt\n")
	if len(errs) != 1 || errs[0].Line != 2 {
		t.Errorf("long line: %+v", errs)
	}
}

func TestCRLFAndTabs(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".text\r\n\tdaddi\tr1,r0,1\r\n\thalt\r\n\x1agarbage")
	if word(s, 0) != 0x60010001 || word(s, 4) != 0x04000000 {
		t.Errorf("code = %08x %08x", word(s, 0), word(s, 4))
	}
	if !strings.HasPrefix(s.codelines[0], "  daddi  r1,r0,1 ") {
		t.Errorf("tabs must become two spaces: %q", s.codelines[0][:20])
	}
}

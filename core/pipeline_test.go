package core

import (
	"encoding/json"
	"math"
	"strings"
	"testing"
)

func run(t *testing.T, cfg Config, src string) *Sim {
	t.Helper()
	s := mustLoad(t, cfg, src)
	r := s.RunTo(100000)
	if r.StoppedBy != "halted" {
		t.Fatalf("stopped by %s", r.StoppedBy)
	}
	return s
}

func cfgWith(f func(*Config)) Config {
	c := DefaultConfig()
	f(&c)
	return c
}

func hasMsg(ms []Message, code string) bool {
	for _, m := range ms {
		if m.Code == code {
			return true
		}
	}
	return false
}

func TestSimplePipelineTiming(t *testing.T) {
	s := run(t, DefaultConfig(), ".text\n daddi r1,r0,1\n daddi r2,r0,2\n daddi r3,r0,3\n daddi r4,r0,4\n halt\n")
	sn := s.Snapshot()
	if sn.Cycles != 9 || sn.Instructions != 5 {
		t.Errorf("cycles/instructions = %d/%d, want 9/5", sn.Cycles, sn.Instructions)
	}
	if sn.Regs[4].Value != "0000000000000004" || sn.Regs[4].Name != "$a0" {
		t.Errorf("r4 = %+v", sn.Regs[4])
	}
	if sn.Status != "halted" {
		t.Errorf("status %s", sn.Status)
	}
}

func TestForwardingRAW(t *testing.T) {
	src := ".text\n daddi r1,r0,5\n dadd r2,r1,r1\n dadd r3,r2,r1\n halt\n"
	fw := run(t, DefaultConfig(), src).Snapshot()
	nf := run(t, cfgWith(func(c *Config) { c.Forwarding = false }), src).Snapshot()
	if fw.Stats.RawStalls != 0 {
		t.Errorf("forwarding: %d RAW stalls, want 0", fw.Stats.RawStalls)
	}
	if nf.Stats.RawStalls != 4 || nf.Cycles != fw.Cycles+4 {
		t.Errorf("no forwarding: raw=%d cycles=%d (fwd %d)", nf.Stats.RawStalls, nf.Cycles, fw.Cycles)
	}
	for _, sn := range []Snapshot{fw, nf} {
		if sn.Regs[3].Value != "000000000000000f" {
			t.Errorf("r3 = %s", sn.Regs[3].Value)
		}
	}
}

func TestLoadUseStall(t *testing.T) {
	s := run(t, DefaultConfig(), ".data\nx: .word 7\n.text\n ld r1,x(r0)\n dadd r2,r1,r1\n halt\n")
	sn := s.Snapshot()
	if sn.Stats.RawStalls != 1 || sn.Stats.Loads != 1 || sn.Regs[2].Value != "000000000000000e" {
		t.Errorf("raw=%d loads=%d r2=%s", sn.Stats.RawStalls, sn.Stats.Loads, sn.Regs[2].Value)
	}
}

func TestBranchTakenAndDelaySlot(t *testing.T) {
	src := ".text\n j over\n daddi r1,r0,5\nover: halt\n"
	nd := run(t, DefaultConfig(), src).Snapshot()
	ds := run(t, cfgWith(func(c *Config) { c.DelaySlot = true }), src).Snapshot()
	if nd.Stats.BranchTakenStalls != 1 || nd.Regs[1].Value != "0000000000000000" {
		t.Errorf("no delay slot: bts=%d r1=%s", nd.Stats.BranchTakenStalls, nd.Regs[1].Value)
	}
	if ds.Stats.BranchTakenStalls != 0 || ds.Regs[1].Value != "0000000000000005" {
		t.Errorf("delay slot: bts=%d r1=%s", ds.Stats.BranchTakenStalls, ds.Regs[1].Value)
	}
}

func TestBTB(t *testing.T) {
	src := ".text\n daddi r1,r0,4\nloop: daddi r1,r1,-1\n bnez r1,loop\n halt\n"
	plain := run(t, DefaultConfig(), src).Snapshot()
	s := run(t, cfgWith(func(c *Config) { c.BTB = true }), src)
	btb := s.Snapshot()
	if plain.Stats.BranchTakenStalls != 3 || plain.Stats.BranchMispredictionStalls != 0 {
		t.Errorf("plain: %+v", plain.Stats)
	}
	// first taken: not in the BTB, ID holds the branch one extra cycle to
	// record it (2 taken stalls); then predicted twice; final mispredict
	// costs 2 stalls.
	if btb.Stats.BranchTakenStalls != 2 || btb.Stats.BranchMispredictionStalls != 2 {
		t.Errorf("btb: %+v", btb.Stats)
	}
	if len(btb.Predicted) != 0 {
		t.Errorf("prediction bit must be cleared after the mispredict: %v", btb.Predicted)
	}
	if btb.Regs[1].Value != "0000000000000000" {
		t.Errorf("r1 = %s", btb.Regs[1].Value)
	}
}

func TestFPLatency(t *testing.T) {
	src := ".data\nx: .double 1.5\n.text\n l.d f1,x(r0)\n add.d f2,f1,f1\n mul.d f3,f2,f1\n div.d f4,f3,f1\n s.d f4,x(r0)\n halt\n"
	a := run(t, DefaultConfig(), src).Snapshot()
	b := run(t, cfgWith(func(c *Config) { c.AddLatency, c.MulLatency, c.DivLatency = 2, 3, 10 }), src).Snapshot()
	if a.Cycles-b.Cycles != (4-2)+(7-3)+(24-10) {
		t.Errorf("latency difference = %d cycles", a.Cycles-b.Cycles)
	}
	if a.FRegs[4].Value != 3.0 || a.Data[0].Double != 3.0 {
		t.Errorf("f4 = %v, x = %v", a.FRegs[4].Value, a.Data[0].Double)
	}
	if a.FRegs[4].Bits != "4008000000000000" {
		t.Errorf("f4 bits = %s", a.FRegs[4].Bits)
	}
}

func TestStructuralStall(t *testing.T) {
	src := ".text\n div.d f1,f2,f3\n div.d f4,f5,f6\n halt\n"
	s := run(t, DefaultConfig(), src)
	if s.Snapshot().Stats.StructuralStalls == 0 {
		t.Errorf("expected structural stalls waiting for the divider")
	}
}

func TestWAWAndWAR(t *testing.T) {
	s := run(t, DefaultConfig(), ".text\n div.d f1,f2,f3\n add.d f1,f4,f5\n halt\n")
	if s.Snapshot().Stats.WawStalls == 0 {
		t.Errorf("expected WAW stalls")
	}
}

func TestDivideByZeroAndOverflow(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".data\nb: .word 0x7fffffffffffffff\n.text\n ld r1,b(r0)\n ddiv r2,r1,r0\n dadd r3,r1,r1\n halt\n")
	var sawDiv, sawOvf bool
	for i := 0; i < 100 && s.Snapshot().Status != "halted"; i++ {
		r := s.Step()
		sawDiv = sawDiv || hasMsg(r.Messages, "divide_by_zero")
		sawOvf = sawOvf || hasMsg(r.Messages, "integer_overflow")
	}
	if !sawDiv || !sawOvf {
		t.Errorf("divide_by_zero=%v integer_overflow=%v", sawDiv, sawOvf)
	}
}

func TestMemoryErrors(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".data\nx: .word 1\ny: .space 8\n.text\n ld r1,1(r0)\n ld r2,y(r0)\n sd r1,4096(r0)\n halt\n")
	seen := map[string]bool{}
	for i := 0; i < 100 && s.Snapshot().Status != "halted"; i++ {
		for _, m := range s.Step().Messages {
			seen[m.Code] = true
		}
	}
	for _, c := range []string{"data_misaligned", "uninitialized_memory", "no_such_data_memory"} {
		if !seen[c] {
			t.Errorf("missing message %s (got %v)", c, seen)
		}
	}
}

func TestNoSuchCodeMemory(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".text\n daddi r1,r0,0x3fc\n jr r1\n")
	r := s.RunTo(1000)
	if r.StoppedBy != "halted" || !hasMsg(s.lastMsgs, "no_such_code_memory") {
		t.Errorf("run = %+v", r)
	}
}

const ioProg = `
	.data
CONTROL: .word32 0x10000
DATA:    .word32 0x10008
msg:	.asciiz "hi\n"
dbl:	.double 2.5
	.text
	lwu r1,CONTROL(r0)
	lwu r2,DATA(r0)
	daddi r3,r0,-7
	sd r3,(r2)
	daddi r4,r0,1
	sd r4,(r1)
	daddi r4,r0,2
	sd r4,(r1)
	daddi r3,r0,msg
	sd r3,(r2)
	daddi r4,r0,4
	sd r4,(r1)
	l.d f1,dbl(r0)
	s.d f1,(r2)
	daddi r4,r0,3
	sd r4,(r1)
	lui r5,0x0102
	dsll r5,r5,16
	ori r5,r5,0x00ff
	sd r5,(r2)
	daddi r4,r0,5
	sd r4,(r1)
	daddi r4,r0,8
	sd r4,(r1)
	ld r6,(r2)
	daddi r4,r0,9
	sd r4,(r1)
	lbu r7,(r2)
	halt
`

func TestMMIO(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ioProg)
	r := s.RunTo(10000)
	if r.StoppedBy != "input" || r.Status != "waiting_input" || s.Snapshot().InputKind != "number" {
		t.Fatalf("first stop: %+v", r)
	}
	if err := s.SendInput("-42"); err != nil {
		t.Fatal(err)
	}
	r = s.RunTo(10000)
	if r.StoppedBy != "input" || s.Snapshot().InputKind != "char" {
		t.Fatalf("second stop: %+v", r)
	}
	s.SendInput("A")
	if err := s.SendInput("again"); err == nil {
		t.Errorf("SendInput must fail when not waiting")
	}
	r = s.RunTo(10000)
	if r.StoppedBy != "halted" {
		t.Fatalf("third stop: %+v", r)
	}
	sn := s.Snapshot()
	want := "18446744073709551609\n-7\nhi\n2.500000\n-42\n"
	if sn.Terminal != want {
		t.Errorf("terminal = %q, want %q", sn.Terminal, want)
	}
	if sn.Regs[6].Value != "ffffffffffffffd6" || sn.Regs[7].Value != "0000000000000041" {
		t.Errorf("r6=%s r7=%s", sn.Regs[6].Value, sn.Regs[7].Value)
	}
	// pixel (x=1,y=2) = 0x0000ff (red in COLORREF)
	if !sn.Screen.Drawn || s.cpu.screen[2*gSXY+1] != 0xff {
		t.Errorf("pixel not drawn: %x", s.cpu.screen[2*gSXY+1])
	}
}

func TestMMIOClear(t *testing.T) {
	s := run(t, DefaultConfig(), `
	.data
CONTROL: .word32 0x10000
DATA:    .word32 0x10008
	.text
	lwu r1,CONTROL(r0)
	lwu r2,DATA(r0)
	daddi r3,r0,1
	sd r3,(r2)
	sd r3,(r1)
	daddi r3,r0,5
	sd r3,(r2)
	sd r3,(r1)
	daddi r3,r0,6
	sd r3,(r1)
	daddi r3,r0,7
	sd r3,(r1)
	halt
`)
	sn := s.Snapshot()
	if sn.Terminal != "" || sn.Screen.Drawn {
		t.Errorf("CONTROL 6/7 must clear: %q drawn=%v", sn.Terminal, sn.Screen.Drawn)
	}
}

func TestInputDoubleAndBackspace(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ioProg)
	s.RunTo(10000)
	s.SendInput("1x\b.25")
	if got := f64(pack(s.cpu.mm[8:16])); got != 1.25 {
		t.Errorf("input = %v", got)
	}
	if !strings.HasSuffix(string(s.cpu.Terminal), "1.25\n") {
		t.Errorf("echo = %q", s.cpu.Terminal)
	}
}

func TestBreakpointAndStepN(t *testing.T) {
	s := mustLoad(t, DefaultConfig(), ".text\n daddi r1,r0,1\n daddi r2,r0,2\n daddi r3,r0,3\n halt\n")
	s.ToggleBreakpoint(8)
	r := s.RunTo(1000)
	if r.StoppedBy != "breakpoint" || s.Snapshot().PC != 8 {
		t.Errorf("run = %+v pc=%d", r, s.Snapshot().PC)
	}
	if bp := s.Snapshot().Breakpoints; len(bp) != 1 || bp[0] != 8 {
		t.Errorf("breakpoints = %v", bp)
	}
	s.ToggleBreakpoint(8)
	s.StepN(3)
	if c := s.Snapshot().Cycles; c != r.Cycles+3 {
		t.Errorf("cycles after StepN = %d", c)
	}
	r = s.RunTo(2)
	if r.StoppedBy != "limit" || r.Cycles != 2 {
		t.Errorf("limit run = %+v", r)
	}
}

func TestResetAndEdits(t *testing.T) {
	s := run(t, DefaultConfig(), ".data\nx: .word 5\n.text\n ld r1,x(r0)\n halt\n")
	s.Reset(false)
	sn := s.Snapshot()
	if sn.Cycles != 0 || sn.Regs[1].Value != "0000000000000000" || sn.Data[0].Value != "0000000000000005" {
		t.Errorf("reset: %+v", sn.Stats)
	}
	s.SetReg(1, 0xabc)
	s.SetReg(0, 1)
	s.SetFReg(2, 2.5)
	s.SetMem(8, 0x1234)
	s.SetMemDouble(16, -1.0)
	sn = s.Snapshot()
	if sn.Regs[1].Value != "0000000000000abc" || sn.Regs[0].Value != "0000000000000000" || sn.FRegs[2].Value != 2.5 ||
		sn.Data[1].Value != "0000000000001234" || !sn.Data[1].Written || sn.Data[2].Double != -1 {
		t.Errorf("edits not applied: %+v %+v", sn.Regs[1], sn.Data[1])
	}
	s.Reset(true)
	if s.Snapshot().Data[0].Value != "0000000000000000" {
		t.Errorf("full reset must clear data memory")
	}
}

func TestIndependentInstances(t *testing.T) {
	a := mustLoad(t, DefaultConfig(), ".text\n daddi r1,r0,1\n halt\n")
	b := mustLoad(t, cfgWith(func(c *Config) { c.CodeBits = 8 }), ".text\n daddi r1,r0,2\n halt\n")
	a.RunTo(100)
	if b.Snapshot().Cycles != 0 || b.Snapshot().Regs[1].Value != "0000000000000000" {
		t.Errorf("instances share state")
	}
	b.RunTo(100)
	if a.Snapshot().Regs[1].Value != "0000000000000001" || b.Snapshot().Regs[1].Value != "0000000000000002" {
		t.Errorf("wrong results")
	}
	if len(b.Program().Code) != 64 {
		t.Errorf("codebits 8 must give 64 code words")
	}
}

func TestSnapshotJSON(t *testing.T) {
	s := run(t, DefaultConfig(), ".data\nz: .double 0.0\n.text\n l.d f1,z(r0)\n div.d f2,f1,f1\n c.eq.d f1,f1\n halt\n")
	sn := s.Snapshot()
	b, err := json.Marshal(sn)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{`"pipeline":{"if":`, `"rawStalls":`, `"inputKind"`, `"startCycle":`, `"screen":{"width":50`} {
		if k == `"inputKind"` {
			if strings.Contains(string(b), k) {
				t.Errorf("inputKind must be omitted when not waiting")
			}
			continue
		}
		if !strings.Contains(string(b), k) {
			t.Errorf("JSON lacks %s", k)
		}
	}
	if !sn.FPCC || len(sn.History) == 0 || len(sn.Pipeline.Add) != 4 || len(sn.Pipeline.Mul) != 7 {
		t.Errorf("snapshot: fpcc=%v hist=%d", sn.FPCC, len(sn.History))
	}
	if _, err := json.Marshal(s.Program()); err != nil {
		t.Fatal(err)
	}
}

func TestHistoryCap(t *testing.T) {
	src := ".text\nl: daddi r1,r1,1\n slti r2,r1,100\n bnez r2,l\n halt\n"
	s := mustLoad(t, DefaultConfig(), src)
	s.RunTo(100000)
	if n := len(s.Snapshot().History); n != DefaultHistoryCap-1 {
		t.Errorf("history entries = %d", n)
	}
	s2 := New(DefaultConfig())
	s2.SetHistoryCap(50)
	s2.Load(src)
	s2.RunTo(100000)
	if n := len(s2.Snapshot().History); n != 49 {
		t.Errorf("history entries with cap 50 = %d", n)
	}
	last := s.Snapshot().History[len(s.Snapshot().History)-1]
	if last.Mnemonic != "halt" {
		t.Errorf("last entry = %+v", last)
	}
}

func TestCvtAndFormatQuirks(t *testing.T) {
	if cvtLD(math.NaN()) != math.MinInt64 || cvtLD(1e300) != math.MinInt64 || cvtLD(-3.9) != -3 {
		t.Errorf("cvt.l.d must follow x86 cvttsd2si")
	}
	if formatCf(math.Float64frombits(0xFFF8000000000000)) != "-nan(ind)" || formatCf(math.Inf(-1)) != "-inf" || formatCf(1.0/3) != "0.333333" {
		t.Errorf("printf %%lf emulation")
	}
}

func TestConfigValidate(t *testing.T) {
	if DefaultConfig().Validate() != nil {
		t.Error("default config invalid")
	}
	bad := []Config{
		cfgWith(func(c *Config) { c.CodeBits = 14 }),
		cfgWith(func(c *Config) { c.DataBits = 3 }),
		cfgWith(func(c *Config) { c.AddLatency = 9 }),
		cfgWith(func(c *Config) { c.MulLatency = 1 }),
		cfgWith(func(c *Config) { c.DivLatency = 31 }),
		cfgWith(func(c *Config) { c.BTB, c.DelaySlot = true, true }),
	}
	for _, c := range bad {
		if c.Validate() == nil {
			t.Errorf("%+v should be invalid", c)
		}
	}
	s := mustLoad(t, DefaultConfig(), ".text\n daddi r1,r0,1\n halt\n")
	if err := s.SetConfig(cfgWith(func(c *Config) { c.MulLatency = 3 })); err != nil || !s.Snapshot().Loaded {
		t.Errorf("SetConfig must reload the program: %v", err)
	}
	if s.SetConfig(bad[5]) == nil {
		t.Errorf("SetConfig must validate")
	}
}

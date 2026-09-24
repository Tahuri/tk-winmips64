package core

import (
	"encoding/base64"
	"fmt"
	"math"
	"strings"
)

// AsmError is an assembler error.
type AsmError struct {
	Line int    `json:"line"`
	Code string `json:"code"`
	Text string `json:"text"`
}

// Program is the static view of the assembled program.
type Program struct {
	CodeSize int        `json:"codeSize"`
	DataSize int        `json:"dataSize"`
	Code     []CodeLine `json:"code"`
	Data     []DataLine `json:"data"`
	Symbols  []Symbol   `json:"symbols"`
}

type CodeLine struct {
	Addr  uint32 `json:"addr"`
	Word  string `json:"word"`
	Text  string `json:"text"`
	Label string `json:"label,omitempty"`
	Used  bool   `json:"used"`
}

type DataLine struct {
	Addr  uint32 `json:"addr"`
	Text  string `json:"text"`
	Label string `json:"label,omitempty"`
}

type Symbol struct {
	Name string `json:"name"`
	Addr uint32 `json:"addr"`
	Kind string `json:"kind"`
}

type Stats struct {
	Loads                     int     `json:"loads"`
	Stores                    int     `json:"stores"`
	RawStalls                 int     `json:"rawStalls"`
	WawStalls                 int     `json:"wawStalls"`
	WarStalls                 int     `json:"warStalls"`
	StructuralStalls          int     `json:"structuralStalls"`
	BranchTakenStalls         int     `json:"branchTakenStalls"`
	BranchMispredictionStalls int     `json:"branchMispredictionStalls"`
	CodeSize                  int     `json:"codeSize"`
	DataSize                  int     `json:"dataSize"`
	CPI                       float64 `json:"cpi"`
}

type RegView struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Source string `json:"source"`
}

type FRegView struct {
	Name   string  `json:"name"`
	Bits   string  `json:"bits"`
	Value  float64 `json:"value"`
	Source string  `json:"source"`
}

type StageSlot struct {
	Active   bool   `json:"active"`
	Addr     uint32 `json:"addr"`
	Mnemonic string `json:"mnemonic"`
}

type PipeView struct {
	IF  StageSlot   `json:"if"`
	ID  StageSlot   `json:"id"`
	EX  StageSlot   `json:"ex"`
	Add []StageSlot `json:"add"`
	Mul []StageSlot `json:"mul"`
	Div StageSlot   `json:"div"`
	MEM StageSlot   `json:"mem"`
	WB  StageSlot   `json:"wb"`
}

type DataWord struct {
	Addr    uint32  `json:"addr"`
	Value   string  `json:"value"`
	Double  float64 `json:"double"`
	Written bool    `json:"written"`
	Label   string  `json:"label,omitempty"`
	Text    string  `json:"text"`
}

type HistEntry struct {
	Addr       uint32     `json:"addr"`
	Mnemonic   string     `json:"mnemonic"`
	StartCycle int        `json:"startCycle"`
	Cells      []HistCell `json:"cells"`
}

type HistCell struct {
	Stage string `json:"stage"`
	Cause string `json:"cause,omitempty"`
}

type ScreenView struct {
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Drawn  bool   `json:"drawn"`
	Pixels string `json:"pixels"`
}

type Snapshot struct {
	Config       Config      `json:"config"`
	Loaded       bool        `json:"loaded"`
	Status       string      `json:"status"`
	InputKind    string      `json:"inputKind,omitempty"`
	Cycles       int         `json:"cycles"`
	Instructions int         `json:"instructions"`
	Stats        Stats       `json:"stats"`
	Messages     []Message   `json:"messages"`
	PC           uint32      `json:"pc"`
	Regs         []RegView   `json:"regs"`
	FRegs        []FRegView  `json:"fregs"`
	FPCC         bool        `json:"fpcc"`
	Breakpoints  []uint32    `json:"breakpoints"`
	Predicted    []uint32    `json:"predicted"`
	Pipeline     PipeView    `json:"pipeline"`
	Data         []DataWord  `json:"data"`
	History      []HistEntry `json:"history"`
	Terminal     string      `json:"terminal"`
	Screen       ScreenView  `json:"screen"`
}

func labelMap(t []symbol) map[uint32]string {
	m := map[uint32]string{}
	for _, sy := range t {
		if _, ok := m[sy.value]; !ok {
			m[sy.value] = string(sy.symb)
		}
	}
	return m
}

func trimPad(s string) string { return strings.TrimRight(s, " ") }

// Program returns the static program view (valid after Load).
func (s *Sim) Program() Program {
	p := Program{CodeSize: s.CODESIZE, DataSize: s.DATASIZE, Code: []CodeLine{}, Data: []DataLine{}, Symbols: []Symbol{}}
	cl := labelMap(s.codeTable)
	dl := labelMap(s.dataTable)
	for i := 0; i < s.CODESIZE/4; i++ {
		a := uint32(4 * i)
		p.Code = append(p.Code, CodeLine{
			Addr: a, Word: fmt.Sprintf("%08x", pack32(s.cpu.code[a:])), Text: trimPad(s.codelines[i]),
			Label: cl[a], Used: s.codelines[i] != "",
		})
	}
	for i := 0; i < s.DATASIZE/8; i++ {
		a := uint32(8 * i)
		p.Data = append(p.Data, DataLine{Addr: a, Text: trimPad(s.datalines[i]), Label: dl[a]})
	}
	for _, sy := range s.codeTable {
		p.Symbols = append(p.Symbols, Symbol{Name: string(sy.symb), Addr: sy.value, Kind: "code"})
	}
	for _, sy := range s.dataTable {
		p.Symbols = append(p.Symbols, Symbol{Name: string(sy.symb), Addr: sy.value, Kind: "data"})
	}
	return p
}

func sourceName(src int32) string {
	switch {
	case src <= nOT_AVAILABLE:
		return "na"
	case src == fROM_REGISTER:
		return "reg"
	case src == fROM_MEM:
		return "mem"
	case src == fROM_EX:
		return "ex"
	case src == fROM_ID:
		return "id"
	case src == fROM_ADD:
		return "add"
	case src == fROM_MUL:
		return "mul"
	case src == fROM_DIV:
		return "div"
	}
	return "na"
}

func finite(f float64) float64 {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return 0 // JSON cannot carry NaN/Inf; the raw bits are also exported
	}
	return f
}

func stageName(stage, sub byte) string {
	switch stage {
	case stIFETCH:
		return "IF"
	case stIDECODE:
		return "ID"
	case stINTEX:
		return "EX"
	case stADDEX:
		return fmt.Sprintf("A%d", sub)
	case stMULEX:
		return fmt.Sprintf("M%d", sub)
	case stDIVEX:
		return "DIV"
	case stMEMORY:
		return "MEM"
	case stWRITEB:
		return "WB"
	}
	return ""
}

func causeName(c byte) string {
	switch int(c) {
	case RAW:
		return "raw"
	case WAW:
		return "waw"
	case WAR:
		return "war"
	case STRUCTURAL:
		return "structural"
	case BRANCH_TAKEN_STALL:
		return "branch_taken"
	case BRANCH_MISPREDICTED_STALL:
		return "branch_mispredicted"
	}
	return ""
}

func (s *Sim) mnem(addr uint32) string {
	if i := int(addr / 4); i < len(s.mnemonic) {
		return s.mnemonic[i]
	}
	return ""
}

// Snapshot returns the dynamic state of the machine.
func (s *Sim) Snapshot() Snapshot {
	cpu := &s.cpu
	sn := Snapshot{
		Config:       s.cfg,
		Loaded:       s.loaded,
		Status:       s.runStatus(),
		Cycles:       int(s.cycles),
		Instructions: int(s.instructions),
		Messages:     s.messages(),
		PC:           cpu.PC,
		FPCC:         cpu.fp_cc,
		Breakpoints:  []uint32{},
		Predicted:    []uint32{},
		Terminal:     string(cpu.Terminal),
	}
	if !s.loaded {
		sn.Status = "idle"
	}
	switch cpu.keyboard {
	case 1:
		sn.InputKind = "number"
	case 2:
		sn.InputKind = "char"
	}
	sn.Stats = Stats{
		Loads: int(s.loads), Stores: int(s.stores), RawStalls: int(s.rawStalls), WawStalls: int(s.wawStalls),
		WarStalls: int(s.warStalls), StructuralStalls: int(s.structuralStalls),
		BranchTakenStalls: int(s.branchTakenStalls), BranchMispredictionStalls: int(s.branchMispredictStalls),
		CodeSize: int(s.codeptr), DataSize: int(s.dataptr),
	}
	if s.instructions > 0 {
		sn.Stats.CPI = float64(s.cycles) / float64(s.instructions)
	}
	for i := 0; i < 32; i++ {
		sn.Regs = append(sn.Regs, RegView{
			Name: registerName(i, s.cfg.RegistersAsNumbers), Value: fmt.Sprintf("%016x", cpu.rreg[i].val),
			Source: sourceName(cpu.rreg[i].source),
		})
	}
	for i := 0; i < 32; i++ {
		r := cpu.rreg[i+32]
		sn.FRegs = append(sn.FRegs, FRegView{
			Name: fmt.Sprintf("F%d", i), Bits: fmt.Sprintf("%016x", r.val), Value: finite(f64(r.val)),
			Source: sourceName(r.source),
		})
	}
	for a := 0; a < len(cpu.cstat); a += 4 {
		if cpu.cstat[a]&1 != 0 {
			sn.Breakpoints = append(sn.Breakpoints, uint32(a))
		}
		if cpu.cstat[a]&2 != 0 {
			sn.Predicted = append(sn.Predicted, uint32(a))
		}
	}
	p := &s.pipe
	slot := func(active bool, ir uint32) StageSlot {
		return StageSlot{Active: active, Addr: ir, Mnemonic: s.mnem(ir)}
	}
	sn.Pipeline = PipeView{
		IF:  slot(p.active, cpu.PC),
		ID:  slot(p.if_id.active, p.if_id.IR),
		EX:  slot(p.integer.active, p.integer.IR),
		Div: slot(p.div.active, p.div.IR),
		MEM: slot(p.ex_mem.active, p.ex_mem.IR),
		WB:  slot(p.mem_wb.active, p.mem_wb.IR),
		Add: []StageSlot{}, Mul: []StageSlot{},
	}
	for i := 0; i < s.ADD_LATENCY; i++ {
		sn.Pipeline.Add = append(sn.Pipeline.Add, slot(p.a[i].active, p.a[i].IR))
	}
	for i := 0; i < s.MUL_LATENCY; i++ {
		sn.Pipeline.Mul = append(sn.Pipeline.Mul, slot(p.m[i].active, p.m[i].IR))
	}
	dl := labelMap(s.dataTable)
	for i := 0; i < s.DATASIZE/8; i++ {
		a := 8 * i
		v := pack(cpu.data[a:])
		w := false
		for k := 0; k < 8; k++ {
			if cpu.dstat[a+k] != 0 {
				w = true
			}
		}
		sn.Data = append(sn.Data, DataWord{
			Addr: uint32(a), Value: fmt.Sprintf("%016x", v), Double: finite(f64(v)), Written: w,
			Label: dl[uint32(a)], Text: trimPad(s.datalines[i]),
		})
	}
	h := s.hist
	sn.History = []HistEntry{}
	for i := 0; i < h.n(); i++ {
		st := h.start(i)
		e := HistEntry{Addr: h.ir(i), StartCycle: int(st), Cells: []HistCell{}}
		if k := int(h.ir(i) / 4); k < len(s.assembly) {
			e.Mnemonic = s.assembly[k]
		}
		for j := 0; j <= int(s.cycles-st); j++ {
			stg, sub, c := h.get(i, j)
			e.Cells = append(e.Cells, HistCell{Stage: stageName(stg, sub), Cause: causeName(c)})
		}
		sn.History = append(sn.History, e)
	}
	// screen: RGB rows top to bottom as displayed (the original flips y)
	pix := make([]byte, 0, gSXY*gSXY*3)
	for row := 0; row < gSXY; row++ {
		y := gSXY - 1 - row
		for x := 0; x < gSXY; x++ {
			c := cpu.screen[gSXY*y+x] // COLORREF 0x00BBGGRR
			pix = append(pix, byte(c), byte(c>>8), byte(c>>16))
		}
	}
	sn.Screen = ScreenView{Width: gSXY, Height: gSXY, Drawn: cpu.drawit, Pixels: base64.StdEncoding.EncodeToString(pix)}
	return sn
}

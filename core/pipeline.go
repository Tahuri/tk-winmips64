package core

// Port of pipeline.cpp (MIPS64 pipeline simulator, Mike Scott 2003, andoni
// fork v1.60). Stages are executed in reverse order WB/MEM/EX/ID/IF.

import (
	"encoding/binary"
	"math"
)

func getType(instruct uint32) int {
	opcode := int(instruct >> 26)
	if opcode == i_SPECIAL {
		function := int(instruct & 0x3F)
		typ := tREG3
		if function == r_DMULU || function == r_DMUL || function == r_DDIV || function == r_DDIVU {
			typ = tREG3X
		}
		if function == r_DSRL || function == r_DSRA || function == r_DSLL {
			typ = tREG2S
		}
		if function == r_JR || function == r_JALR {
			typ = tJREG
		}
		if function == r_NOP {
			typ = tNOP
		}
		return typ
	}
	if opcode == i_COP1 {
		typ := 0
		fmt := int((instruct >> 21) & 0x1F)
		function := int(instruct & 0x3F)
		if fmt == i_DOUBLE {
			typ = tREG3F
			if function == f_CVT_L_D || function == f_CVT_D_L || function == f_MOV_D {
				typ = tREG2F
			}
			if function == f_C_LT_D || function == f_C_LE_D || function == f_C_EQ_D {
				typ = tREG2C
			}
		}
		if fmt == i_MTC1 {
			typ = tREGID
		}
		if fmt == i_MFC1 {
			typ = tREGDI
		}
		if fmt == i_BC {
			typ = tBC
		}
		return typ
	}
	typ := tREG2I
	if opcode == i_J || opcode == i_JAL {
		typ = tJUMP
	}
	if opcode == i_LUI {
		typ = tREG1I
	}
	op3 := opcode >> 3
	if op3 == 0x4 || opcode == i_LD {
		typ = tLOAD
	}
	if op3 == 0x5 || opcode == i_SD {
		typ = tSTORE
	}
	if opcode == i_L_D {
		typ = tFLOAD
	}
	if opcode == i_S_D {
		typ = tFSTORE
	}
	if opcode == i_BEQ || opcode == i_BNE {
		typ = tBRANCH
	}
	if opcode == i_BEQZ || opcode == i_BNEZ {
		typ = tJREGN
	}
	if opcode == i_HALT {
		typ = tHALT
	}
	return typ
}

func parse(instruct uint32, ins *instruction) int {
	typ := getType(instruct)
	ins.opcode = int(instruct >> 26)
	ins.typ = typ
	ins.function = int(instruct & 0x3F)

	r1 := int((instruct >> 16) & 0x1F)
	r2 := int((instruct >> 21) & 0x1F)
	r3 := int((instruct >> 11) & 0x1F)
	r4 := int((instruct >> 6) & 0x1F)

	ins.target = -1
	ins.src1 = -1
	ins.src2 = -1
	ins.tf = int((instruct >> 16) & 0x01)

	ins.Imm = int32(int16(instruct & 0xFFFF)) // sign extended

	switch typ {
	case tREG3, tREG3X:
		ins.rt, ins.src1 = r1, r1
		ins.rs, ins.src2 = r2, r2
		ins.rd = r3
		ins.target = r3
	case tREG2I:
		ins.rt = r1
		ins.rs, ins.src2 = r2, r2
		ins.rd = -1
		ins.target = r1
	case tREG1I:
		ins.rt = r1
		ins.rs = -1
		ins.rd = -1
		ins.target = r1
	case tLOAD:
		ins.rt = r1
		ins.rs, ins.src2 = r2, r2
		ins.rd = -1
		ins.target = r1
	case tSTORE:
		ins.rt, ins.src1 = r1, r1
		ins.rs, ins.src2 = r2, r2
		ins.rd = -1
	case tFSTORE:
		ins.rs, ins.src1 = r2, r2
		ins.rt, ins.src2 = r1+32, r1+32
		ins.rd = -1
	case tFLOAD:
		ins.rs, ins.src2 = r2, r2
		ins.rt = r1 + 32
		ins.rd = -1
		ins.target = ins.rt
	case tREG3F:
		ins.rs, ins.src1 = r3+32, r3+32
		ins.rd = r4 + 32
		ins.rt, ins.src2 = r1+32, r1+32
		ins.target = ins.rd
	case tREG2F:
		ins.rs, ins.src1 = r3+32, r3+32
		ins.rt = -1
		ins.rd = r4 + 32
		ins.target = ins.rd
	case tREG2C:
		ins.rs, ins.src1 = r3+32, r3+32
		ins.rt, ins.src2 = r1+32, r1+32
		ins.rd = -1
	case tREGID:
		ins.rs = -1
		ins.rt, ins.src1 = r1, r1
		ins.rd = r3 + 32
		ins.target = ins.rd
	case tREGDI:
		ins.rs = -1
		ins.rt = r1
		ins.rd, ins.src1 = r3+32, r3+32
		ins.target = ins.rt
	case tJUMP:
		ins.rt, ins.rs, ins.rd = -1, -1, -1
		ins.Imm = int32(instruct&0x3FFFFFF<<6) >> 6
		if ins.opcode == i_JAL {
			ins.target = 31
		}
	case tJREG:
		ins.rt, ins.src1 = r1, r1
		ins.rs = -1
		ins.rd = -1
		if ins.opcode == i_SPECIAL && ins.function == r_JALR {
			ins.target = 31
		}
	case tREG2S:
		ins.rt = -1
		ins.rs, ins.src2 = r2, r2
		ins.rd = r3
		ins.target = r3
		ins.Imm = int32((instruct >> 6) & 0x1F)
	case tJREGN:
		ins.rt, ins.src1 = r1, r1
		ins.rs = -1
		ins.rd = -1
	case tBRANCH:
		ins.rt, ins.src1 = r1, r1
		ins.rs, ins.src2 = r2, r2
		ins.rd = -1
	default: // HALT, NOP, BC
		ins.rt, ins.rs, ins.rd = -1, -1, -1
	}
	return typ
}

func available(cpu *processor, r int) bool {
	if r == 0 || r < 0 {
		return true
	}
	return cpu.rreg[r].source > nOT_AVAILABLE
}

func unavail(cpu *processor, typ int, r int) {
	if r < 0 {
		return
	}
	if typ == accREAD || typ == accBOTH {
		if cpu.rreg[r].source > nOT_AVAILABLE {
			cpu.rreg[r].source = nOT_AVAILABLE
		} else {
			cpu.rreg[r].source-- // even less available!
		}
	}
	if typ == accWRITE || typ == accBOTH {
		if cpu.wreg[r].source > nOT_AVAILABLE {
			cpu.wreg[r].source = nOT_AVAILABLE
		} else {
			cpu.wreg[r].source--
		}
	}
}

func makeAvailable(cpu *processor, r int, lmd uint64) {
	if cpu.rreg[r].source < nOT_AVAILABLE {
		cpu.rreg[r].source++
	} else {
		cpu.rreg[r].source = fROM_REGISTER
		cpu.rreg[r].val = lmd
	}
	if cpu.wreg[r].source < nOT_AVAILABLE {
		cpu.wreg[r].source++
	} else {
		cpu.wreg[r].source = fROM_REGISTER
		cpu.wreg[r].val = lmd
	}
}

// waw: check if the destination register r is going to be written by one of
// the FP units.
func waw(pipe *pipeline, typ, function, r int) bool {
	switch typ {
	case tREGID, tREGDI, tLOAD, tFLOAD, tFSTORE, tSTORE, tREG2F, tREG3, tREG1I, tREG2I, tREG2S:
		if pipe.div.active && pipe.div.cycles < pipe.DIV_LATENCY && pipe.div.ins.rd == r {
			return true
		}
		for i := 1; i < pipe.MUL_LATENCY; i++ {
			if pipe.m[i].active && pipe.m[i].ins.rd == r {
				return true
			}
		}
		for i := 1; i < pipe.ADD_LATENCY; i++ {
			if pipe.a[i].active && pipe.a[i].ins.rd == r {
				return true
			}
		}
	case tREG3F, tREG3X:
		switch function {
		case f_DIV_D, r_DDIV, r_DDIVU:
		case f_MUL_D, r_DMUL, r_DMULU:
			if pipe.div.active && pipe.div.cycles < pipe.DIV_LATENCY && pipe.div.ins.rd == r {
				return true
			}
			for i := 1; i < pipe.ADD_LATENCY; i++ {
				if pipe.a[i].active && pipe.a[i].ins.rd == r {
					return true
				}
			}
		case f_ADD_D, f_SUB_D:
			if pipe.div.active && pipe.div.cycles < pipe.DIV_LATENCY && pipe.div.ins.rd == r {
				return true
			}
			for i := 1; i < pipe.MUL_LATENCY; i++ {
				if pipe.m[i].active && pipe.m[i].ins.rd == r {
					return true
				}
			}
		}
	}
	return false
}

func ifStage(pipe *pipeline, cpu *processor, delaySlot, btb bool) int {
	var ins instruction

	if pipe.if_id.active {
		return STALLED
	}
	if pipe.active {
		pipe.if_id.IR = cpu.PC
		var cw [4]byte
		for i := uint32(0); i < 4; i++ {
			if cpu.PC+i < uint32(len(cpu.code)) {
				cw[i] = cpu.code[cpu.PC+i]
			}
		}
		codeword := binary.LittleEndian.Uint32(cw[:])
		parse(codeword, &ins)
	} else {
		pipe.if_id.active = false
		return OK
	}
	pipe.if_id.ins = ins

	// Instruction just fetched may need to be nullified
	pipe.if_id.active = true
	if !delaySlot && pipe.branch {
		pipe.if_id.active = false
		pipe.if_id.ins.typ = tNOP
	}
	if pipe.if_id.ins.typ == tHALT {
		pipe.active = false
	}

	pipe.if_id.NPC = cpu.PC + 4
	pipe.if_id.predicted = false

	if !pipe.branch && btb && (cpu.cstat[cpu.PC]&2) != 0 {
		pipe.if_id.NPC = cpu.PC + 4 + uint32(4*ins.Imm)
		pipe.if_id.predicted = true
	} else if pipe.branch && pipe.active {
		pipe.if_id.NPC = pipe.destination
	}
	status := OK
	if pipe.if_id.NPC >= cpu.codesize {
		status = NO_SUCH_CODE_MEMORY
		pipe.active = false
	} else {
		cpu.PC = pipe.if_id.NPC
	}
	pipe.branch = false
	return status
}

func alreadyTarget(pipe *pipeline, r int) bool {
	if pipe.div.active && pipe.div.cycles == pipe.DIV_LATENCY && pipe.div.ins.rd == r {
		return true
	}
	if pipe.m[0].active && pipe.m[0].ins.rd == r {
		return true
	}
	if pipe.a[0].active && pipe.a[0].ins.rd == r {
		return true
	}
	if pipe.integer.active && pipe.integer.ins.target == r {
		return true
	}
	return false
}

func alreadyWaitedFor(pipe *pipeline, r int) bool {
	if pipe.m[0].active {
		if pipe.m[0].ins.rs == r || pipe.m[0].ins.rt == r {
			return true
		}
	}
	if pipe.a[0].active {
		if pipe.a[0].ins.rs == r || pipe.a[0].ins.rt == r {
			return true
		}
	}
	if pipe.div.active && pipe.div.cycles == pipe.DIV_LATENCY {
		if pipe.div.ins.rs == r || pipe.div.ins.rt == r {
			return true
		}
	}
	if pipe.integer.active { // V1.53
		if pipe.integer.ins.src1 == r || pipe.integer.ins.src2 == r {
			return true
		}
	}
	return false
}

func idStage(pipe *pipeline, cpu *processor, forwarding, btb bool, rawreg *int) int {
	var A, B uint64

	if !pipe.if_id.active {
		return EMPTY
	}
	ins := pipe.if_id.ins
	typ := ins.typ

	if pipe.integer.active {
		if typ != tREG3F && typ != tREG3X {
			return STALLED // exit is blocked
		}
	}

	switch typ {
	case tLOAD, tFLOAD, tREG2I, tREG2S, tREG2F:
		if !forwarding {
			if !available(cpu, ins.rs) {
				*rawreg = ins.rs
				return RAW
			}
		}
		pipe.integer.rA = ins.rs
		pipe.integer.rB = -1
	case tREGID:
		if !forwarding {
			if !available(cpu, ins.rt) {
				*rawreg = ins.rt
				return RAW
			}
		}
		pipe.integer.rA = ins.rt
		pipe.integer.rB = -1
	case tREGDI:
		if !forwarding {
			if !available(cpu, ins.rd) {
				*rawreg = ins.rd
				return RAW
			}
		}
		pipe.integer.rA = ins.rd
		pipe.integer.rB = -1
	case tSTORE, tREG3, tFSTORE, tREG2C:
		if !forwarding {
			if !available(cpu, ins.rt) {
				*rawreg = ins.rt
				return RAW
			}
			if !available(cpu, ins.rs) {
				*rawreg = ins.rs
				return RAW
			}
		}
		pipe.integer.rA = ins.rs
		pipe.integer.rB = ins.rt
	case tREG3F, tREG3X:
		if !forwarding {
			if !available(cpu, ins.rt) {
				*rawreg = ins.rt
				return RAW
			}
			if !available(cpu, ins.rs) {
				*rawreg = ins.rs
				return RAW
			}
		}
	case tJREG, tJREGN:
		if !available(cpu, ins.rt) {
			*rawreg = ins.rt
			return RAW
		}
		B = cpu.rreg[ins.rt].val
	case tBRANCH:
		if !available(cpu, ins.rs) {
			*rawreg = ins.rs
			return RAW
		}
		if !available(cpu, ins.rt) {
			*rawreg = ins.rt
			return RAW
		}
		A = cpu.rreg[ins.rs].val
		B = cpu.rreg[ins.rt].val
	}

	branchStatus := nOT_A_BRANCH
	predictable := false

	chk := func(r, code int) bool {
		if alreadyTarget(pipe, r) {
			*rawreg = r
			return true
		}
		_ = code
		return false
	}
	chkW := func(r int) bool {
		if alreadyWaitedFor(pipe, r) {
			*rawreg = r
			return true
		}
		return false
	}
	dest := func() uint32 { return pipe.if_id.NPC + uint32(4*pipe.if_id.ins.Imm) }

	switch typ {
	case tREG2F, tREG2S:
		if chk(ins.rs, RAW) {
			return RAW
		}
		if chk(ins.rd, WAW) {
			return WAW
		}
		if chkW(ins.rd) {
			return WAR
		}
	case tREG2C, tSTORE, tFSTORE:
		if chk(ins.rs, RAW) {
			return RAW
		}
		if chk(ins.rt, RAW) {
			return RAW
		}
	case tREG2I, tLOAD, tFLOAD:
		if chk(ins.rs, RAW) {
			return RAW
		}
		if chk(ins.rt, RAW) {
			return RAW
		}
		if chkW(ins.rt) {
			return WAR
		}
	case tREG1I:
		if chk(ins.rt, RAW) {
			return RAW
		}
		if chkW(ins.rt) {
			return WAR
		}
	case tREG3:
		if chk(ins.rs, RAW) {
			return RAW
		}
		if chk(ins.rt, RAW) {
			return RAW
		}
		if chk(ins.rd, WAW) {
			return WAW
		}
		if chkW(ins.rd) {
			return WAR
		}
	case tREGDI, tREGID:
		if chk(ins.rt, RAW) {
			return RAW
		}
		if chk(ins.rd, WAW) {
			return WAW
		}
		if chkW(ins.rd) {
			return WAR
		}
	case tREG3F, tREG3X:
		var unit *idEXReg
		switch ins.function {
		case f_ADD_D, f_SUB_D:
			if pipe.a[0].active {
				return STALLED
			}
			unit = &pipe.a[0]
		case f_MUL_D, r_DMUL, r_DMULU:
			if pipe.m[0].active {
				return STALLED
			}
			unit = &pipe.m[0]
		case f_DIV_D, r_DDIV, r_DDIVU:
			if pipe.div.active {
				return STALLED
			}
			unit = &pipe.div
		}
		if unit != nil {
			if chk(ins.rs, RAW) {
				return RAW
			}
			if chk(ins.rt, RAW) {
				return RAW
			}
			if chk(ins.rd, WAW) {
				return WAW
			}
			if chkW(ins.rd) {
				return WAR
			}
			unit.rA = ins.rs
			unit.rB = ins.rt
			unit.active = true
			unit.NPC = pipe.if_id.NPC
			unit.IR = pipe.if_id.IR
			unit.ins = ins
			if unit == &pipe.div {
				pipe.div.cycles = pipe.DIV_LATENCY
			}
		}
		pipe.if_id.active = false
		return OK
	case tBRANCH:
		if chk(ins.rs, RAW) {
			return RAW
		}
		if chk(ins.rt, RAW) {
			return RAW
		}
		branchStatus = bRANCH_NOT_TAKEN
		predictable = true
		if ins.opcode == i_BEQ && A == B {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		if ins.opcode == i_BNE && A != B {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		pipe.if_id.active = false
	case tBC:
		branchStatus = bRANCH_NOT_TAKEN
		predictable = true
		if ins.tf != 0 && cpu.fp_cc {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		if ins.tf == 0 && !cpu.fp_cc {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		pipe.if_id.active = false
	case tJREGN:
		if chk(ins.rt, RAW) {
			return RAW
		}
		branchStatus = bRANCH_NOT_TAKEN
		predictable = true
		if B == 0 && ins.opcode == i_BEQZ {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		if B != 0 && ins.opcode == i_BNEZ {
			branchStatus = bRANCH_TAKEN
			pipe.branch = true
			pipe.destination = dest()
		}
		pipe.if_id.active = false
	case tJUMP:
		predictable = true
		branchStatus = bRANCH_TAKEN
		if ins.opcode == i_J {
			pipe.branch = true
			pipe.destination = dest()
			pipe.if_id.active = false
		}
		if ins.opcode == i_JAL {
			pipe.branch = true
			pipe.destination = dest()
		}
	case tJREG:
		if chk(ins.rt, RAW) {
			return RAW
		}
		branchStatus = bRANCH_TAKEN
		if ins.opcode == i_SPECIAL && ins.function == r_JR {
			pipe.branch = true
			pipe.destination = uint32(B)
			pipe.if_id.active = false
		}
		if ins.opcode == i_SPECIAL && ins.function == r_JALR {
			pipe.branch = true
			pipe.destination = uint32(B)
		}
	}

	status := OK
	if btb && predictable && !pipe.if_id.active {
		if branchStatus == bRANCH_TAKEN {
			status = BRANCH_TAKEN_STALL
			if (cpu.cstat[pipe.if_id.IR] & 2) == 0 { // throw in an extra stall...
				cpu.cstat[pipe.if_id.IR] |= 2
				pipe.if_id.active = true
				return status
			}
			if pipe.if_id.predicted {
				pipe.branch = false
				status = OK
			}
		}
		if branchStatus == bRANCH_NOT_TAKEN {
			if pipe.if_id.predicted { // it was predicted, but it didn't happen!
				status = BRANCH_MISPREDICTED_STALL
				if (cpu.cstat[pipe.if_id.IR] & 2) != 0 {
					cpu.cstat[pipe.if_id.IR] &= 0xfd
					pipe.if_id.active = true
					return status
				}
				// tell IF to re-fetch from here + 4 - costs another stall!
				pipe.branch = true
				pipe.destination = pipe.if_id.IR + 4
			}
		}
	} else if branchStatus == bRANCH_TAKEN {
		status = BRANCH_TAKEN_STALL
	}

	pipe.integer.NPC = pipe.if_id.NPC
	pipe.integer.IR = pipe.if_id.IR
	pipe.integer.ins = ins
	pipe.integer.Imm = ins.Imm
	pipe.integer.active = true
	pipe.if_id.active = false
	return status
}

func f64(u uint64) float64 { return math.Float64frombits(u) }
func u64(f float64) uint64 { return math.Float64bits(f) }

// cvtLD emulates (__int64)double as compiled for x86/x64 (cvttsd2si): NaN
// and out-of-range values give 0x8000000000000000.
func cvtLD(d float64) int64 {
	if d != d || d >= 9223372036854775808.0 || d < -9223372036854775808.0 {
		return math.MinInt64
	}
	return int64(d)
}

func exDiv(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int) int {
	status := EMPTY
	var fpR uint64 // fpR.d = 0.0

	ins := pipe.div.ins
	rA := pipe.div.rA
	rB := pipe.div.rB
	if pipe.div.active {
		if pipe.div.cycles == pipe.DIV_LATENCY { // Trying to start a new one...
			if !available(cpu, rA) {
				*rawreg = rA
				return RAW
			}
			if !available(cpu, rB) {
				*rawreg = rB
				return RAW
			}
			if waw(pipe, ins.typ, ins.function, ins.rd) {
				*rawreg = ins.rd
				return WAW
			}
			unavail(cpu, accBOTH, ins.rd)
			a := cpu.rreg[rA].val
			b := cpu.rreg[rB].val
			status = OK
			switch ins.function {
			case f_DIV_D:
				if f64(b) != 0.0 {
					fpR = u64(f64(a) / f64(b))
				} else {
					status = DIVIDE_BY_ZERO
				}
			case r_DDIV:
				if int64(b) != 0 {
					fpR = uint64(int64(a) / int64(b))
				} else {
					status = DIVIDE_BY_ZERO
				}
			case r_DDIVU:
				if b != 0 {
					fpR = a / b
				} else {
					status = DIVIDE_BY_ZERO
				}
			}
			pipe.div.ALUOutput = fpR
			pipe.div.cycles--
		} else if pipe.div.cycles > 0 {
			pipe.div.cycles--
			status = OK
		}
		if pipe.div.cycles == 0 {
			if !pipe.ex_mem.active { // finish divide
				pipe.ex_mem.ins = ins
				pipe.ex_mem.IR = pipe.div.IR
				pipe.ex_mem.ALUOutput = pipe.div.ALUOutput
				if forwarding {
					cpu.wreg[ins.rd].val = pipe.div.ALUOutput
					cpu.wreg[ins.rd].source = fROM_DIV
				}
				pipe.ex_mem.active = true
				pipe.ex_mem.NPC = pipe.div.NPC
				pipe.ex_mem.rB = pipe.div.rB
				pipe.div.active = false
				return status
			}
			return STALLED
		}
	}
	return status
}

// exFP implements EX_MUL (mul=true) and EX_ADD (mul=false), which are
// structurally identical in the original.
func exFP(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int, status []int, mul bool) {
	var u *[10]idEXReg
	var n int
	var src int32
	if mul {
		u, n, src = &pipe.m, pipe.MUL_LATENCY, fROM_MUL
	} else {
		u, n, src = &pipe.a, pipe.ADD_LATENCY, fROM_ADD
	}
	var idle idEXReg // idle.active = FALSE (other fields zero)

	for i := 0; i < n; i++ {
		status[i] = OK
	}
	ins := u[n-1].ins
	if u[n-1].active {
		if !pipe.ex_mem.active {
			pipe.ex_mem.ins = ins
			pipe.ex_mem.IR = u[n-1].IR
			pipe.ex_mem.ALUOutput = u[n-1].ALUOutput
			if forwarding {
				cpu.wreg[ins.rd].val = u[n-1].ALUOutput
				cpu.wreg[ins.rd].source = src
			}
			u[n-1].active = false
			pipe.ex_mem.active = true
			pipe.ex_mem.NPC = u[n-1].NPC
			pipe.ex_mem.rB = u[n-1].rB
		} else {
			status[n-1] = STALLED
		}
	} else {
		status[n-1] = EMPTY
	}

	for i := n - 1; i > 1; i-- {
		if !u[i].active {
			u[i] = u[i-1]
			u[i-1].active = false
		} else if u[i-1].active {
			status[i-1] = STALLED
		}
	}

	if !u[1].active {
		if u[0].active { // Trying to start a new one...
			rA := u[0].rA
			rB := u[0].rB
			ins = u[0].ins
			if !available(cpu, rA) {
				*rawreg = rA
				status[0] = RAW
				return
			}
			if !available(cpu, rB) {
				*rawreg = rB
				status[0] = RAW
				return
			}
			if waw(pipe, ins.typ, ins.function, ins.rd) {
				*rawreg = ins.rd
				status[0] = WAW
				return
			}
			unavail(cpu, accBOTH, ins.rd)
			a := cpu.rreg[rA].val
			b := cpu.rreg[rB].val
			var fpR uint64
			if mul {
				switch ins.function {
				case f_MUL_D:
					fpR = u64(f64(a) * f64(b))
				case r_DMUL:
					fpR = uint64(int64(a) * int64(b))
				case r_DMULU:
					fpR = a * b
				}
			} else {
				switch ins.function {
				case f_ADD_D:
					fpR = u64(f64(a) + f64(b))
				case f_SUB_D:
					fpR = u64(f64(a) - f64(b))
				}
			}
			u[0].ALUOutput = fpR
		}
		u[1] = u[0]
		u[0] = idle
	} else if u[0].active {
		status[0] = STALLED
	}
}

func exMul(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int, status []int) {
	exFP(pipe, cpu, forwarding, rawreg, status, true)
}

func exAdd(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int, status []int) {
	exFP(pipe, cpu, forwarding, rawreg, status, false)
}

func sext(imm int32) uint64 { return uint64(int64(imm)) }

func exInt(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int) int {
	condition := true
	status := OK

	if pipe.integer.active {
		if pipe.ex_mem.active {
			return STALLED
		}
	} else {
		return EMPTY
	}

	ins := pipe.integer.ins
	typ := ins.typ
	function := ins.function
	opcode := ins.opcode
	rA := pipe.integer.rA
	rB := pipe.integer.rB
	em := &pipe.ex_mem

	fwd := func(r int) {
		if forwarding && r != 0 {
			cpu.wreg[r].val = em.ALUOutput
			cpu.wreg[r].source = fROM_EX
		}
	}
	rv := func(r int) uint64 {
		if r < 0 {
			return 0
		}
		return cpu.rreg[r].val
	}

	switch typ {
	case tFLOAD:
		if waw(pipe, typ, function, ins.rt) {
			*rawreg = ins.rt
			return WAW
		}
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		unavail(cpu, accREAD, ins.rt)
		em.ALUOutput = rv(rA) + sext(ins.Imm)
	case tLOAD:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		unavail(cpu, accREAD, ins.rt)
		em.ALUOutput = rv(rA) + sext(ins.Imm)
	case tFSTORE, tSTORE:
		if waw(pipe, typ, function, ins.rt) {
			*rawreg = ins.rt
			return RAW
		}
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		em.ALUOutput = rv(rA) + sext(ins.Imm)
	case tREG1I:
		if waw(pipe, typ, function, ins.rt) {
			*rawreg = ins.rt
			return WAW
		}
		unavail(cpu, accBOTH, ins.rt)
		if opcode == i_LUI {
			em.ALUOutput = uint64(int64(ins.Imm) << 16) // thanks Katia
		}
		fwd(ins.rt)
	case tREG2I:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if waw(pipe, typ, function, ins.rt) {
			*rawreg = ins.rt
			return WAW
		}
		unavail(cpu, accBOTH, ins.rt)
		a := rv(rA)
		switch opcode {
		case i_DADDI:
			rlt := int64(a) + int64(ins.Imm)
			em.ALUOutput = uint64(rlt)
			if ins.Imm > 0 && rlt < int64(a) {
				status = INTEGER_OVERFLOW
			}
			if ins.Imm < 0 && rlt > int64(a) {
				status = INTEGER_OVERFLOW
			}
		case i_DADDIU:
			em.ALUOutput = a + sext(ins.Imm)
		case i_ANDI:
			em.ALUOutput = a & sext(ins.Imm&0xffff)
		case i_ORI:
			em.ALUOutput = a | sext(ins.Imm&0xffff)
		case i_XORI:
			em.ALUOutput = a ^ sext(ins.Imm&0xffff)
		case i_SLTI:
			if int64(a) < int64(ins.Imm) {
				em.ALUOutput = 1
			} else {
				em.ALUOutput = 0
			}
		case i_SLTIU:
			if a < sext(ins.Imm) {
				em.ALUOutput = 1
			} else {
				em.ALUOutput = 0
			}
		}
		fwd(ins.rt)
	case tREG2S:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if waw(pipe, typ, function, ins.rd) {
			*rawreg = ins.rd
			return WAW
		}
		unavail(cpu, accBOTH, ins.rd)
		a := rv(rA)
		sh := uint(ins.Imm)
		switch function {
		case r_DSLL:
			em.ALUOutput = a << sh
		case r_DSRL:
			em.ALUOutput = a >> sh
		case r_DSRA:
			em.ALUOutput = uint64(int64(a) >> sh)
		}
		fwd(ins.rd)
	case tJUMP, tJREG:
		if ins.opcode == i_JAL || (ins.opcode == i_SPECIAL && ins.function == r_JALR) {
			unavail(cpu, accBOTH, 31)
			if forwarding {
				cpu.wreg[31].val = uint64(pipe.integer.NPC)
				cpu.wreg[31].source = fROM_EX
			}
		}
	case tHALT:
	case tREG3:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if !available(cpu, rB) {
			*rawreg = rB
			return RAW
		}
		if waw(pipe, typ, function, ins.rd) {
			*rawreg = ins.rd
			return WAW
		}
		a := rv(rA)
		b := rv(rB)
		switch function {
		case r_AND:
			em.ALUOutput = a & b
		case r_OR:
			em.ALUOutput = a | b
		case r_XOR:
			em.ALUOutput = a ^ b
		case r_SLT:
			if int64(a) < int64(b) {
				em.ALUOutput = 1
			} else {
				em.ALUOutput = 0
			}
		case r_SLTU:
			if a < b {
				em.ALUOutput = 1
			} else {
				em.ALUOutput = 0
			}
		case r_DADD:
			rlt := int64(a) + int64(b)
			em.ALUOutput = uint64(rlt)
			if rlt < int64(a) {
				status = INTEGER_OVERFLOW
			}
		case r_DADDU:
			em.ALUOutput = a + b
		case r_DSUB:
			rlt := int64(a) - int64(b)
			em.ALUOutput = uint64(rlt)
			if rlt > int64(a) {
				status = INTEGER_OVERFLOW
			}
		case r_DSUBU:
			em.ALUOutput = a - b
		case r_DSLLV:
			em.ALUOutput = a << (b & 0x3F)
		case r_DSRLV:
			em.ALUOutput = a >> (b & 0x3F)
		case r_DSRAV:
			em.ALUOutput = uint64(int64(a) >> (b & 0x3F))
		case r_MOVZ:
			if b == 0 {
				em.ALUOutput = a
			} else {
				condition = false
			}
		case r_MOVN:
			if b != 0 {
				em.ALUOutput = a
			} else {
				condition = false
			}
		}
		if condition {
			unavail(cpu, accBOTH, ins.rd)
		}
		if condition {
			fwd(ins.rd)
		}
	case tREGID:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if waw(pipe, typ, function, ins.rd) {
			*rawreg = ins.rd
			return WAW
		}
		unavail(cpu, accBOTH, ins.rd)
		em.ALUOutput = rv(rA)
		fwd(ins.rd)
	case tREGDI:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if waw(pipe, typ, function, ins.rt) {
			*rawreg = ins.rt
			return WAW
		}
		unavail(cpu, accBOTH, ins.rt)
		em.ALUOutput = rv(rA)
		fwd(ins.rt)
	case tJREGN:
	case tREG2C:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if !available(cpu, rB) {
			*rawreg = rB
			return RAW
		}
		a := f64(rv(rA))
		b := f64(rv(rB))
		cpu.fp_cc = false
		switch function {
		case f_C_LT_D:
			if a < b {
				cpu.fp_cc = true
			}
		case f_C_LE_D:
			if a <= b {
				cpu.fp_cc = true
			}
		case f_C_EQ_D:
			if a == b {
				cpu.fp_cc = true
			}
		}
	case tREG2F:
		if !available(cpu, rA) {
			*rawreg = rA
			return RAW
		}
		if waw(pipe, typ, function, ins.rd) {
			*rawreg = ins.rd
			return WAW
		}
		unavail(cpu, accBOTH, ins.rd)
		a := rv(rA)
		switch function {
		case f_CVT_D_L:
			em.ALUOutput = u64(float64(int64(a)))
		case f_CVT_L_D:
			em.ALUOutput = uint64(cvtLD(f64(a)))
		case f_MOV_D:
			em.ALUOutput = a
		}
		fwd(ins.rd)
	}
	em.IR = pipe.integer.IR
	em.ins = ins
	em.NPC = pipe.integer.NPC
	em.rB = pipe.integer.rB
	em.active = true
	em.condition = condition
	pipe.integer.active = false
	return status
}

func memStage(pipe *pipeline, cpu *processor, forwarding bool, rawreg *int) int {
	pipe.mem_wb.active = false
	if !pipe.ex_mem.active {
		return EMPTY
	}
	condition := pipe.ex_mem.condition
	ins := pipe.ex_mem.ins
	pipe.mem_wb.ins = ins

	typ := ins.typ
	status := OK
	opcode := ins.opcode
	function := ins.function
	ptr := uint32(pipe.ex_mem.ALUOutput)
	mw := &pipe.mem_wb

	outside := ptr >= cpu.datasize && (ptr < mMIO || ptr >= mMIO+16)
	mmio := ptr >= mMIO && ptr < mMIO+16

	// mem returns the byte slice starting at ptr (MMIO or data memory)
	mem := func() []byte {
		if mmio {
			return cpu.mm[ptr-mMIO:]
		}
		return cpu.data[ptr:]
	}
	checkInit := func(n uint32) {
		if mmio {
			return
		}
		for i := uint32(0); i < n; i++ {
			if cpu.dstat[ptr+i] == vACANT {
				status = DATA_ERR
			}
		}
	}

	switch typ {
	case tLOAD, tFLOAD:
		status = LOADS
		unavail(cpu, accBOTH, ins.rt)
		if outside {
			mw.LMD = 0
			status = NO_SUCH_DATA_MEMORY
		} else {
			switch opcode {
			case i_LB:
				checkInit(1)
				mw.LMD = uint64(int64(int8(mem()[0])))
			case i_LBU:
				checkInit(1)
				mw.LMD = uint64(mem()[0])
			case i_LH:
				if ptr%2 != 0 {
					status = DATA_MISALIGNED
					mw.LMD = 0
					break
				}
				checkInit(2)
				mw.LMD = uint64(int64(int16(binary.LittleEndian.Uint16(mem()))))
			case i_LHU:
				if ptr%2 != 0 {
					status = DATA_MISALIGNED
					mw.LMD = 0
					break
				}
				checkInit(2)
				mw.LMD = uint64(binary.LittleEndian.Uint16(mem()))
			case i_LW:
				if ptr%4 != 0 {
					status = DATA_MISALIGNED
					mw.LMD = 0
					break
				}
				checkInit(4)
				mw.LMD = uint64(int64(int32(binary.LittleEndian.Uint32(mem()))))
			case i_LWU:
				if ptr%4 != 0 {
					status = DATA_MISALIGNED
					mw.LMD = 0
					break
				}
				checkInit(4)
				mw.LMD = uint64(binary.LittleEndian.Uint32(mem()))
			case i_LD, i_L_D:
				if ptr%8 != 0 {
					status = DATA_MISALIGNED
					mw.LMD = 0
					break
				}
				checkInit(8)
				mw.LMD = binary.LittleEndian.Uint64(mem())
			}
		}
		if forwarding && ins.rt != 0 {
			cpu.wreg[ins.rt].val = mw.LMD
			cpu.wreg[ins.rt].source = fROM_MEM
		}
	case tSTORE, tFSTORE:
		rB := pipe.ex_mem.rB
		if !available(cpu, rB) {
			*rawreg = rB
			return RAW
		}
		status = STORES
		var v uint64
		if rB >= 0 {
			v = cpu.rreg[rB].val
		}
		mark := func(n uint32) {
			if !mmio {
				for i := uint32(0); i < n; i++ {
					cpu.dstat[ptr+i] = wRITTEN
				}
			}
		}
		if outside {
			status = NO_SUCH_DATA_MEMORY
		} else {
			switch opcode {
			case i_SB:
				mark(1)
				mem()[0] = byte(v)
			case i_SH:
				if ptr%2 != 0 {
					status = DATA_MISALIGNED
					break
				}
				mark(2)
				binary.LittleEndian.PutUint16(mem(), uint16(v))
			case i_SW:
				if ptr%4 != 0 {
					status = DATA_MISALIGNED
					break
				}
				mark(4)
				binary.LittleEndian.PutUint32(mem(), uint32(v))
			case i_SD, i_S_D:
				if ptr%8 != 0 {
					status = DATA_MISALIGNED
					break
				}
				mark(8)
				binary.LittleEndian.PutUint64(mem(), v)
			}
		}
	case tREG3, tREG3X, tREG3F, tREG2S:
		if condition && forwarding && ins.rd != 0 {
			cpu.wreg[ins.rd].source = fROM_MEM
		}
	case tREG1I, tREG2I, tREGDI:
		if forwarding && ins.rt != 0 {
			cpu.wreg[ins.rt].source = fROM_MEM
		}
	case tREGID, tREG2F:
		if forwarding && ins.rd != 0 {
			cpu.wreg[ins.rd].source = fROM_MEM
		}
	case tJUMP, tJREG:
		if forwarding && (opcode == i_JAL || (opcode == i_SPECIAL && function == r_JALR)) {
			cpu.wreg[31].source = fROM_MEM
		}
	}

	pipe.ex_mem.active = false
	mw.active = true
	mw.ALUOutput = pipe.ex_mem.ALUOutput
	mw.IR = pipe.ex_mem.IR
	mw.ins = ins
	mw.NPC = pipe.ex_mem.NPC
	mw.condition = pipe.ex_mem.condition
	return status
}

// wbStage: Write Back. Since writing takes place on the leading edge, update
// both the read and write registers.
func wbStage(pipe *pipeline, cpu *processor, forwarding bool) int {
	ins := pipe.mem_wb.ins
	if !pipe.mem_wb.active {
		return EMPTY
	}
	condition := pipe.mem_wb.condition
	status := OK

	wb := func(r int, v uint64) {
		if !forwarding {
			makeAvailable(cpu, r, v)
		} else if cpu.rreg[r].source == fROM_MEM {
			cpu.rreg[r].source = fROM_REGISTER
			cpu.wreg[r].source = fROM_REGISTER
		}
	}

	switch ins.typ {
	case tLOAD, tFLOAD:
		if ins.rt != 0 {
			wb(ins.rt, pipe.mem_wb.LMD)
		}
	case tREG1I, tREG2I, tREGDI:
		if ins.rt != 0 {
			wb(ins.rt, pipe.mem_wb.ALUOutput)
		}
	case tREGID, tREG2F:
		if ins.rd != 0 {
			wb(ins.rd, pipe.mem_wb.ALUOutput)
		}
	case tREG3, tREG3X, tREG2S, tREG3F:
		if ins.rd != 0 && condition {
			wb(ins.rd, pipe.mem_wb.ALUOutput)
		}
	case tJUMP:
		if ins.opcode == i_JAL {
			wb(31, uint64(pipe.mem_wb.NPC))
		}
	case tJREG:
		if ins.opcode == i_SPECIAL && ins.function == r_JALR {
			wb(31, uint64(pipe.mem_wb.NPC))
		}
	case tHALT:
		status = HALTED
	}
	pipe.mem_wb.active = false
	return status
}

func clockTick(pipe *pipeline, mips *processor, forwarding, delaySlot, btb bool, res *result) int {
	// activate WB first as it activates on leading edge
	status := wbStage(pipe, mips, forwarding)
	if status == HALTED || pipe.halting { // check that pipeline is empty...
		pipe.halting = true
		empty := true
		for i := 0; i < pipe.MUL_LATENCY; i++ {
			if pipe.m[i].active {
				empty = false
			}
		}
		for i := 0; i < pipe.ADD_LATENCY; i++ {
			if pipe.a[i].active {
				empty = false
			}
		}
		if pipe.div.active && pipe.div.cycles > 0 {
			empty = false
		}
		if pipe.ex_mem.active {
			empty = false
		}
		if pipe.mem_wb.active {
			empty = false
		}
		if empty {
			res.WB, res.MEM, res.DIVIDER, res.EX, res.ID, res.IF = OK, OK, OK, OK, OK, OK
			for i := 0; i < pipe.MUL_LATENCY; i++ {
				res.MULTIPLIER[i] = OK
			}
			for i := 0; i < pipe.ADD_LATENCY; i++ {
				res.ADDER[i] = OK
			}
			return HALTED
		}
	}

	res.WB = status
	res.MEM = memStage(pipe, mips, forwarding, &res.memrr)
	exMul(pipe, mips, forwarding, &res.mulrr, res.MULTIPLIER[:])
	exAdd(pipe, mips, forwarding, &res.addrr, res.ADDER[:])
	res.DIVIDER = exDiv(pipe, mips, forwarding, &res.divrr)
	res.EX = exInt(pipe, mips, forwarding, &res.exrr)
	res.ID = idStage(pipe, mips, forwarding, btb, &res.idrr)
	res.IF = ifStage(pipe, mips, delaySlot, btb)

	// Copy Write to Read registers
	mips.rreg = mips.wreg
	return OK
}

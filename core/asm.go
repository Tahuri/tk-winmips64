package core

// Port of the two-pass assembler embedded in CWinMIPS64Doc
// (openit / first_pass / second_pass / directive / instruction).

import (
	"strings"
)

var directives = []string{
	".space",
	".asciiz",
	".align",
	".word",
	".byte",
	".ascii",
	".global",
	".data",
	".text",
	".org",
	".word32",
	".word16",
	".double",
	".code",
}

type opCodeInfo struct {
	name    string
	typ     int
	subtype int
	opCode  uint32
}

var codes = []opCodeInfo{
	{"lb", i_TYPE, tLOAD, sI(i_LB)},
	{"lbu", i_TYPE, tLOAD, sI(i_LBU)},
	{"sb", i_TYPE, tSTORE, sI(i_SB)},
	{"lh", i_TYPE, tLOAD, sI(i_LH)},
	{"lhu", i_TYPE, tLOAD, sI(i_LHU)},
	{"sh", i_TYPE, tSTORE, sI(i_SH)},
	{"lw", i_TYPE, tLOAD, sI(i_LW)},
	{"lwu", i_TYPE, tLOAD, sI(i_LWU)},
	{"sw", i_TYPE, tSTORE, sI(i_SW)},
	{"ld", i_TYPE, tLOAD, sI(i_LD)},
	{"sd", i_TYPE, tSTORE, sI(i_SD)},
	{"l.d", i_TYPE, tFLOAD, sI(i_L_D)},
	{"s.d", i_TYPE, tFSTORE, sI(i_S_D)},
	{"halt", i_TYPE, tHALT, sI(i_HALT)},

	{"daddi", i_TYPE, tREG2I, sI(i_DADDI)},
	{"daddui", i_TYPE, tREG2I, sI(i_DADDIU)},
	{"andi", i_TYPE, tREG2I, sI(i_ANDI)},
	{"ori", i_TYPE, tREG2I, sI(i_ORI)},
	{"xori", i_TYPE, tREG2I, sI(i_XORI)},
	{"lui", i_TYPE, tREG1I, sI(i_LUI)},

	{"slti", i_TYPE, tREG2I, sI(i_SLTI)},
	{"sltiu", i_TYPE, tREG2I, sI(i_SLTIU)},

	{"beq", i_TYPE, tBRANCH, sI(i_BEQ)},
	{"bne", i_TYPE, tBRANCH, sI(i_BNE)},
	{"beqz", i_TYPE, tJREGN, sI(i_BEQZ)},
	{"bnez", i_TYPE, tJREGN, sI(i_BNEZ)},

	{"j", j_TYPE, tJUMP, sI(i_J)},
	{"jr", r_TYPE, tJREG, sR(r_JR)},
	{"jal", j_TYPE, tJUMP, sI(i_JAL)},
	{"jalr", r_TYPE, tJREG, sR(r_JALR)},

	{"dsll", r_TYPE, tREG2S, sR(r_DSLL)},
	{"dsrl", r_TYPE, tREG2S, sR(r_DSRL)},
	{"dsra", r_TYPE, tREG2S, sR(r_DSRA)},
	{"dsllv", r_TYPE, tREG3, sR(r_DSLLV)},
	{"dsrlv", r_TYPE, tREG3, sR(r_DSRLV)},
	{"dsrav", r_TYPE, tREG3, sR(r_DSRAV)},
	{"movz", r_TYPE, tREG3, sR(r_MOVZ)},
	{"movn", r_TYPE, tREG3, sR(r_MOVN)},
	{"nop", r_TYPE, tNOP, sR(r_NOP)},
	{"and", r_TYPE, tREG3, sR(r_AND)},
	{"or", r_TYPE, tREG3, sR(r_OR)},
	{"xor", r_TYPE, tREG3, sR(r_XOR)},
	{"slt", r_TYPE, tREG3, sR(r_SLT)},
	{"sltu", r_TYPE, tREG3, sR(r_SLTU)},

	{"dadd", r_TYPE, tREG3, sR(r_DADD)},
	{"daddu", r_TYPE, tREG3, sR(r_DADDU)},
	{"dsub", r_TYPE, tREG3, sR(r_DSUB)},
	{"dsubu", r_TYPE, tREG3, sR(r_DSUBU)},

	{"dmul", r_TYPE, tREG3, sR(r_DMUL)},
	{"dmulu", r_TYPE, tREG3, sR(r_DMULU)},
	{"ddiv", r_TYPE, tREG3, sR(r_DDIV)},
	{"ddivu", r_TYPE, tREG3, sR(r_DDIVU)},

	{"add.d", f_TYPE, tREG3F, sF(f_ADD_D)},
	{"sub.d", f_TYPE, tREG3F, sF(f_SUB_D)},
	{"mul.d", f_TYPE, tREG3F, sF(f_MUL_D)},
	{"div.d", f_TYPE, tREG3F, sF(f_DIV_D)},
	{"mov.d", f_TYPE, tREG2F, sF(f_MOV_D)},
	{"cvt.d.l", f_TYPE, tREG2F, sF(f_CVT_D_L)},
	{"cvt.l.d", f_TYPE, tREG2F, sF(f_CVT_L_D)},
	{"c.lt.d", f_TYPE, tREG2C, sF(f_C_LT_D)},
	{"c.le.d", f_TYPE, tREG2C, sF(f_C_LE_D)},
	{"c.eq.d", f_TYPE, tREG2C, sF(f_C_EQ_D)},

	{"bc1f", b_TYPE, tBC, sBC1F},
	{"bc1t", b_TYPE, tBC, sBC1T},
	{"mtc1", m_TYPE, tREGID, sMTC1},
	{"mfc1", m_TYPE, tREGDI, sMFC1},
}

// cstr returns the C string starting at p (up to the first NUL).
func cstr(b []byte, p int) string {
	if p < 0 {
		return ""
	}
	e := p
	for e < len(b) && b[e] != 0 {
		e++
	}
	if p > len(b) {
		return ""
	}
	return string(b[p:e])
}

// mygets reads a line even if not terminated by CR (binary reads, text mode
// CRLF translation is done by the caller).
func mygets(src []byte, pos *int, max int) ([]byte, int) {
	var line []byte
	i := 0
	var ch byte
	for {
		if *pos >= len(src) {
			if i == 0 {
				return nil, 1
			}
			break
		}
		ch = src[*pos]
		*pos++
		if i < max-3 {
			line = append(line, ch)
			i++
		} else {
			line = append(line, '\n')
			for {
				if *pos >= len(src) {
					break
				}
				ch = src[*pos]
				*pos++
				if ch == '\n' || ch == 0 {
					break
				}
			}
			return line, -1
		}
		if ch == '\n' || ch == 0 {
			break
		}
	}
	if ch != '\n' {
		line = append(line, '\n')
	}
	return line, 0
}

func (s *Sim) getcodesym(b []byte, pp *int, m *uint32) bool {
	return getsym(s.codeTable, b, pp, m)
}

func (s *Sim) getdatasym(b []byte, pp *int, m *uint32) bool {
	return getsym(s.dataTable, b, pp, m)
}

func instructionIndex(b []byte, p int) int {
	if p == nilp {
		return -1
	}
	var text []byte
	i := 0
	for i < 10 && delimiter(at(b, p)) == 0 {
		text = append(text, cToLower(at(b, p)))
		i++
		p++
	}
	if i > 9 {
		return -1
	}
	t := string(text)
	for k := range codes {
		if codes[k].name == t {
			return k
		}
	}
	return -1
}

func (s *Sim) setDataline(idx uint32, line string) {
	if int(idx) < len(s.datalines) {
		s.datalines[idx] = line
	}
}

func (s *Sim) putData(v byte) {
	if int(s.dataptr) < len(s.cpu.data) {
		s.cpu.dstat[s.dataptr] = wRITTEN
		s.cpu.data[s.dataptr] = v
	}
	s.dataptr++
}

// directive processes assembler directives.
func (s *Sim) directive(pass int, b []byte, ptr int) bool {
	var num, m uint32
	var fw uint64
	var db float64
	var bb [8]byte
	line := cstr(b, 0)

	if ptr == nilp || at(b, ptr) != '.' {
		return false
	}
	k := 0
	for ; ; k++ {
		if k >= len(directives) {
			return false
		}
		if compare(b, ptr, []byte(directives[k])) == 0 {
			continue
		}
		break
	}
	for delimiter(at(b, ptr)) == 0 {
		ptr++
	}
	DATASIZE := uint32(s.DATASIZE)
	zero := true
	switch k {
	case 0: // .space
		if s.CODEORDATA == segCODE {
			return false
		}
		if !getnum(b, &ptr, &num) {
			return false
		}
		if num == 0 {
			return false
		}
		if s.CODEORDATA == segDATA {
			if pass == 2 && s.dataptr <= DATASIZE {
				s.setDataline(s.dataptr/sTEP, line)
			}
			s.dataptr += num
		}
		return true
	case 5, 1: // .ascii / .asciiz
		if k == 5 {
			zero = false
		}
		if s.CODEORDATA == segCODE {
			return false
		}
		ptr = eatwhite(b, ptr)
		if ptr == nilp {
			return true // original returns (-1), i.e. TRUE
		}
		if at(b, ptr) != '"' && at(b, ptr) != '\'' {
			return true
		}
		sc := at(b, ptr)
		ptr++
		num = 0
		iptr := ptr
		bs := false
		for at(b, iptr) != sc {
			if delimiter(at(b, iptr)) == dENDLINE {
				return false
			}
			if bs {
				num++
				bs = false
			} else {
				if at(b, ptr) == '\\' { // sic: tests *ptr, not *iptr
					bs = true
				} else {
					num++
				}
			}
			iptr++
		}
		if zero {
			num++ // trailing 0 needed
		}
		if s.CODEORDATA == segDATA {
			if pass == 1 {
				s.dataptr += num
				if s.dataptr > DATASIZE {
					return false
				}
			}
			if pass == 2 {
				s.setDataline(s.dataptr/sTEP, line)
				if zero && iptr < len(b) {
					b[iptr] = 0 // stuff in a zero
				}
				m = 0
				bs = false
				for m < num {
					m++
					if bs {
						ch := at(b, ptr)
						if ch == 'n' {
							ch = '\n'
						}
						s.putData(ch)
						bs = false
					} else {
						if at(b, ptr) == '\\' {
							bs = true
						} else {
							s.putData(at(b, ptr))
						}
					}
					ptr++
				}
			}
		}
		return true
	case 2: // .align
		if !getnum(b, &ptr, &num) {
			return false
		}
		if num < 2 || num > 16 {
			return false
		}
		if s.CODEORDATA == segCODE {
			return false
		}
		if s.CODEORDATA == segDATA {
			s.dataptr = uint32(alignment(int(s.dataptr), int(num)))
			if pass == 2 {
				s.setDataline(s.dataptr/sTEP, line)
			}
		}
		return true
	case 3, 12, 10, 11, 4: // .word .double .word32 .word16 .byte
		if s.CODEORDATA == segCODE {
			return false
		}
		step := uint32(sTEP)
		switch k {
		case 10:
			step = 4
		case 11:
			step = 2
		case 4:
			step = 1
		}
		if pass == 1 {
			if s.CODEORDATA == segDATA {
				for {
					s.dataptr += step
					if ptr = skipover(b, ptr, ','); ptr == nilp {
						break
					}
				}
				if s.dataptr > DATASIZE {
					return false
				}
			}
		}
		if pass == 2 {
			// get one value and store it
			get := func() bool {
				switch k {
				case 3:
					return getfullnum(b, &ptr, &fw)
				case 12:
					return getdouble(b, &ptr, &db)
				case 10:
					return s.getdatasym(b, &ptr, &num)
				case 11:
					if !getnum(b, &ptr, &num) {
						return false
					}
					return inRange(num, 0xffff)
				default:
					if !getnum(b, &ptr, &num) {
						return false
					}
					return inRange(num, 0xff)
				}
			}
			store := func() {
				switch k {
				case 3:
					unpack(fw, bb[:])
					for i := 0; i < sTEP; i++ {
						s.putData(bb[i])
					}
				case 12:
					unpack(u64(db), bb[:])
					for i := 0; i < sTEP; i++ {
						s.putData(bb[i])
					}
				case 10:
					unpack32(num, bb[:])
					for i := 0; i < 4; i++ {
						s.putData(bb[i])
					}
				case 11:
					unpack16(int16(uint16(num)), bb[:])
					for i := 0; i < 2; i++ {
						s.putData(bb[i])
					}
				default:
					s.putData(byte(num))
				}
			}
			if !get() {
				return false
			}
			if s.CODEORDATA == segDATA {
				s.setDataline(s.dataptr/sTEP, line)
				store()
				for {
					if ptr = skip(b, ptr, ','); ptr == nilp {
						break
					}
					if !get() {
						return false
					}
					store()
				}
			}
		}
		return true
	case 6, 7: // .global .data
		s.CODEORDATA = segDATA
		if pass == 1 {
			if eatwhite(b, ptr) != nilp {
				return false
			}
		}
		return true
	case 13, 8: // .code .text
		s.CODEORDATA = segCODE
		if pass == 1 {
			if eatwhite(b, ptr) != nilp {
				return false
			}
		}
		return true
	case 9: // .org
		if s.CODEORDATA == segDATA {
			if !getnum(b, &ptr, &num) {
				return false
			}
			if num < s.dataptr {
				return false
			}
			s.dataptr = uint32(alignment(int(num), sTEP))
			return true
		}
		if s.CODEORDATA == segCODE {
			if !getnum(b, &ptr, &num) {
				return false
			}
			if num < s.codeptr || num > uint32(s.CODESIZE) {
				return false
			}
			s.codeptr = uint32(alignment(int(num), 4))
			return true
		}
		return false
	}
	return false
}

// firstPass fills in symbol tables and checks for syntax errors.
// Returns (errors, code).
func (s *Sim) firstPass(b []byte) (int, string) {
	ptr := eatwhite(b, 0)
	if ptr == nilp {
		return 0, ""
	}
	if delimiter(at(b, ptr)) != 0 {
		return 0, ""
	}
	for {
		l := isSymbol(b, ptr)
		if l <= 0 {
			break
		}
		if s.CODEORDATA == 0 {
			return 1, "syntax"
		}
		if s.CODEORDATA == segCODE || s.CODEORDATA == segDATA {
			tab := &s.codeTable
			val := s.codeptr
			if s.CODEORDATA == segDATA {
				tab = &s.dataTable
				val = s.dataptr
			}
			if len(*tab) >= symTabSize {
				return 1, "out_of_memory"
			}
			sym := make([]byte, l)
			copy(sym, b[ptr:ptr+l])
			ptr += l
			*tab = append(*tab, symbol{symb: sym, value: val})
			ptr++ // skip over the ":"
		}
		ptr = eatwhite(b, ptr)
		if ptr == nilp {
			return 0, ""
		}
	}
	if instructionIndex(b, ptr) >= 0 {
		if s.CODEORDATA != segCODE {
			return 1, "syntax"
		}
		s.codeptr = uint32(alignment(int(s.codeptr), 4))
		s.codeptr += 4
		if s.codeptr > uint32(s.CODESIZE) {
			return 1, "out_of_memory"
		}
		return 0, ""
	}
	if s.directive(1, b, ptr) {
		return 0, ""
	}
	if at(b, ptr) == '.' {
		return 1, "bad_directive"
	}
	return 1, "bad_instruction"
}

// secondPass assembles one (tab-expanded, space padded) line.
func (s *Sim) secondPass(b []byte) (int, string) {
	var w, flags, codeWord uint32
	var bb [4]byte
	var rs, rt, rd int
	errCode := "syntax"
	errored := true

	line := cstr(b, 0)
	ptr := eatwhite(b, 0)
	if ptr == nilp {
		return 0, ""
	}
	// skip over any symbols on the line
	for isSymbol(b, ptr) != 0 {
		for at(b, ptr) != ':' {
			ptr++
		}
		ptr++
		ptr = eatwhite(b, ptr)
		if ptr == nilp {
			break
		}
	}
	instruct := instructionIndex(b, ptr)
	if instruct < 0 {
		if !s.directive(2, b, ptr) {
			if ptr != nilp && s.CODEORDATA == segDATA {
				return 1, "bad_directive"
			}
		}
		return 0, ""
	}

	start := ptr
	op := codes[instruct].opCode
	sub := codes[instruct].subtype
	typ := codes[instruct].typ
	for delimiter(at(b, ptr)) == 0 {
		ptr++
	}
	fin := ptr

	badReg := func(r int) bool {
		if r < 0 {
			errCode = "bad_register"
			return true
		}
		return false
	}
	sym := func(ok bool) bool {
		if !ok {
			errCode = "undefined_symbol"
		}
		return ok
	}
	// comma: ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break; ptr=eatwhite(ptr)
	comma := func(c byte) bool {
		ptr = skip(b, ptr, c)
		if eatwhite(b, ptr) == nilp {
			return false
		}
		ptr = eatwhite(b, ptr)
		return true
	}
	endOK := func() bool { return eatwhite(b, ptr) == nilp }
	rel := func() {
		w -= s.codeptr + 4 // relative jump
		w = uint32(int32(w) / 4)
	}

	switch sub {
	case tNOP, tHALT:
		if !endOK() {
			break
		}
		errored = false
	case tSTORE, tLOAD, tFSTORE, tFLOAD:
		if sub == tSTORE || sub == tLOAD {
			rt = getreg(b, &ptr)
		} else {
			rt = fgetreg(b, &ptr)
		}
		if badReg(rt) {
			break
		}
		if !comma(',') {
			break
		}
		if at(b, ptr) == '(' {
			w = 0
		} else if !sym(s.getdatasym(b, &ptr, &w)) {
			break
		}
		if !comma('(') {
			break
		}
		rs = getreg(b, &ptr)
		if badReg(rs) {
			break
		}
		ptr = skip(b, ptr, ')')
		if ptr == nilp {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG2I:
		rt = getreg(b, &ptr)
		if badReg(rt) || !comma(',') {
			break
		}
		rs = getreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		if !sym(s.getdatasym(b, &ptr, &w)) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG1I:
		rt = getreg(b, &ptr)
		if badReg(rt) || !comma(',') {
			break
		}
		if !sym(s.getdatasym(b, &ptr, &w)) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tJREG:
		rt = getreg(b, &ptr)
		if badReg(rt) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG3:
		rd = getreg(b, &ptr)
		if badReg(rd) || !comma(',') {
			break
		}
		rs = getreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		rt = getreg(b, &ptr)
		if badReg(rt) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG3F:
		rd = fgetreg(b, &ptr)
		if badReg(rd) || !comma(',') {
			break
		}
		rs = fgetreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		rt = fgetreg(b, &ptr)
		if badReg(rt) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG2F:
		rd = fgetreg(b, &ptr)
		if badReg(rd) || !comma(',') {
			break
		}
		rs = fgetreg(b, &ptr)
		if badReg(rs) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG2C:
		rs = fgetreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		rt = fgetreg(b, &ptr)
		if badReg(rt) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREGID, tREGDI:
		rt = getreg(b, &ptr)
		if badReg(rt) || !comma(',') {
			break
		}
		rd = fgetreg(b, &ptr)
		if badReg(rd) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tREG2S:
		rd = getreg(b, &ptr)
		if badReg(rd) || !comma(',') {
			break
		}
		rs = getreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		if !sym(s.getdatasym(b, &ptr, &flags)) {
			break
		}
		if !endOK() {
			break
		}
		errored = false
	case tJUMP, tBC:
		if !sym(s.getcodesym(b, &ptr, &w)) {
			break
		}
		rel()
		if !endOK() {
			break
		}
		errored = false
	case tBRANCH:
		rt = getreg(b, &ptr)
		if badReg(rt) || !comma(',') {
			break
		}
		rs = getreg(b, &ptr)
		if badReg(rs) || !comma(',') {
			break
		}
		if !sym(s.getcodesym(b, &ptr, &w)) {
			break
		}
		rel()
		if !endOK() {
			break
		}
		errored = false
	case tJREGN:
		rt = getreg(b, &ptr)
		if badReg(rt) || !comma(',') {
			break
		}
		if !sym(s.getcodesym(b, &ptr, &w)) {
			break
		}
		rel()
		if !endOK() {
			break
		}
		errored = false
	default:
		errored = false
	}

	urs, urt, urd := uint32(rs), uint32(rt), uint32(rd)
	if !errored {
		switch typ {
		case i_TYPE:
			if inRange(w, 0xffff) {
				codeWord = op | urs<<21 | urt<<16 | (w & 0xffff)
			} else {
				errored = true
			}
		case r_TYPE:
			if inRange(flags, 0x1F) {
				codeWord = op | urs<<21 | urt<<16 | urd<<11 | flags<<6
			} else {
				errored = true
			}
		case j_TYPE:
			if inRange(w, 0x3ffffff) {
				codeWord = op | w&0x3ffffff
			} else {
				errored = true
			}
		case f_TYPE:
			codeWord = op | urs<<11 | urt<<16 | urd<<6
		case m_TYPE:
			codeWord = op | urt<<16 | urd<<11
		case b_TYPE:
			if inRange(w, 0xffff) {
				codeWord = op | w&0xffff
			} else {
				errored = true
			}
		}
		if errored {
			errCode = "bad_number"
		}
	}

	s.codeptr = uint32(alignment(int(s.codeptr), 4))
	idx := int(s.codeptr / 4)
	ret := 0
	if errored {
		codeWord = 0
		if int(s.codeptr) < len(s.cpu.cstat) {
			s.cpu.cstat[s.codeptr] = 4
		}
		ret = 1
	}
	if idx < len(s.codelines) {
		if ptr == nilp {
			s.assembly[idx] = ""
			s.mnemonic[idx] = ""
		} else {
			l := ptr - start
			if l > 25 {
				l = 25
			}
			if l < 0 {
				l = 0
			}
			s.assembly[idx] = cstrN(b, start, l)
			l = fin - start
			if l > 7 {
				l = 7
			}
			s.mnemonic[idx] = strings.ToLower(cstrN(b, start, l))
		}
		s.codelines[idx] = line
	}
	unpack32(codeWord, bb[:])
	for i := 0; i < 4; i++ {
		if int(s.codeptr) < len(s.cpu.code) {
			s.cpu.code[s.codeptr] = bb[i]
		}
		s.codeptr++
	}
	if ret != 0 {
		return ret, errCode
	}
	return 0, ""
}

// cstrN emulates CString(start, len): len bytes (stops at NUL for display).
func cstrN(b []byte, p, n int) string {
	out := make([]byte, 0, n)
	for i := 0; i < n; i++ {
		c := at(b, p+i)
		if c == 0 {
			break
		}
		out = append(out, c)
	}
	return string(out)
}

// openit assembles src. Returns 0 (ok), 2 (pass 1 errors), 3 (pass 2 errors)
// plus the list of errors.
func (s *Sim) openit(srcText string) (int, []AsmError) {
	src := textMode(srcText)
	var errs []AsmError
	errors := 0
	s.CODEORDATA = 0
	s.codeptr = 0
	s.dataptr = 0
	s.codeTable = s.codeTable[:0]
	s.dataTable = s.dataTable[:0]

	srcLine := func(l []byte) string {
		t := string(l)
		if i := strings.IndexByte(t, 0); i >= 0 {
			t = t[:i]
		}
		return strings.TrimRight(t, "\n")
	}

	pos := 0
	for lineptr := 1; ; lineptr++ {
		l, got := mygets(src, &pos, maxLine)
		if got == 1 {
			break
		}
		code := ""
		if got == -1 { // something got chopped
			errors++
			code = "syntax"
		}
		b := append(append([]byte{}, l...), 0)
		e, c := s.firstPass(b)
		errors += e
		if code == "" {
			code = c
		}
		if errors != 0 {
			errs = append(errs, AsmError{Line: lineptr, Code: code, Text: srcLine(l)})
			break
		}
		s.dataptr = uint32(alignment(int(s.dataptr), sTEP))
	}
	s.CODEORDATA = 0
	s.codeptr = 0
	s.dataptr = 0
	if errors != 0 {
		return 2, errs
	}

	errors = 0
	pos = 0
	for lineptr := 1; ; lineptr++ {
		pre, got := mygets(src, &pos, maxLine)
		if got == 1 {
			break
		}
		// preline[strlen(preline)-1]=0; remove CR/LF
		n := len(pre)
		for i, c := range pre {
			if c == 0 {
				n = i
				break
			}
		}
		if n > 0 {
			pre = pre[:n-1]
		} else {
			pre = pre[:0]
		}
		// replace tabs with 2 spaces, pad with spaces to MAX_LINE
		lb := make([]byte, 0, maxLine+1)
		for _, c := range pre {
			if c == '\t' {
				lb = append(lb, ' ', ' ')
			} else {
				lb = append(lb, c)
			}
		}
		lb = append(lb, ' ')
		for len(lb) < maxLine {
			lb = append(lb, ' ')
		}
		lb = append(lb[:maxLine:maxLine], 0)

		e, c := s.secondPass(lb)
		if e != 0 {
			errs = append(errs, AsmError{Line: lineptr, Code: c, Text: string(pre)})
		}
		errors += e
		s.dataptr = uint32(alignment(int(s.dataptr), sTEP))
	}
	if errors != 0 {
		return 3, errs
	}
	return 0, nil
}

// textMode emulates reading the file through a Windows text-mode stream:
// CR LF becomes LF and Ctrl-Z (0x1A) ends the file.
func textMode(t string) []byte {
	out := make([]byte, 0, len(t))
	for i := 0; i < len(t); i++ {
		c := t[i]
		if c == 0x1A {
			break
		}
		if c == '\r' && i+1 < len(t) && t[i+1] == '\n' {
			continue
		}
		out = append(out, c)
	}
	return out
}

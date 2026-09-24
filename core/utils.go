package core

import (
	"encoding/binary"
	"math"
	"strconv"
)

// C-string helpers. A "char*" is modelled as an index into a NUL-terminated
// byte buffer; nilp (-1) stands for the NULL pointer. Reads past the end of
// the buffer return 0, like the NUL terminator.

const nilp = -1

func at(b []byte, p int) byte {
	if p < 0 || p >= len(b) {
		return 0
	}
	return b[p]
}

func cIsDigit(c byte) bool { return c >= '0' && c <= '9' }
func cIsAlpha(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func cIsAlnum(c byte) bool { return cIsDigit(c) || cIsAlpha(c) }
func cIsSpace(c byte) bool { return c == ' ' || (c >= 9 && c <= 13) }
func cToLower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c + 32
	}
	return c
}

func bits(num int) int {
	r := 0
	for {
		num >>= 1
		if num == 0 {
			break
		}
		r++
	}
	return r
}

func pack32(b []byte) uint32    { return binary.LittleEndian.Uint32(b) }
func pack(b []byte) uint64      { return binary.LittleEndian.Uint64(b) }
func unpack(a uint64, b []byte) { binary.LittleEndian.PutUint64(b, a) }

func unpack32(a uint32, b []byte) { binary.LittleEndian.PutUint32(b, a) }

func unpack16(a int16, b []byte) {
	b[0] = byte(a)
	b[1] = byte(a >> 8)
}

// in_range: check that num will fit into the number of bits in mask
func inRange(num uint32, mask uint32) bool {
	n := int32(num)
	if n >= 0 && (num&mask) != num {
		return false
	}
	if n < 0 && (num|mask) != 0xFFFFFFFF {
		return false
	}
	return true
}

func alignment(ptr, num int) int {
	r := ptr
	t := r % num
	if t > 0 {
		r += num - t
	}
	return r
}

func delimiter(c byte) int {
	if c == ';' || c == '#' {
		return dCOMMENT
	}
	if c == 0 || c == '\n' {
		return dENDLINE
	}
	if c == ' ' || c == 9 {
		return dSPACE
	}
	return 0
}

// compare strings, up to 0 in the second with all of the first
func compare(b []byte, p int, s []byte) int {
	if p == nilp {
		return 0
	}
	incr := 0
	j := 0
	sat := func(j int) byte {
		if j < len(s) {
			return s[j]
		}
		return 0
	}
	for sat(j) == at(b, p) {
		if sat(j) == 0 {
			break
		}
		p++
		j++
		incr++
	}
	if sat(j) == 0 && !cIsAlnum(at(b, p)) {
		return incr
	}
	return 0
}

func skipover(b []byte, p int, c byte) int {
	for at(b, p) != c {
		res := delimiter(at(b, p))
		if res == dENDLINE || res == dCOMMENT {
			return nilp
		}
		if c == ',' && at(b, p) == '(' {
			return nilp
		}
		p++
	}
	p++
	return p
}

func skip(b []byte, p int, c byte) int {
	p = eatwhite(b, p)
	if p == nilp {
		return nilp
	}
	if at(b, p) != c {
		return nilp
	}
	return p + 1
}

func isSymbol(b []byte, start int) int {
	l := 0
	p := start
	for delimiter(at(b, p)) == 0 {
		p++
		l++
	}
	p--
	l--
	if at(b, p) == ':' {
		return l
	}
	return 0
}

func eatwhite(b []byte, p int) int {
	if p == nilp {
		return nilp
	}
	for at(b, p) == ' ' || at(b, p) == 9 {
		p++
	}
	c := at(b, p)
	if c == 0 || c == ';' || c == '#' || c == '\n' {
		return nilp
	}
	return p
}

// cAtoi emulates atoi(): whitespace, optional sign, decimal digits.
func cAtoi(b []byte, p int) int {
	for cIsSpace(at(b, p)) {
		p++
	}
	neg := false
	if at(b, p) == '-' || at(b, p) == '+' {
		neg = at(b, p) == '-'
		p++
	}
	var n int64
	for cIsDigit(at(b, p)) {
		n = n*10 + int64(at(b, p)-'0')
		if n > 1<<40 {
			n = 1 << 40
		}
		p++
	}
	if neg {
		n = -n
	}
	return int(int32(n))
}

func hasPrefix(b []byte, p int, s string) bool {
	for i := 0; i < len(s); i++ {
		if at(b, p+i) != s[i] {
			return false
		}
	}
	return true
}

func getreg(b []byte, pp *int) int {
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return -1
	}
	ch := cToLower(at(b, ptr))
	if ch != 'r' && ch != '$' {
		return -1
	}
	ptr++
	*pp = ptr
	if ch == 'r' && !cIsDigit(at(b, ptr)) {
		return -1
	}
	typ := 0
	switch {
	case hasPrefix(b, ptr, "zero"):
		*pp = ptr + 4
		return 0
	case hasPrefix(b, ptr, "at"):
		*pp = ptr + 2
		return 1
	case hasPrefix(b, ptr, "gp"):
		*pp = ptr + 2
		return 28
	case hasPrefix(b, ptr, "sp"):
		*pp = ptr + 2
		return 29
	case hasPrefix(b, ptr, "fp"):
		*pp = ptr + 2
		return 30
	case hasPrefix(b, ptr, "ra"):
		*pp = ptr + 2
		return 31
	}
	if hasPrefix(b, ptr, "v") {
		typ = 1
		ptr++
	}
	if hasPrefix(b, ptr, "a") {
		typ = 2
		ptr++
	}
	if hasPrefix(b, ptr, "t") {
		typ = 3
		ptr++
	}
	if hasPrefix(b, ptr, "s") {
		typ = 4
		ptr++
	}
	if hasPrefix(b, ptr, "k") {
		typ = 5
		ptr++
	}
	n := cAtoi(b, ptr)
	ptr++
	for cIsDigit(at(b, ptr)) {
		ptr++
	}
	*pp = ptr
	switch typ {
	case 1:
		if n < 0 || n > 1 {
			return -1
		}
		n += 2
	case 2:
		if n < 0 || n > 3 {
			return -1
		}
		n += 4
	case 3:
		if n < 0 || n > 9 {
			return -1
		}
		if n < 8 {
			n += 8
		} else {
			n += 16
		}
	case 4:
		if n < 0 || n > 7 {
			return -1
		}
		n += 16
	case 5:
		if n < 0 || n > 1 {
			return -1
		}
		n += 26
	}
	if n < 0 || n > 31 {
		return -1
	}
	return n
}

func fgetreg(b []byte, pp *int) int {
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return -1
	}
	if cToLower(at(b, ptr)) != 'f' {
		return -1
	}
	ptr++
	*pp = ptr
	if !cIsDigit(at(b, ptr)) {
		return -1
	}
	n := cAtoi(b, ptr)
	ptr++
	for cIsDigit(at(b, ptr)) {
		ptr++
	}
	*pp = ptr
	if n < 0 || n > 31 {
		return -1
	}
	return n
}

// strtoint64 is the original's own 64-bit parser (wraps on overflow).
func strtoint64(b []byte, p int, end *int, base0 int) uint64 {
	var n uint64
	s := 0
	base := uint64(10)
	for at(b, p) == ' ' || at(b, p) == 9 {
		p++
	}
	if at(b, p) == '-' {
		s = 1
		p++
	} else if at(b, p) == '+' {
		p++
	}
	for at(b, p) == ' ' || at(b, p) == 9 {
		p++
	}
	if base0 > 0 {
		base = uint64(base0)
	} else if at(b, p) == '0' {
		p++
		if at(b, p) == 'x' || at(b, p) == 'X' {
			base = 16
			p++
		} else {
			base = 8
		}
	}
	for {
		ch := at(b, p)
		if base == 8 {
			if ch < '0' || ch > '7' {
				break
			}
			n = n*base + uint64(ch-'0')
		}
		if base == 10 {
			if ch < '0' || ch > '9' {
				break
			}
			n = n*base + uint64(ch-'0')
		}
		if base == 16 {
			if (ch < '0' || ch > '9') && (ch < 'A' || ch > 'F') && (ch < 'a' || ch > 'f') {
				break
			}
			if ch >= '0' && ch <= '9' {
				n = n*base + uint64(ch-'0')
			}
			if ch >= 'A' && ch <= 'F' {
				n = n*base + 10 + uint64(ch-'A')
			}
			if ch >= 'a' && ch <= 'f' {
				n = n*base + 10 + uint64(ch-'a')
			}
		}
		p++
	}
	if end != nil {
		*end = p
	}
	if s == 1 {
		return ^n + 1
	}
	return n
}

func digitVal(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'z':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'Z':
		return int(c-'A') + 10
	}
	return 99
}

// cStrtoul32 emulates strtoul(ptr,&end,0) with a 32-bit unsigned long
// (Windows): saturates to ULONG_MAX on overflow, negates for '-'.
func cStrtoul32(b []byte, p int, end *int) uint32 {
	start := p
	for cIsSpace(at(b, p)) {
		p++
	}
	neg := false
	if at(b, p) == '-' || at(b, p) == '+' {
		neg = at(b, p) == '-'
		p++
	}
	base := 10
	if at(b, p) == '0' {
		if (at(b, p+1) == 'x' || at(b, p+1) == 'X') && digitVal(at(b, p+2)) < 16 {
			base = 16
			p += 2
		} else {
			base = 8
		}
	}
	var n uint64
	any := false
	over := false
	for {
		d := digitVal(at(b, p))
		if d >= base {
			break
		}
		any = true
		n = n*uint64(base) + uint64(d)
		if n > 0xFFFFFFFF {
			over = true
			n = 0xFFFFFFFF
		}
		p++
	}
	if !any {
		*end = start
		return 0
	}
	*end = p
	if over {
		return 0xFFFFFFFF
	}
	r := uint32(n)
	if neg {
		r = -r
	}
	return r
}

func getfullnum(b []byte, pp *int, num *uint64) bool {
	*num = 0
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return false
	}
	c := at(b, ptr)
	if !cIsDigit(c) && c != '-' && c != '+' {
		return false
	}
	*num = strtoint64(b, ptr, pp, 0)
	return true
}

// cStrtod emulates C strtod (decimal, hex, inf, nan). Returns value and the
// number of bytes consumed from p (0 = no conversion).
func cStrtod(b []byte, p int) (float64, int) {
	start := p
	for cIsSpace(at(b, p)) {
		p++
	}
	neg := false
	if at(b, p) == '-' || at(b, p) == '+' {
		neg = at(b, p) == '-'
		p++
	}
	sign := func(v float64) float64 {
		if neg {
			return -v
		}
		return v
	}
	lowerMatch := func(q int, s string) bool {
		for i := 0; i < len(s); i++ {
			if cToLower(at(b, q+i)) != s[i] {
				return false
			}
		}
		return true
	}
	if lowerMatch(p, "inf") {
		q := p + 3
		if lowerMatch(q, "inity") {
			q += 5
		}
		return sign(math.Inf(1)), q - start
	}
	if lowerMatch(p, "nan") {
		q := p + 3
		if at(b, q) == '(' {
			r := q + 1
			for cIsAlnum(at(b, r)) || at(b, r) == '_' {
				r++
			}
			if at(b, r) == ')' {
				q = r + 1
			}
		}
		return sign(math.NaN()), q - start
	}
	if at(b, p) == '0' && (at(b, p+1) == 'x' || at(b, p+1) == 'X') {
		q := p + 2
		mant := []byte{}
		nd := 0
		for digitVal(at(b, q)) < 16 {
			mant = append(mant, at(b, q))
			q++
			nd++
		}
		if at(b, q) == '.' {
			mant = append(mant, '.')
			q++
			for digitVal(at(b, q)) < 16 {
				mant = append(mant, at(b, q))
				q++
				nd++
			}
		}
		if nd > 0 {
			exp := "p0"
			if at(b, q) == 'p' || at(b, q) == 'P' {
				r := q + 1
				if at(b, r) == '-' || at(b, r) == '+' {
					r++
				}
				if cIsDigit(at(b, r)) {
					for cIsDigit(at(b, r)) {
						r++
					}
					exp = "p" + string(b[q+1:r])
					q = r
				}
			}
			s := "0x" + string(mant) + exp
			if mant[0] == '.' {
				s = "0x0" + string(mant) + exp
			}
			v, _ := strconv.ParseFloat(s, 64)
			return sign(v), q - start
		}
		// "0x" without digits: parse the "0"
		return sign(0), p + 1 - start
	}
	q := p
	nd := 0
	for cIsDigit(at(b, q)) {
		q++
		nd++
	}
	if at(b, q) == '.' {
		q++
		for cIsDigit(at(b, q)) {
			q++
			nd++
		}
	}
	if nd == 0 {
		return 0, 0
	}
	if at(b, q) == 'e' || at(b, q) == 'E' {
		r := q + 1
		if at(b, r) == '-' || at(b, r) == '+' {
			r++
		}
		if cIsDigit(at(b, r)) {
			for cIsDigit(at(b, r)) {
				r++
			}
			q = r
		}
	}
	s := string(b[p:q])
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		// range errors: ParseFloat returns ±Inf or 0 like strtod
		if ne, ok := err.(*strconv.NumError); ok && ne.Err == strconv.ErrRange {
			return sign(v), q - start
		}
		return 0, 0
	}
	return sign(v), q - start
}

func getdouble(b []byte, pp *int, num *float64) bool {
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return false
	}
	c := at(b, ptr)
	if !cIsDigit(c) && c != '.' && c != '-' && c != '+' {
		return false
	}
	v, n := cStrtod(b, ptr)
	*pp = ptr + n
	*num = v
	return true
}

func getnum(b []byte, pp *int, n *uint32) bool {
	sign := uint32(1)
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return false
	}
	c := at(b, ptr)
	if !cIsDigit(c) && c != '-' && c != '+' {
		return false
	}
	if at(b, ptr) == '-' {
		sign = 0xFFFFFFFF
		ptr++
	}
	if at(b, ptr) == '+' {
		ptr++
	}
	var end int
	*n = cStrtoul32(b, ptr, &end)
	*pp = end
	*n *= sign
	return true
}

// getsym: get symbol (or number) from a symbol table
func getsym(table []symbol, b []byte, pp *int, n *uint32) bool {
	ptr := eatwhite(b, *pp)
	*pp = ptr
	if ptr == nilp {
		return false
	}
	var m uint32
	if getnum(b, pp, &m) {
		*n = m
		return true
	}
	ptr = *pp
	for k := range table {
		incr := compare(b, ptr, table[k].symb)
		if incr != 0 {
			*n = table[k].value
			ptr += incr
			*pp = ptr
			if eatwhite(b, ptr) == nilp {
				return true
			}
			ptr = eatwhite(b, ptr)
			*pp = ptr
			if at(b, ptr) == '-' || at(b, ptr) == '+' {
				if getnum(b, pp, &m) {
					*n += m
				}
			}
			return true
		}
	}
	return false
}

// cAtoi64 emulates _atoi64 / atoll (saturating).
func cAtoi64(s string) int64 {
	i := 0
	for i < len(s) && cIsSpace(s[i]) {
		i++
	}
	neg := false
	if i < len(s) && (s[i] == '-' || s[i] == '+') {
		neg = s[i] == '-'
		i++
	}
	var n uint64
	over := false
	for i < len(s) && cIsDigit(s[i]) {
		n = n*10 + uint64(s[i]-'0')
		if n > 1<<63 {
			over = true
			n = 1 << 63
		}
		i++
	}
	if neg {
		if over || n >= 1<<63 {
			return math.MinInt64
		}
		return -int64(n)
	}
	if over || n >= 1<<63 {
		return math.MaxInt64
	}
	return int64(n)
}

// cAtof emulates atof.
func cAtof(s string) float64 {
	b := append([]byte(s), 0)
	v, _ := cStrtod(b, 0)
	return v
}

var conventionRegisterName = [32]string{
	"0 ", "at", "v0", "v1", "a0", "a1", "a2", "a3",
	"t0", "t1", "t2", "t3", "t4", "t5", "t6", "t7",
	"s0", "s1", "s2", "s3", "s4", "s5", "s6", "s7",
	"t8", "t9", "k0", "k1", "gp", "sp", "fp", "ra",
}

// registerName returns the display name (without the trailing "=  ").
func registerName(regnum int, asNumber bool) string {
	if asNumber {
		return "R" + strconv.Itoa(regnum)
	}
	n := conventionRegisterName[regnum]
	if n == "0 " {
		n = "0"
	}
	return "$" + n
}

func initProcessor(cpu *processor, codesize, datasize int) {
	for i := 0; i < 64; i++ {
		cpu.rreg[i].val, cpu.wreg[i].val = 0, 0
		cpu.rreg[i].source, cpu.wreg[i].source = fROM_REGISTER, fROM_REGISTER
	}
	cpu.PC = 0
	cpu.codesize = uint32(codesize)
	cpu.datasize = uint32(datasize)
	cpu.code = nil
	cpu.data = nil
	cpu.status = cpuRUNNING
}

func initPipeline(pipe *pipeline, adds, muls, divs int) {
	pipe.branch = false
	pipe.destination = 0
	pipe.ADD_LATENCY = adds
	pipe.MUL_LATENCY = muls
	pipe.DIV_LATENCY = divs
	pipe.if_id.active = false
	pipe.integer.active = false
	pipe.ex_mem.active = false
	pipe.mem_wb.active = false
	for i := 0; i < adds; i++ {
		pipe.a[i].active = false
	}
	for i := 0; i < muls; i++ {
		pipe.m[i].active = false
	}
	pipe.div.active = false
	pipe.div.cycles = 0
	pipe.active = true
	pipe.halting = false
	pipe.mem_wb.condition = true
	pipe.ex_mem.condition = true
}

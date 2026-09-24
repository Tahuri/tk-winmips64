package core

// Pipeline history ("Cycles" window). The original keeps
//
//	record history[50];  record = { WORD32 IR; WORD32 start_cycle; entry status[500]; }
//
// and update_history() writes status[cc] fields individually (substage is not
// always written, so stale bytes from earlier records leak into the output).
// Two stores implement the same algorithm:
//   - flatHist: byte-exact emulation of the 50-entry array (used by Trace),
//     including stale bytes and out-of-bounds status writes spilling into the
//     next record, as in the C memory layout.
//   - uiHist: an efficient store with a configurable number of entries for
//     the UI Snapshot.

type histStore interface {
	n() int
	setN(int)
	ir(i int) uint32
	setIR(i int, v uint32)
	start(i int) uint32
	setStart(i int, v uint32)
	get(i, cc int) (stage, sub, cause byte)
	setStage(i, cc int, v byte)
	setSub(i, cc int, v byte)
	setCause(i, cc int, v byte)
	shift() // history[i] = history[i+1] for i < n()
	capacity() int
}

const (
	recStatus = 500
	recSize   = 8 + 3*recStatus // 1508 bytes
	flatRecs  = 50
)

type flatHist struct {
	buf     []byte
	entries int
}

func newFlatHist() *flatHist { return &flatHist{buf: make([]byte, flatRecs*recSize), entries: 1} }

func (h *flatHist) rd(off int) byte {
	if off < 0 || off >= len(h.buf) {
		return 0
	}
	return h.buf[off]
}
func (h *flatHist) wr(off int, v byte) {
	if off < 0 || off >= len(h.buf) {
		return // would clobber other CWinMIPS64Doc members; dropped
	}
	h.buf[off] = v
}
func (h *flatHist) rd32(off int) uint32 {
	return uint32(h.rd(off)) | uint32(h.rd(off+1))<<8 | uint32(h.rd(off+2))<<16 | uint32(h.rd(off+3))<<24
}
func (h *flatHist) wr32(off int, v uint32) {
	h.wr(off, byte(v))
	h.wr(off+1, byte(v>>8))
	h.wr(off+2, byte(v>>16))
	h.wr(off+3, byte(v>>24))
}
func sOff(i, cc int) int { return i*recSize + 8 + 3*cc }

func (h *flatHist) n() int                   { return h.entries }
func (h *flatHist) setN(v int)               { h.entries = v }
func (h *flatHist) capacity() int            { return flatRecs }
func (h *flatHist) ir(i int) uint32          { return h.rd32(i * recSize) }
func (h *flatHist) setIR(i int, v uint32)    { h.wr32(i*recSize, v) }
func (h *flatHist) start(i int) uint32       { return h.rd32(i*recSize + 4) }
func (h *flatHist) setStart(i int, v uint32) { h.wr32(i*recSize+4, v) }
func (h *flatHist) get(i, cc int) (byte, byte, byte) {
	o := sOff(i, cc)
	return h.rd(o), h.rd(o + 1), h.rd(o + 2)
}
func (h *flatHist) setStage(i, cc int, v byte) { h.wr(sOff(i, cc), v) }
func (h *flatHist) setSub(i, cc int, v byte)   { h.wr(sOff(i, cc)+1, v) }
func (h *flatHist) setCause(i, cc int, v byte) { h.wr(sOff(i, cc)+2, v) }
func (h *flatHist) shift() {
	copy(h.buf[0:h.entries*recSize], h.buf[recSize:(h.entries+1)*recSize])
}

type histCell struct{ stage, sub, cause byte }

type uiRec struct {
	ir, start uint32
	st        []histCell
}

type uiHist struct {
	recs    []*uiRec
	entries int
	cap     int
}

func newUIHist(capacity int) *uiHist {
	if capacity < 2 {
		capacity = 2
	}
	h := &uiHist{cap: capacity, entries: 1}
	h.recs = make([]*uiRec, capacity)
	for i := range h.recs {
		h.recs[i] = &uiRec{}
	}
	return h
}

func (h *uiHist) n() int                   { return h.entries }
func (h *uiHist) setN(v int)               { h.entries = v }
func (h *uiHist) capacity() int            { return h.cap }
func (h *uiHist) ir(i int) uint32          { return h.recs[i].ir }
func (h *uiHist) setIR(i int, v uint32)    { h.recs[i].ir = v }
func (h *uiHist) start(i int) uint32       { return h.recs[i].start }
func (h *uiHist) setStart(i int, v uint32) { h.recs[i].start = v }
func (h *uiHist) cell(i, cc int) *histCell {
	r := h.recs[i]
	if cc < 0 || cc > 1<<20 {
		return &histCell{}
	}
	for len(r.st) <= cc {
		r.st = append(r.st, histCell{})
	}
	return &r.st[cc]
}
func (h *uiHist) get(i, cc int) (byte, byte, byte) {
	r := h.recs[i]
	if cc < 0 || cc >= len(r.st) {
		return 0, 0, 0
	}
	c := r.st[cc]
	return c.stage, c.sub, c.cause
}
func (h *uiHist) setStage(i, cc int, v byte) { h.cell(i, cc).stage = v }
func (h *uiHist) setSub(i, cc int, v byte)   { h.cell(i, cc).sub = v }
func (h *uiHist) setCause(i, cc int, v byte) { h.cell(i, cc).cause = v }
func (h *uiHist) shift() {
	r0 := h.recs[0]
	copy(h.recs, h.recs[1:])
	r0.st = r0.st[:0]
	r0.ir, r0.start = 0, 0
	h.recs[len(h.recs)-1] = r0
}

// histClear is the history part of CWinMIPS64Doc::clear().
func histClear(h histStore) {
	h.setN(1)
	h.setIR(0, 0)
	h.setStart(0, 0)
	h.setStage(0, 0, stIFETCH)
}

// updateHistory is CWinMIPS64Doc::update_history (the RESULT rewrite of
// STALLED -> STRUCTURAL is done once by the caller).
func updateHistory(h histStore, pipe *pipeline, cpu *processor, res *result, cycles uint32) {
	for i := 0; i < h.n(); i++ {
		previous := h.ir(i)
		cc := int(cycles - h.start(i))
		stage, substage, _ := h.get(i, cc-1)
		sub := int(substage)

		switch stage {
		case stIFETCH:
			if pipe.if_id.active {
				if pipe.if_id.IR == previous {
					h.setStage(i, cc, stIDECODE)
					h.setCause(i, cc, 0)
				} else {
					h.setStage(i, cc, stIFETCH)
					h.setCause(i, cc, byte(res.IF))
				}
			} else {
				h.setStage(i, cc, 0)
				h.setCause(i, cc, 0)
			}
		case stIDECODE:
			passed := false
			if pipe.integer.active && pipe.integer.IR == previous && res.ID != STALLED {
				passed = true
				h.setStage(i, cc, stINTEX)
				h.setCause(i, cc, 0)
			}
			if pipe.m[0].active && pipe.m[0].IR == previous && res.ID != STALLED {
				passed = true
				h.setStage(i, cc, stMULEX)
				h.setSub(i, cc, 0)
				h.setCause(i, cc, 0)
			}
			if pipe.a[0].active && pipe.a[0].IR == previous && res.ID != STALLED {
				passed = true
				h.setStage(i, cc, stADDEX)
				h.setSub(i, cc, 0)
				h.setCause(i, cc, 0)
			}
			if pipe.div.active && pipe.div.IR == previous && res.ID != STALLED {
				passed = true
				h.setStage(i, cc, stDIVEX)
				h.setCause(i, cc, 0)
			}
			if !passed {
				h.setStage(i, cc, stIDECODE)
				h.setCause(i, cc, byte(res.ID))
			}
		case stINTEX:
			if pipe.ex_mem.active && pipe.ex_mem.IR == previous {
				h.setStage(i, cc, stMEMORY)
				h.setCause(i, cc, 0)
			} else {
				h.setStage(i, cc, stINTEX)
				h.setCause(i, cc, byte(res.EX))
			}
		case stMULEX, stADDEX:
			lat := pipe.MUL_LATENCY
			units := &pipe.m
			rs := res.MULTIPLIER[:]
			if stage == stADDEX {
				lat = pipe.ADD_LATENCY
				units = &pipe.a
				rs = res.ADDER[:]
			}
			if sub == lat-1 {
				if pipe.ex_mem.active && pipe.ex_mem.IR == previous {
					h.setStage(i, cc, stMEMORY)
					h.setCause(i, cc, 0)
				} else {
					h.setStage(i, cc, stage)
					h.setSub(i, cc, byte(sub))
					h.setCause(i, cc, byte(rs[lat-1]))
				}
			} else {
				var nextActive bool
				var nextIR uint32
				if sub+1 < len(units) {
					nextActive, nextIR = units[sub+1].active, units[sub+1].IR
				}
				if nextActive && nextIR == previous {
					h.setStage(i, cc, stage)
					h.setSub(i, cc, byte(sub+1))
					h.setCause(i, cc, 0)
				} else {
					h.setStage(i, cc, stage)
					h.setSub(i, cc, byte(sub))
					c := 0
					if sub < len(rs) {
						c = rs[sub]
					}
					h.setCause(i, cc, byte(c))
				}
			}
		case stDIVEX:
			if pipe.ex_mem.active && pipe.ex_mem.IR == previous {
				h.setStage(i, cc, stMEMORY)
				h.setCause(i, cc, 0)
			} else {
				h.setStage(i, cc, stDIVEX)
				h.setCause(i, cc, byte(res.DIVIDER))
			}
		case stMEMORY:
			if pipe.mem_wb.active && pipe.mem_wb.IR == previous {
				h.setStage(i, cc, stWRITEB)
				h.setCause(i, cc, 0)
			} else {
				h.setStage(i, cc, stMEMORY)
				h.setCause(i, cc, byte(res.MEM))
			}
		default: // WRITEB and anything else
			h.setStage(i, cc, 0)
			h.setCause(i, cc, 0)
		}
	}

	// make a new entry
	n := h.n()
	if (res.ID == OK || res.ID == EMPTY || cpu.PC != h.ir(n-1)) && pipe.active {
		h.setIR(n, cpu.PC)
		h.setStage(n, 0, stIFETCH)
		h.setCause(n, 0, 0)
		h.setStart(n, cycles)
		h.setN(n + 1)
	}
	if h.n() == h.capacity() {
		h.setN(h.n() - 1)
		h.shift()
	}
}

package core

// Constants mirror mytypes.h of the original WinMIPS64 (andoni fork v1.60).

const (
	fALSE = 0
	tRUE  = 1
)

// register access for unavail()
const (
	accREAD  = 0
	accWRITE = 1
	accBOTH  = 2
)

// delimiter() results
const (
	dENDLINE = 1
	dSPACE   = 2
	dCOMMENT = 3
)

const (
	vACANT  = 0
	wRITTEN = 1
)

// cpu.status
const (
	cpuRUNNING = 0
	cpuSTOPPED = 1
)

// history stages
const (
	stIFETCH  = 1
	stIDECODE = 2
	stINTEX   = 3
	stADDEX   = 4
	stMULEX   = 5
	stDIVEX   = 6
	stMEMORY  = 7
	stWRITEB  = 8
)

// Branch status
const (
	nOT_A_BRANCH     = 0
	bRANCH_TAKEN     = 1
	bRANCH_NOT_TAKEN = 2
)

// Stage results / stalls / advisories.
const (
	OK                        = 0
	RAW                       = 1
	WAW                       = 2
	STALLED                   = 3
	HALTED                    = 4
	STRUCTURAL                = 5
	WAR                       = 6
	BRANCH_TAKEN_STALL        = 7
	BRANCH_MISPREDICTED_STALL = 8
	DATA_ERR                  = 9
	EMPTY                     = 10
	DIVIDE_BY_ZERO            = 11
	INTEGER_OVERFLOW          = 12
	NO_SUCH_DATA_MEMORY       = 13
	LOADS                     = 14
	STORES                    = 15
	NO_SUCH_CODE_MEMORY       = 16
	DATA_MISALIGNED           = 17
	WAITING_FOR_INPUT         = 18
)

// register status (source)
const (
	nOT_AVAILABLE = 0
	fROM_REGISTER = 1
	fROM_MEM      = 2
	fROM_EX       = 3
	fROM_ID       = 4
	fROM_ADD      = 5
	fROM_MUL      = 6
	fROM_DIV      = 7
)

// assembler instruction formats
const (
	r_TYPE = 1
	i_TYPE = 2
	j_TYPE = 3
	f_TYPE = 4
	m_TYPE = 5
	b_TYPE = 6
)

// instruction (sub)types
const (
	tNOP    = 0
	tLOAD   = 1
	tSTORE  = 2
	tREG1I  = 3
	tREG2I  = 4
	tREG2S  = 5
	tJUMP   = 6
	tJREG   = 7
	tHALT   = 8
	tREG3F  = 9
	tBRANCH = 10
	tREG3   = 11
	tREGID  = 12
	tFLOAD  = 13
	tFSTORE = 14
	tJREGN  = 15
	tREG2F  = 16
	tREG3X  = 17
	tREGDI  = 18
	tREG2C  = 19
	tBC     = 20
)

const (
	i_SPECIAL = 0x00
	i_COP1    = 0x11
	i_DOUBLE  = 0x11
	i_MTC1    = 0x04
	i_MFC1    = 0x00
	i_BC      = 0x08
	i_HALT    = 0x01

	i_J    = 0x02
	i_JAL  = 0x03
	i_BEQ  = 0x04
	i_BNE  = 0x05
	i_BEQZ = 0x06
	i_BNEZ = 0x07

	i_DADDI  = 0x18
	i_DADDIU = 0x19
	i_SLTI   = 0x0A
	i_SLTIU  = 0x0B
	i_ANDI   = 0x0C
	i_ORI    = 0x0D
	i_XORI   = 0x0E
	i_LUI    = 0x0F

	i_LB  = 0x20
	i_LH  = 0x21
	i_LW  = 0x23
	i_LBU = 0x24
	i_LHU = 0x25
	i_LWU = 0x27
	i_SB  = 0x28
	i_SH  = 0x29
	i_SW  = 0x2B
	i_L_D = 0x35
	i_S_D = 0x3D
	i_LD  = 0x37
	i_SD  = 0x3F

	r_NOP  = 0x00
	r_JR   = 0x08
	r_JALR = 0x09
	r_MOVZ = 0x0A
	r_MOVN = 0x0B

	r_DSLLV = 0x14
	r_DSRLV = 0x16
	r_DSRAV = 0x17
	r_DMUL  = 0x1C
	r_DMULU = 0x1D
	r_DDIV  = 0x1E
	r_DDIVU = 0x1F

	r_AND   = 0x24
	r_OR    = 0x25
	r_XOR   = 0x26
	r_SLT   = 0x2A
	r_SLTU  = 0x2B
	r_DADD  = 0x2C
	r_DADDU = 0x2D
	r_DSUB  = 0x2E
	r_DSUBU = 0x2F

	r_DSLL = 0x38
	r_DSRL = 0x3A
	r_DSRA = 0x3B

	f_ADD_D   = 0x00
	f_SUB_D   = 0x01
	f_MUL_D   = 0x02
	f_DIV_D   = 0x03
	f_MOV_D   = 0x06
	f_CVT_D_L = 0x21
	f_CVT_L_D = 0x25
	f_C_LT_D  = 0x3C
	f_C_LE_D  = 0x3E
	f_C_EQ_D  = 0x32
)

func sI(x uint32) uint32 { return x << 26 }
func sR(x uint32) uint32 { return x | i_SPECIAL<<26 }
func sF(x uint32) uint32 { return x | i_COP1<<26 | i_DOUBLE<<21 }

const (
	sMTC1 = i_COP1<<26 | i_MTC1<<21
	sMFC1 = i_COP1<<26 | i_MFC1<<21
	sBC1F = i_COP1<<26 | i_BC<<21
	sBC1T = i_COP1<<26 | i_BC<<21 | 1<<16
)

// CODEORDATA
const (
	segCODE = 1
	segDATA = 2
)

const (
	sTEP = 8
	mMIO = 0x10000
	gSXY = 50

	// WHITE = RGB(255,255,255) as a Windows COLORREF (0x00BBGGRR)
	colWHITE uint32 = 0x00FFFFFF
)

const (
	minCodeBits   = 8
	maxCodeBits   = 13
	minDataBits   = 4
	maxDataBits   = 11
	minAddLatency = 2
	maxAddLatency = 8
	minMulLatency = 2
	maxMulLatency = 8
	minDivLatency = 10
	maxDivLatency = 30
	symTabSize    = 1000
	maxLine       = 200
)

// result mirrors the RESULT struct.
type result struct {
	IF, ID, EX, MEM, WB, DIVIDER int
	ADDER                        [10]int
	MULTIPLIER                   [10]int
	idrr, exrr, memrr            int
	addrr, mulrr, divrr          int
}

type instruction struct {
	typ, function, opcode, tf, target int
	rs, rt, rd                        int
	src1, src2                        int
	Imm                               int32
}

type reg struct {
	val    uint64
	source int32
}

type processor struct {
	status   int
	codesize uint32
	datasize uint32
	code     []byte
	cstat    []byte
	data     []byte
	dstat    []byte
	mm       [16]byte
	screen   []uint32
	nlines   uint32
	ncols    uint32
	drawit   bool
	Terminal []byte
	keyboard uint32
	PC       uint32
	rreg     [64]reg
	wreg     [64]reg
	fp_cc    bool
}

type ifIDReg struct {
	IR        uint32
	ins       instruction
	NPC       uint32
	active    bool
	predicted bool
}

type idEXReg struct {
	IR        uint32
	ins       instruction
	rA, rB    int
	NPC       uint32
	ALUOutput uint64
	Imm       int32
	active    bool
	cycles    int
}

type exMEMReg struct {
	IR        uint32
	ins       instruction
	rB        int
	ALUOutput uint64
	NPC       uint32
	active    bool
	condition bool
}

type memWBReg struct {
	IR             uint32
	ins            instruction
	ALUOutput, LMD uint64
	NPC            uint32
	active         bool
	condition      bool
}

type pipeline struct {
	active      bool
	halting     bool
	branch      bool
	destination uint32
	ADD_LATENCY int
	MUL_LATENCY int
	DIV_LATENCY int

	if_id   ifIDReg
	integer idEXReg
	m       [10]idEXReg
	a       [10]idEXReg
	div     idEXReg
	ex_mem  exMEMReg
	mem_wb  memWBReg
}

type symbol struct {
	symb  []byte
	value uint32
}

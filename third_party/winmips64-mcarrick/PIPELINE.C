/*
   MIPS 64 Pipeline Simulator, a la Hennesy & Patterson

   Stages are executed in reverse order - WB/MEM/EX/ID/IF
   In effect instructions are "dragged" through the pipeline 
   from the end. 
  
   However this causes problems that need to be addressed, as each stage should 
   really be executed simultaneously, taking care to do appropraite things 
   on both the Leading and Trailing clock edges.

   Forwarding is handled by maintaining two sets of registers
   the Read Registers and the Write Registers. Register status
   is also recorded, indicating where the latest register value is available,
   from, i.e. from the register itself, or forwarded from one of the various 
   pipeline registers. If a register is not available - from anywhere - we 
   get a stall.
 
   At each pipeline stage Register values are read from the Read register, 
   (indicating its status after the last clock tick) and updated to the Write 
   register. However..
  
   1. As WB takes place on the leading edge, it updates both Read and Write 
      Registers
   2. A forwarded value must be nullified once a newer value becomes 
      available, so if EX is writing to a register, both its Read and Write
      Status are updated so that earlier stages are immediately notified.
  
   At the end of a clock-tick the write registers are copied into the Read 
   registers.
  
   Note that only the most recently available value for a register is 
   stored. So a register display should only be updated is a register
   has the availability status FROM_REGISTER.

   Copyright (c) Mike Scott 2003
*/

#include <stdio.h>
#include <math.h>
#include <string.h>
#include <stdlib.h>

#define I_SPECIAL       0x00
#define I_COP1          0x11
#define I_DOUBLE        0x11
#define I_MTC1          0x04
#define I_HALT          0x01

#define I_J             0x02
#define I_JAL           0x03
#define I_BEQ           0x04
#define I_BNE           0x05
#define I_BEQZ          0x06
#define I_BNEZ          0x07

#define I_DADDI         0x18
#define I_DADDIU        0x19
#define I_SLTI          0x0A
#define I_SLTIU         0x0B
#define I_ANDI          0x0C
#define I_ORI           0x0D
#define I_XORI          0x0E
#define I_LUI           0x0F

#define I_LB            0x20
#define I_LH            0x21
#define I_LW            0x23
#define I_LBU           0x24
#define I_LHU           0x25
#define I_LWU           0x27
#define I_SB            0x28
#define I_SH            0x29
#define I_SW            0x2B
#define I_L_D           0x35
#define I_S_D           0x3D
#define I_LD            0x37
#define I_SD            0x3F

#define R_NOP           0x00
#define R_JR            0x08
#define R_JALR          0x09
#define R_MOVZ          0x0A
#define R_MOVN          0x0B

#define R_DSLLV         0x14
#define R_DSRLV         0x16
#define R_DSRAV         0x17
#define R_DMUL          0x1C
#define R_DMULU         0x1D
#define R_DDIV          0x1E
#define R_DDIVU         0x1F

#define R_AND           0x24
#define R_OR            0x25
#define R_XOR           0x26
#define R_SLT           0x2A
#define R_SLTU          0x2B
#define R_DADD          0x2C
#define R_DADDU         0x2D
#define R_DSUB          0x2E
#define R_DSUBU         0x2F

#define R_DSLL          0x38
#define R_DSRL          0x3A
#define R_DSRA          0x3B

#define F_ADD_D         0x00
#define F_SUB_D         0x01
#define F_MUL_D         0x02
#define F_DIV_D         0x03
#define F_MOV_D         0x06
#define F_CVT_W_D       0x24
#define F_CVT_L_D       0x25

#define FALSE 0
#define TRUE  1
typedef unsigned char BYTE;
typedef unsigned __int64 WORD64;
typedef __int64 SIGNED64;
typedef unsigned long WORD32;
typedef int SIGNED32;
typedef unsigned short WORD16;
typedef int BOOL;

/* pipeline stage status */

#define OK       0
#define RAW      1
#define WAW      2
#define STALLED  3
#define HALTED   4
#define DATA_ERR 5
#define BREAK    6
#define EMPTY    7
#define DIVIDE   8
#define MULTIPLY 9
#define ADDITION 10
#define BRANCH_TAKEN 11
#define BRANCH_NOT_TAKEN 12
#define LOADS    13
#define STORES   14
#define STRUCTURAL 15

/* register status */

#define NOT_AVAILABLE  0
#define FROM_REGISTER  1
#define FROM_MEM       2
#define FROM_EX        3
#define FROM_ID        4

#define RUNNING 0
#define STOPPED 1

/* significant instruction types */

#define R_TYPE 1
#define I_TYPE 2
#define J_TYPE 3
#define F_TYPE 4
#define M_TYPE 5

/* I_TYPEs and R_TYPEs may be branches....??
   Jumps and BEQZ type branches 
*/

#define NOP   0
#define LOAD  1     /* I_TYPE           */
#define STORE 2     /* I_TYPE           */
#define REG1I 3     /* I_TYPE           */
#define REG2I 4     /* I_TYPE           */
#define REG2S 5     /* R_TYPE           */
#define JUMP  6     /* J_TYPE - Branch  */
#define JREG  7     /* R_TYPE - Branch  */
#define HALT  8     /* R_TYPE           */
#define REG3F 9     /* F_TYPE - Float   */
#define BRANCH 10   /* I_TYPE - Branch  */
#define REG3   11   /* R_TYPE           */
#define REGID  12   /* M_TYPE - Float   */
#define FLOAD  13   /* I_TYPE - Float   */
#define FSTORE 14   /* I_TYPE - Float   */
#define JREGN  15   /* I_TYPE - Branch  */
#define REG2F  16   /* F_TYPE - Float   */

/* Doubles are 64-bits. They are stored as 64-bit unsigned integers */

union 
{
    WORD64  u;
    SIGNED64 s; 
    double  d;
} typedef DOUBLE64;

struct
{
    int type,function,opcode;
    int rs,rt,rd;
    SIGNED32 Imm;
} typedef instruction;

/* A register has a value, and a source
   Source = 0 value not available
   Source = 1 value available from Register file
   Source = 2 value available from MEM forwarding registers
   Source = 3 value available from EX forwarding registers
   Source = 4 value available from ID forwarding registes
*/

struct
{
    WORD64 val;
    WORD32 source;
} typedef reg;

#define VACANT  0
#define WRITTEN 1

struct
{
    int    status;
    BYTE   *code;
    BYTE   *cstat;
    BYTE   *data;
    BYTE   *dstat;
    WORD32 PC;
    reg    rreg[64];
    reg    wreg[64];
} typedef processor;

struct 
{
    WORD32 IR;  /* pointer to instruction in memory */
    instruction ins;
    WORD32 NPC;
    BOOL active;
} typedef if_id_reg;

struct 
{
    WORD32 IR;
    instruction ins;
    int rA,rB;
    WORD32 NPC;
    SIGNED32 Imm;
    BOOL active;
    int cycles;
} typedef id_ex_reg;

struct 
{
    WORD32 IR;
    instruction ins;
    int rB;
    WORD64 ALUOutput;
    WORD32 NPC;
    BOOL active;
} typedef ex_mem_reg;

struct
{
    WORD32 IR;
    instruction ins;
    WORD64 ALUOutput,LMD;
    WORD32 NPC;
    BOOL active;
} typedef mem_wb_reg;

struct 
{
    BOOL active;
    BOOL   branch;
    WORD32 destination;
    int ADD_LATENCY,MUL_LATENCY,DIV_LATENCY;

    if_id_reg  if_id;
    id_ex_reg  id_ex;
    id_ex_reg m[10];
    id_ex_reg a[10];
    id_ex_reg div;
    ex_mem_reg ex_mem;
    mem_wb_reg  mem_wb;
} typedef pipeline;

void init_processor(processor *cpu)
{
    int i;
    for (i=0;i<64;i++)
    {
        cpu->rreg[i].val=cpu->wreg[i].val=0;
        cpu->rreg[i].source=cpu->wreg[i].source=FROM_REGISTER;
    }
    cpu->PC=0;
    cpu->code=NULL;
    cpu->data=NULL;
    cpu->status=RUNNING;
}

void init_pipeline(pipeline *pipe,int ADDS,int MULS,int DIVS)
{
    int i;
    pipe->branch=FALSE;
    pipe->destination=0;
    pipe->ADD_LATENCY=ADDS;
    pipe->MUL_LATENCY=MULS;
    pipe->DIV_LATENCY=DIVS;


    pipe->if_id.active=FALSE;
    pipe->id_ex.active=FALSE;
    pipe->ex_mem.active=FALSE;
    pipe->mem_wb.active=FALSE;
    for (i=0;i<ADDS;i++) pipe->a[i].active=FALSE;
    for (i=0;i<MULS;i++) pipe->m[i].active=FALSE;
    pipe->div.active=FALSE;
    pipe->div.cycles=0;
    pipe->active=TRUE;
}

/* get instruction type */

int get_type(WORD32 instruct)
{
    int type,fmt,function,opcode,op3;
    type=0;
    opcode=(instruct >> 26);
    if (opcode==I_SPECIAL) 
    {
        function=(instruct&0x3F);
        type=REG3;
        if (function==R_DSRL || function==R_DSRA || function==R_DSLL)
            type=REG2S;
        if (function==R_JR || function==R_JALR)
            type=JREG;
        if (function==R_NOP) type=NOP;
        return type;
    }

    if (opcode==I_COP1)
    {
        fmt=(instruct>>21)&0x1F;
        function=(instruct&0x3F);
        if (fmt==I_DOUBLE)
        {
            type=REG3F;
            if (function==F_CVT_W_D || function==F_CVT_L_D) type=REG2F;
        }
        if (fmt==I_MTC1)   type=REGID;
        return type;
    }

    type=REG2I;
    if (opcode==I_J || opcode==I_JAL) type=JUMP;
    if (opcode==I_LUI) type=REG1I;
    op3=opcode>>3;
    if (op3==0x4 || opcode==I_LD) type=LOAD;
    if (op3==0x5 || opcode==I_SD) type=STORE;
    if (opcode==I_L_D) type=FLOAD;
    if (opcode==I_S_D) type=FSTORE;
    if (opcode==I_BEQ || opcode==I_BNE) type=BRANCH;
    if (opcode==I_BEQZ || opcode==I_BNEZ) type=JREGN;
    if (opcode==I_HALT) type=HALT;

    return type;
}

int parse(WORD32 instruct,instruction *ins)
{
    SIGNED32 extend;
    int type;

    type=get_type(instruct);

    ins->opcode=(instruct >> 26);
    ins->type=type;
    ins->function=(instruct&0x3F);
    ins->rt=(instruct >> 16)&0x1F;

    ins->rs=(instruct >> 21)&0x1F;
    ins->rd=(instruct >> 11)&0x1F;

    if (type==FSTORE || type==FLOAD || type==REG3F || type==REG2F || type==REGID) 
        ins->rt+=32;

    if (type==REG3F)
    {
        ins->rs=32+((instruct >> 11)&0x1F);
        ins->rd=32+((instruct >> 6)&0x1F);
    }

    if (type==REG2F)
        ins->rs=32+((instruct >> 11)&0x1F);
    if (type==REGID)
        ins->rs=(instruct >> 11)&0x1F;

    extend=(instruct&0xFFFF);                /* sign extended */
    ins->Imm=((extend<<16)>>16);
    if (type==JUMP)
    {
        extend=(instruct&0x3FFFFFF);
        ins->Imm=((extend<<6)>>6);
    }
    if (type==REG2S)
        ins->Imm=(instruct >> 6 )&0x1F;

    return type;
}

BOOL available(processor *cpu,int r)
{
    if (r==0) return TRUE;
    if (cpu->rreg[r].source==NOT_AVAILABLE) return FALSE;
    return TRUE;
}

/*
   check for WAW condition
   Check if the destination register r is going to be written 
   by one of the FP units.... it may be necessary to stall
*/

BOOL waw(pipeline *pipe,int type,int function,int r)
{
    int i;
    switch (type)
    {
    case REGID:
    case FLOAD:
    case FSTORE:
    case REG2F:
        if (pipe->div.active && pipe->div.ins.rt==r) return TRUE;
        for (i=0;i<pipe->MUL_LATENCY;i++) 
            if (pipe->m[i].active && pipe->m[i].ins.rt==r) return TRUE;
        for (i=0;i<pipe->ADD_LATENCY;i++) 
            if (pipe->a[i].active && pipe->a[i].ins.rt==r) return TRUE;
        break;
    case REG3F:
        switch (function)
        {
        case F_DIV_D:
            break;        /* a divide isn't going to pass out anything... */
        case F_MUL_D:
            if (pipe->div.active && pipe->div.ins.rd==r) return TRUE;
            for (i=0;i<pipe->ADD_LATENCY;i++) 
                if (pipe->a[i].active && pipe->a[i].ins.rd==r) return TRUE;
            break;
        case F_ADD_D:
        case F_SUB_D:
            if (pipe->div.active && pipe->div.ins.rd==r) return TRUE;
            for (i=0;i<pipe->MUL_LATENCY;i++) 
                if (pipe->m[i].active && pipe->m[i].ins.rd==r) return TRUE;
            break;
        default: 
            break;
        }
        break;
    default:
        break;
    }

    return FALSE;
}

int IF(pipeline *pipe,processor *cpu,BOOL delay_slot)
{
    int type;
    instruction ins;
    WORD32 codeword;

/* instruction fetch */

    if (pipe->if_id.active) return STALLED;

/*
    if (cpu->cstat!=NULL)
        if (cpu->cstat[cpu->PC]) 
        {   
            cpu->cstat[cpu->PC]=0; 
            return BREAK;         
        }
*/
    if (pipe->active)
    {
        pipe->if_id.IR=cpu->PC;
        codeword=*(WORD32 *)&cpu->code[cpu->PC];
        type=parse(codeword,&ins);
    }
    else
    { /* if pipeline is inactive, deactivate the pipe */
        pipe->if_id.active=FALSE;
        return OK;
    }
    
    pipe->if_id.ins=ins;


/* check for branches */ 

    pipe->if_id.NPC=cpu->PC+4;
    if (pipe->branch) pipe->if_id.NPC=pipe->destination;

    cpu->PC=pipe->if_id.NPC;

/* Instruction just fetched may need to be nullified */
    pipe->if_id.active=TRUE;
    if (!delay_slot && pipe->branch) 
    {
        pipe->if_id.active=FALSE;
        pipe->if_id.ins.type=NOP;
    }
    if (pipe->if_id.ins.type==HALT) pipe->active=FALSE;

    pipe->branch=FALSE;
    return OK;
}

int ID(pipeline *pipe,processor *cpu,BOOL forwarding,int *rawreg)
{
    int status;
    int rA,rB,type;
    instruction ins;
    WORD32 A,B;
    BOOL branch_complete;

    if (!pipe->if_id.active) return EMPTY;      /* nothing to do   */
    if (pipe->id_ex.active)  return STALLED; /* exit is blocked */

    ins=pipe->if_id.ins;
    type=ins.type;

/* get A and B registers as required */
    status=OK;
    switch (type)
    {
    case LOAD:
    case FLOAD:
    case REG2S:
    case REGID:
    case REG2F:
        rA=pipe->id_ex.rA=ins.rs;
        break;
    case STORE:
    case REG2I:
    case REG3:
    case FSTORE:
    case REG3F:
        rA=pipe->id_ex.rA=ins.rs;
        rB=pipe->id_ex.rB=ins.rt;
        break;
    case JREG:
    case JREGN:
        if (!available(cpu,ins.rt))
        {
            *rawreg=ins.rt;
            return RAW;
        }
        pipe->id_ex.rB=ins.rt;
        B=(WORD32)cpu->rreg[ins.rt].val;
        break;
    case BRANCH:
        if (!available(cpu,ins.rs)) {*rawreg=ins.rs; return RAW;}
        if (!available(cpu,ins.rt)) {*rawreg=ins.rt; return RAW;}
        pipe->id_ex.rA=ins.rs;
        pipe->id_ex.rB=ins.rt;
        A=(WORD32)cpu->rreg[ins.rs].val;
        B=(WORD32)cpu->rreg[ins.rt].val;
        break;
    default: 
        break;
    }

 /* jumps and branches */
    branch_complete=FALSE; /* If branch or jump completes here - kill it off */
    switch (type)
    {
    case BRANCH:
        branch_complete=TRUE;
        status=BRANCH_NOT_TAKEN;
        if (ins.opcode==I_BEQ && (A==B))
        {

            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
            status=BRANCH_TAKEN;
        }
        if (ins.opcode==I_BNE && (A!=B))
        { 
            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
            status=BRANCH_TAKEN;
        }
        break;
    case JREGN:
        branch_complete=TRUE;
        status=BRANCH_NOT_TAKEN;
        if (B==0 && ins.opcode==I_BEQZ) 
        { 
            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
            status=BRANCH_TAKEN;
        }
        if (B!=0 && ins.opcode==I_BNEZ) 
        {     
            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
            status=BRANCH_TAKEN;
        }
        break;
    case JUMP:
        status=BRANCH_TAKEN;
        if (ins.opcode==I_J)
        {
            branch_complete=TRUE;
            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
        }
        if (ins.opcode==I_JAL)
        {
            pipe->branch=TRUE;
            pipe->destination=pipe->if_id.NPC+4*pipe->if_id.ins.Imm;
            if (forwarding)
            {
                cpu->wreg[31].val=pipe->if_id.NPC+4;
                cpu->wreg[31].source=FROM_ID;
            }
            else
                cpu->wreg[31].source=NOT_AVAILABLE;

        }
        break;
    case JREG:
        status=BRANCH_TAKEN;
        if (ins.opcode==I_SPECIAL && ins.function==R_JR)
        {
            branch_complete=TRUE;
            pipe->branch=TRUE;
            pipe->destination=B;
        }
        break;
        if (ins.opcode==I_SPECIAL && ins.function==R_JALR)
        {
            pipe->branch=TRUE;
            pipe->destination=B;
            if (forwarding)
            {
                cpu->wreg[31].val=pipe->if_id.NPC+4;
                cpu->wreg[31].source=FROM_ID;
            }
            else
                cpu->wreg[31].source=NOT_AVAILABLE;
        }
        break;
    default:
        break;
    }

    pipe->id_ex.NPC=pipe->if_id.NPC;
    pipe->id_ex.IR =pipe->if_id.IR;
    pipe->id_ex.ins=ins;
    pipe->id_ex.Imm=ins.Imm;
    if (branch_complete) pipe->id_ex.active=FALSE;
    else                 pipe->id_ex.active=TRUE;
    pipe->if_id.active=FALSE;
    return status;
}

int EX(pipeline *pipe,processor *cpu,BOOL forwarding,int *rawreg)
{
    int ADDS,MULS,DIVS,rA,rB,opcode,status,type,i,function;
    instruction ins;     
    BOOL mul_stall,add_stall,mem_busy;
    DOUBLE64 fpA,fpB,fpR;
    id_ex_reg idle;

    mem_busy=pipe->ex_mem.active;
    idle.active=FALSE;
    
/* deal with output side first
   decide which instruction, if any, is to proceed to MEM

   if the division unit is active, count it down by one
   if cycles = 0 and active is TRUE, then result is available
*/

    ADDS=pipe->ADD_LATENCY;
    MULS=pipe->MUL_LATENCY;
    DIVS=pipe->DIV_LATENCY;

    if (pipe->div.active)
    { /* Have I got a result ?? */
        if (pipe->div.cycles>0) pipe->div.cycles--;
        if (pipe->div.cycles==0 && !pipe->ex_mem.active)
        { /*  execute divide  */
            ins=pipe->div.ins;
            fpA.u=cpu->rreg[pipe->div.rA].val;
            fpB.u=cpu->rreg[pipe->div.rB].val;
            fpR.d=fpA.d/fpB.d;

            pipe->ex_mem.ins=ins;
            pipe->ex_mem.IR=pipe->div.IR;

            pipe->ex_mem.ALUOutput=fpR.u;
            if (forwarding)
            {  
                cpu->wreg[ins.rd].val=fpR.u;
                cpu->wreg[ins.rd].source=FROM_EX;
            }
            pipe->ex_mem.active=TRUE;
            pipe->div.active=FALSE;
        }
    }

    mul_stall=TRUE;
    if (pipe->m[MULS-1].active)
    {
        if (!pipe->ex_mem.active)
        { /* execute multiply */
            ins=pipe->m[MULS-1].ins;     

            fpA.u=cpu->rreg[pipe->m[MULS-1].rA].val;
            fpB.u=cpu->rreg[pipe->m[MULS-1].rB].val;
            fpR.d=fpA.d*fpB.d;

            pipe->ex_mem.ins=ins;
            pipe->ex_mem.IR=pipe->m[MULS-1].IR;

            pipe->ex_mem.ALUOutput=fpR.u;
            if (forwarding)
            { 
                cpu->wreg[ins.rd].val=fpR.u;
                cpu->wreg[ins.rd].source=FROM_EX;
            }

            pipe->ex_mem.active=TRUE;
            mul_stall=FALSE;
        }
    }
    else mul_stall=FALSE;

    add_stall=TRUE;
    if (pipe->a[ADDS-1].active)
    {
        if (!pipe->ex_mem.active)
        {
            function=pipe->a[ADDS-1].ins.function;
            ins=pipe->a[ADDS-1].ins;     

            fpA.u=cpu->rreg[pipe->a[ADDS-1].rA].val;
            fpB.u=cpu->rreg[pipe->a[ADDS-1].rB].val;

            switch (function)
            {
            case F_ADD_D:
                fpR.d=fpA.d+fpB.d;
                break;
            case F_SUB_D:
                fpR.d=fpA.d-fpB.d;
                break;
            } 
            pipe->ex_mem.ins=ins;
            pipe->ex_mem.IR=pipe->a[ADDS-1].IR;

            pipe->ex_mem.ALUOutput=fpR.u;
            if (forwarding)
            {
                cpu->wreg[ins.rd].val=fpR.u;
                cpu->wreg[ins.rd].source=FROM_EX;
            }

            pipe->ex_mem.active=TRUE;
            add_stall=FALSE;
        }
    }
    else add_stall=FALSE;

    if (!mul_stall)
    { /* shift forward by one and put NOP in the start */
        for (i=MULS-1;i>0;i--)
            pipe->m[i]=pipe->m[i-1];
        pipe->m[0]=idle;
    }

    if (!add_stall)
    { /* shift forward by one and put NOP in the start */
        for (i=ADDS-1;i>0;i--)
            pipe->a[i]=pipe->a[i-1];
        pipe->a[0]=idle;
    }

/* Now deal with input side */

    if (!pipe->id_ex.active) 
    {
        pipe->ex_mem.active=FALSE;
        return EMPTY;
    }

    ins=pipe->id_ex.ins;
    type=ins.type;
    function=ins.function;
    opcode=ins.opcode;

/* check for structural hazard */

    if (type==REG3F)
    {
        switch (ins.function)
        {
            case F_ADD_D:
            case F_SUB_D:
                if (pipe->a[0].active) return STRUCTURAL;
                break;
            case F_MUL_D:
                if (pipe->m[0].active) return STRUCTURAL;
                break;
            case F_DIV_D:
                if (pipe->div.active) return STRUCTURAL;
                break;
            default:
                break;
        }
    }
    else
    {
        if (pipe->ex_mem.active)
        { 
            if (mem_busy) return STALLED;
            else          return STRUCTURAL;
        }
    }

/* All systems go. Get instruction, and pass to appropriate unit
   Mark destination register as not available for earlier stages...
*/

    if (mul_stall || add_stall) status=STRUCTURAL;
    else status=OK;

    rA=pipe->id_ex.rA;
    rB=pipe->id_ex.rB;

    switch (type)
    {
    case FLOAD:
        if (waw(pipe,type,function,ins.rt)) {*rawreg=ins.rt; return WAW;}
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        cpu->rreg[ins.rt].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        pipe->ex_mem.ALUOutput=fpA.u + ins.Imm;
        break;
    case LOAD:
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        cpu->rreg[ins.rt].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        pipe->ex_mem.ALUOutput=fpA.u + ins.Imm;
        break;
    case FSTORE:
        if (waw(pipe,type,function,ins.rt)) {*rawreg=ins.rt; return RAW;}  
                                          /* bodge */
    case STORE:
   /* calculate address for memory access - same for all flavours */
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        fpA.u=cpu->rreg[rA].val;
        pipe->ex_mem.ALUOutput=fpA.u + ins.Imm;
        break;
    case REG1I:
        cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=NOT_AVAILABLE;
        switch (opcode)
        {
        case I_LUI:
            pipe->ex_mem.ALUOutput=((SIGNED64)ins.Imm<<32);
        default:
            break;
        }
        if (forwarding && ins.rt!=0) 
        {
            cpu->wreg[ins.rt].val=pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rt].source=FROM_EX;
        }
        break;
    case REG2I:
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        switch (opcode)
        {
        case I_DADDI:
            pipe->ex_mem.ALUOutput = fpA.s + ins.Imm;
            break;
        case I_DADDIU:
            pipe->ex_mem.ALUOutput = fpA.u + ins.Imm;
            break;
        case I_ANDI:
            pipe->ex_mem.ALUOutput = fpA.u & ins.Imm;
            break;
        case I_ORI:
            pipe->ex_mem.ALUOutput = fpA.u | ins.Imm;
            break;
        case I_XORI:
            pipe->ex_mem.ALUOutput = fpA.u ^ ins.Imm;
            break;
        case I_SLTI:
            if (fpA.s < ins.Imm) pipe->ex_mem.ALUOutput=1;
            else                 pipe->ex_mem.ALUOutput=0;
            break;
        case I_SLTIU:
            if (fpA.u < ins.Imm) pipe->ex_mem.ALUOutput=1;
            else                 pipe->ex_mem.ALUOutput=0;
            break;
        }
        if (forwarding && ins.rt!=0) 
        {
            cpu->wreg[ins.rt].val = pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rt].source = FROM_EX;
        }
        break;
    case REG2S:
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        cpu->rreg[ins.rd].source=cpu->wreg[ins.rd].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        switch (function)
        {
        case R_DSLL:
            pipe->ex_mem.ALUOutput=fpA.u << ins.Imm;    
            break;
        case R_DSRL:
            pipe->ex_mem.ALUOutput=fpA.u >> ins.Imm;    
            break;
        case R_DSRA:
            pipe->ex_mem.ALUOutput=fpA.s >> ins.Imm;    
            break;
        default: 
            break; 
        }
        if (forwarding && ins.rd!=0)
        {
            cpu->wreg[ins.rd].val = pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rd].source = FROM_EX;
        }
        break;
    case JUMP:
    case JREG:
        if (forwarding && opcode==I_JAL || (opcode==I_SPECIAL && function==R_JALR))
            cpu->wreg[31].source = FROM_EX;
        break;
    case HALT:
        break;
    case REG3F:
        if (waw(pipe,type,function,ins.rd)) {*rawreg=ins.rd; return WAW;}
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        if (!available(cpu,rB)) {*rawreg=rB; return RAW;}
        cpu->rreg[ins.rd].source=cpu->wreg[ins.rd].source=NOT_AVAILABLE;
        switch (function)
        {
        case F_DIV_D:
            pipe->div=pipe->id_ex;
            pipe->div.cycles=DIVS;
            pipe->id_ex.active=FALSE;
            return status;
        case F_MUL_D:
            pipe->m[0]=pipe->id_ex;
            pipe->id_ex.active=FALSE;
            return status;  
        case F_ADD_D:
        case F_SUB_D:
            pipe->a[0]=pipe->id_ex;
            pipe->id_ex.active=FALSE;
            return status;
        default:
            break;
        }
        break;

    case REG3:
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        if (!available(cpu,rB)) {*rawreg=rB; return RAW;}
        cpu->rreg[ins.rd].source=cpu->wreg[ins.rd].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        fpB.u=cpu->rreg[rB].val;
        switch (function)
        {
        case R_AND:
            pipe->ex_mem.ALUOutput=fpA.u & fpB.u;
            break;
        case R_OR:
            pipe->ex_mem.ALUOutput=fpA.u | fpB.u;
            break;
        case R_XOR:
            pipe->ex_mem.ALUOutput=fpA.u ^ fpB.u;
            break;
        case R_SLT:
            if (fpA.s < fpB.s) pipe->ex_mem.ALUOutput=1;
            else               pipe->ex_mem.ALUOutput=0;
            break;
        case R_SLTU:
            if (fpA.u < fpB.u) pipe->ex_mem.ALUOutput=1;
            else               pipe->ex_mem.ALUOutput=0;
            break;
        case R_DADD:
            pipe->ex_mem.ALUOutput=fpA.s + fpB.s;
            break;
        case R_DADDU:
            pipe->ex_mem.ALUOutput=fpA.u + fpB.u;
            break;
        case R_DSUB:
            pipe->ex_mem.ALUOutput=fpA.s - fpB.s;
            break;
        case R_DSUBU:
            pipe->ex_mem.ALUOutput=fpA.u - fpB.u;
            break;
        case R_DMUL:
            pipe->ex_mem.ALUOutput=fpA.s * fpB.s;
            break;
        case R_DMULU:
            pipe->ex_mem.ALUOutput=fpA.u * fpB.u;
            break;
        case R_DDIV:
            pipe->ex_mem.ALUOutput=fpA.s / fpB.s;
            break;
        case R_DDIVU:
            pipe->ex_mem.ALUOutput=fpA.u / fpB.u;
            break;
        case R_DSLLV:
            pipe->ex_mem.ALUOutput=fpA.u << (fpB.u & 0x3F);
            break;
        case R_DSRLV:
            pipe->ex_mem.ALUOutput=fpA.u >> (fpB.u & 0x3F);
            break;
        case R_DSRAV:
            pipe->ex_mem.ALUOutput=fpA.s >> (fpB.u & 0x3F);
            break;
        case R_MOVZ:
            if (fpB.u==0) pipe->ex_mem.ALUOutput=fpA.u;
            break;
        case R_MOVN:
            if (fpB.u!=0) pipe->ex_mem.ALUOutput=fpA.u;
            break;
        default:
            break;
        }
        if (forwarding && ins.rd!=0)
        {
            cpu->wreg[ins.rd].val = pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rd].source = FROM_EX;
        }
        break;
    case REGID:
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        if (waw(pipe,type,function,ins.rt)) {*rawreg=ins.rt; return WAW;}
        cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=NOT_AVAILABLE;
        fpA.u=cpu->rreg[rA].val;
        pipe->ex_mem.ALUOutput=fpA.s;
        if (forwarding && ins.rt!=0)
        {
            cpu->wreg[ins.rt].val = pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rt].source = FROM_EX;
        }
        break;
    case JREGN:
        break;
    case REG2F:
        if (waw(pipe,type,function,ins.rt)) {*rawreg=ins.rt; return WAW;}
        if (!available(cpu,rA)) {*rawreg=rA; return RAW;}
        cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=NOT_AVAILABLE;

        fpA.u=cpu->rreg[rA].val;
        switch (function)
        {
        case F_CVT_W_D:
        case F_CVT_L_D:
            fpR.d = (double)fpA.s;
            pipe->ex_mem.ALUOutput=fpR.u;
            break;
        default:
            break;
        }
        if (forwarding && ins.rt!=0)
        {
            cpu->wreg[ins.rt].val = pipe->ex_mem.ALUOutput;
            cpu->wreg[ins.rt].source = FROM_EX;
        }
        break;

    case NOP:
    default:
        break;
    }
    pipe->ex_mem.IR=pipe->id_ex.IR;
    pipe->ex_mem.ins=ins;
    pipe->ex_mem.NPC=pipe->id_ex.NPC;
    pipe->ex_mem.rB=pipe->id_ex.rB;
    pipe->ex_mem.active=TRUE;
    pipe->id_ex.active=FALSE;
    
    return status;
}

int MEM(pipeline *pipe,processor *cpu,BOOL forwarding,int *rawreg)
{
    int i,ptr,opcode,function,status,type,rB;
    instruction ins; 
    BYTE   b;
    WORD16 h;
    WORD32 w;
    WORD64  u;
    SIGNED64 s;

    pipe->mem_wb.active=FALSE;
    if (!pipe->ex_mem.active) return EMPTY;

    ins=pipe->ex_mem.ins;
    pipe->mem_wb.ins=ins;

    type=ins.type;
    status=OK;
    opcode=ins.opcode;
    function=ins.function;
    ptr=pipe->ex_mem.ALUOutput;
    switch (type)
    {
    case LOAD:
    case FLOAD:
        status=LOADS;
        cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=NOT_AVAILABLE;
        switch (opcode)
        {
        case I_LB:
            if (cpu->dstat[ptr]==VACANT) status=DATA_ERR;
            b=*(BYTE *)(&cpu->data[ptr]);
            s=b;
            pipe->mem_wb.LMD=(s<<56)>>56;
            break;
        case I_LBU:
            if (cpu->dstat[ptr]==VACANT) status=DATA_ERR;
            b=*(BYTE *)(&cpu->data[ptr]);
            u=b;
            pipe->mem_wb.LMD=(u<<56)>>56;
            break;
        case I_LH:
            for (i=0;i<2;i++) 
                if (cpu->dstat[ptr+i]==VACANT) status=DATA_ERR;
            h=*(WORD16 *)(&cpu->data[ptr]);
            s=h;
            pipe->mem_wb.LMD=(s<<48)>>48;
            break;
        case I_LHU:
            for (i=0;i<2;i++) 
                if (cpu->dstat[ptr+i]==VACANT) status=DATA_ERR;
            h=*(WORD16 *)(&cpu->data[ptr]);
            u=h;
            pipe->mem_wb.LMD=(u<<48)>>48;
            break;
        case I_LW:
            for (i=0;i<4;i++) 
                if (cpu->dstat[ptr+i]==VACANT) status=DATA_ERR;
            w=*(WORD32 *)(&cpu->data[ptr]);
            s=w;
            pipe->mem_wb.LMD=(s<<32)>>32;
            break;
        case I_LWU:
            for (i=0;i<4;i++) 
                if (cpu->dstat[ptr+i]==VACANT) status=DATA_ERR;
            w=*(WORD32 *)(&cpu->data[ptr]);
            u=w;
            pipe->mem_wb.LMD=(u<<32)>>32;
            break;
        case I_LD:
        case I_L_D:
            for (i=0;i<8;i++) 
                if (cpu->dstat[ptr+i]==VACANT) status=DATA_ERR;
            pipe->mem_wb.LMD=*(WORD64 *)(&cpu->data[ptr]);
            break;
        default:
            break;
        }
        if (forwarding && ins.rt!=0)
        {
            cpu->wreg[ins.rt].val=pipe->mem_wb.LMD;
            cpu->wreg[ins.rt].source=FROM_MEM;
        }
        break;
    case STORE:
    case FSTORE:
        rB=pipe->ex_mem.rB;
        if (!available(cpu,rB)) {*rawreg=rB; return RAW;}
        status=STORES;
        switch (opcode)
        {
        case I_SB:
            cpu->dstat[ptr]=WRITTEN;
            b=(BYTE)cpu->rreg[rB].val;
            *(BYTE *)(&cpu->data[ptr])=b;
            break;
        case I_SH:
            for (i=0;i<2;i++) cpu->dstat[ptr+i]=WRITTEN;
            h=(WORD16)cpu->rreg[rB].val;
            *(WORD16 *)(&cpu->data[ptr])=h;
            break;
        case I_SW:
            for (i=0;i<4;i++) cpu->dstat[ptr+i]=WRITTEN;
            w=(WORD32)cpu->rreg[rB].val;
            *(WORD32 *)(&cpu->data[ptr])=w;
            break;
        case I_SD:
        case I_S_D:
            for (i=0;i<8;i++) cpu->dstat[ptr+i]=WRITTEN;
            *(WORD64 *)(&cpu->data[ptr])=cpu->rreg[rB].val;
            break;
        default:
            break;
        }
        break;
    case REG3:
    case REG3F:
    case REG2S:
        if (forwarding && ins.rd!=0) cpu->wreg[ins.rd].source=FROM_MEM;
        break;
    case REG1I:
    case REG2I:
    case REG2F:
    case REGID:
        if (forwarding && ins.rt!=0) cpu->wreg[ins.rt].source=FROM_MEM;
    case JUMP:
    case JREG:
        if (forwarding && opcode==I_JAL || (opcode==I_SPECIAL && function==R_JALR))
            cpu->wreg[31].source = FROM_MEM;
        break;
    default:
        break;
    }

    pipe->ex_mem.active=FALSE;
    pipe->mem_wb.active=TRUE;
    pipe->mem_wb.ALUOutput=ptr;
    pipe->mem_wb.IR=pipe->ex_mem.IR;
    pipe->mem_wb.ins=ins;
    pipe->mem_wb.NPC=pipe->ex_mem.NPC;

    return status;
}


/*
   Write Back. Since writing takes place on the leading edge, update both the 
   read and write registers
*/

int WB(pipeline *pipe,processor *cpu,BOOL forwarding)
{
    int status,type;
    instruction ins=pipe->mem_wb.ins;
   
    if (!pipe->mem_wb.active) return EMPTY;

    type=ins.type;
    status=OK;
    switch (type)
    {
    case LOAD:
    case FLOAD:
        if (ins.rt==0) break;
        if (!forwarding)
        {
            cpu->rreg[ins.rt].val=cpu->wreg[ins.rt].val=pipe->mem_wb.LMD;
            cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=FROM_REGISTER;
        }
        else
        { /* be careful in case a new value is now being forwarded */
          /* from further back... */
            if (cpu->rreg[ins.rt].source==FROM_MEM) 
                cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=FROM_REGISTER;
        }
        break;
    case REG1I:
    case REG2I:
    case REGID:
    case REG2F:
        if (ins.rt==0) break;
        if (!forwarding)
        {
            cpu->rreg[ins.rt].val=cpu->wreg[ins.rt].val=pipe->mem_wb.ALUOutput;
            cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=FROM_REGISTER;
        }
        else
        {
            if (cpu->rreg[ins.rt].source==FROM_MEM)
                cpu->rreg[ins.rt].source=cpu->wreg[ins.rt].source=FROM_REGISTER;
        }
        break;
    case REG3:
    case REG2S:
    case REG3F:
        if (ins.rd==0) break;
        if (!forwarding)
        {
            cpu->rreg[ins.rd].val=cpu->wreg[ins.rd].val=pipe->mem_wb.ALUOutput;
            cpu->rreg[ins.rd].source=cpu->wreg[ins.rd].source=FROM_REGISTER;
        }
        else
        {
            if (cpu->rreg[ins.rd].source==FROM_MEM)
                cpu->rreg[ins.rd].source=cpu->wreg[ins.rd].source=FROM_REGISTER;
        }
        break;
    case JUMP:
        if (ins.opcode==I_JAL)
        {
            if (!forwarding)
            {
                cpu->rreg[31].val=cpu->wreg[31].val=pipe->mem_wb.NPC;
                cpu->rreg[31].source=cpu->wreg[31].source=FROM_REGISTER;
            }  
            else
            {
                if (cpu->rreg[31].source==FROM_MEM)
                    cpu->rreg[31].source=cpu->wreg[31].source=FROM_REGISTER;
            }
        }
        break;
    case JREG:
        if (ins.opcode==I_SPECIAL && ins.function==R_JALR)
        {
            if (!forwarding)
            { 
                cpu->rreg[31].val=cpu->wreg[31].val=pipe->mem_wb.NPC;
                cpu->rreg[31].source=cpu->wreg[31].source=FROM_REGISTER;
            }
            else
            {
                if (cpu->rreg[31].source==FROM_MEM)
                    cpu->rreg[31].source=cpu->wreg[31].source=FROM_REGISTER;
            }
        }
        break;
    case HALT:
        status=HALTED;
        break;
    default:
        break;
    }
    pipe->mem_wb.active=FALSE;
    return status;
}

static void unpack32(WORD32 a,BYTE *b)
{ /* unpack bytes from a word */
    b[0]=(BYTE)a;
    b[1]=(BYTE)(a>>8);
    b[2]=(BYTE)(a>>16);
    b[3]=(BYTE)(a>>24);
}

static void unpack64(WORD64 a,BYTE *b)
{
    int i;
    for (i=0;i<8;i++)
    {
        b[i]=(BYTE)a;
        a>>=8;
    }
}

void printword64(WORD64 val)
{
    int i;
    BYTE a[8];
    val&=(-1);
    unpack64(val,a);
    for (i=0;i<8;i++) printf("%02x",a[7-i]);
}

WORD32 clock_tick(pipeline *pipe,processor *mips,BOOL forwarding,BOOL delay_slot,int *rawreg)
{
    int i,status;

/* activate WB first as it activates on leading edge */

    status=WB(pipe,mips,forwarding);
    if (status==HALTED) return HALTED;           /* halted */
    status<<=4;
    status|=MEM(pipe,mips,forwarding,rawreg);
    status<<=4;
    status|=EX(pipe,mips,forwarding,rawreg);
    status<<=4;
    status|=ID(pipe,mips,forwarding,rawreg);
    status<<=4;
    status|=IF(pipe,mips,delay_slot);

/* Copy Write to Read registers */
 
    for (i=0;i<64;i++) mips->rreg[i]=mips->wreg[i];

    return status;
}

void pipe_stat(pipeline *pipe)
{
    int i;
    if (pipe->if_id.active) printf("IF/ID is active %d\n",pipe->if_id.ins.type);
    if (pipe->id_ex.active) printf("ID/EX is active %d\n",pipe->id_ex.ins.type);
    for (i=0;i<10;i++) 
        if (pipe->a[i].active) printf("A[%d] is active %d\n",i+1,pipe->a[i].ins.type);

    for (i=0;i<10;i++) 
        if (pipe->m[i].active) printf("M[%d] is active %d\n",i+1,pipe->m[i].ins.type);

    if (pipe->ex_mem.active) printf("EX/MEM is active %d\n",pipe->ex_mem.ins.type);
    if (pipe->mem_wb.active) printf("MEM/WB is active %d\n",pipe->mem_wb.ins.type);
    if (pipe->branch) printf("Branch set\n");
}

void cpu_stat(processor *cpu)
{
    int i;
    printf("PC= %d\n",cpu->PC);
    for (i=1;i<5;i++)
    { 
        printf("R[%d]=  %I64d ",i,cpu->rreg[i].val);
        if (cpu->rreg[i].source==NOT_AVAILABLE) printf("NOT AVAILABLE\n"); 
        if (cpu->rreg[i].source==FROM_REGISTER) printf("FROM REGISTER\n"); 
        if (cpu->rreg[i].source==FROM_ID) printf("FROM ID\n"); 
        if (cpu->rreg[i].source==FROM_EX) printf("FROM EX\n"); 
        if (cpu->rreg[i].source==FROM_MEM) printf("FROM_MEM\n"); 
    }
}

int main(int argc,char **argv)
{
    FILE *fp;
    WORD32 codeword;
    WORD64 dataword;
    int i,ptr,status,rawreg;
    int DIV_LATENCY=24;
    int MUL_LATENCY=7;
    int ADD_LATENCY=4;

    char fname[100];
    BOOL forwarding,delay_slot;
    processor mips;
    pipeline pipe;

    init_processor(&mips);
    init_pipeline(&pipe,ADD_LATENCY,MUL_LATENCY,DIV_LATENCY);

    forwarding=TRUE;
    delay_slot=FALSE;

    argc--;argv++;
    if (argc!=1)
    {
        printf("Incorrect usage\n");
        printf("pipeline <file>\n");
        printf("and <file> is the input file with .cod and .dat extensions\n");
        exit(0);
    }

    mips.code=malloc(1000);
    mips.data=malloc(1000);
    mips.dstat=malloc(1000);

    for (i=0;i<1000;i++)
    {
        mips.code[i]=mips.data[i]=mips.dstat[i]=0;
    }

    strcpy(fname,argv[0]);
    strcat(fname,".cod");

    fp=fopen(fname,"r");
    if (fp==NULL)
    {
        printf("No such file\n");
        exit(0);
    }
    for (ptr=0;;ptr+=4)
    {
        fscanf(fp,"%08x",&codeword);
        if (feof(fp)) break;
        unpack32(codeword,&mips.code[ptr]);
        printf("%08x\n",codeword);

    }
    fclose(fp);

    strcpy(fname,argv[0]);
    strcat(fname,".dat");

    fp=fopen(fname,"r");
    if (fp!=NULL)
    {
        for (ptr=0;;ptr+=8)
        {
            fscanf(fp,"%016I64x",&dataword);
            if (feof(fp)) break;
            printword64(dataword); printf("\n");
            unpack64(dataword,&mips.data[ptr]);
            for (i=0;i<8;i++) mips.dstat[ptr+i]=WRITTEN;
        }
        fclose(fp);
    }

    for (i=0;i<24;i++)
    {
        status=clock_tick(&pipe,&mips,forwarding,delay_slot,&rawreg);
        if (status==HALTED) break;
        printf("status= %d\n",status);
        pipe_stat(&pipe);
        cpu_stat(&mips);
        printf("\n");
    }

    return 0;
}


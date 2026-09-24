#include "stdafx.h"
#include "simulator.h"


static int hot_interrupt(WORDFULL *ir)
{	// returns the index of the first interrupt that is enabled and pending, else -1
	// resets the pending flag
	int i,bits=HBITS;
	for (i=0;i<bits;i++)
	{
		if ((*ir>>i)&1)
		{ // this one is enabled
			if ((*ir>>(bits+i))&1)
			{ // this one is hot
				*ir^=(1<<(bits+i));
				return i;
			}
		}
	}
	return -1;
}

static UFULLWORD swap(UFULLWORD a)
{ /* swap halves of a */
	UFULLWORD r;
	r=a>>HBITS;
	r=((a<<HBITS) | r);
	return r;
}

static int nbytes(int bits)
{
	int r=bits/8;
	if (bits%8!=0) r++;
	return r;
}

static UFULLWORD pack(BYTE *ptr,int n,BOOL extend)
{	/* pack n bit number from byte buffer into one FULLWORD 
	   Sign-extend if required */
	UFULLWORD r=0;
	SFULLWORD s;
	int bytes=nbytes(n);

	while (bytes>0)
	{
		r<<=8;
		r|=*(ptr+bytes-1);
		bytes--;
	}
	if (extend)
	{
		if (n>=FBITS) return r;
		s=r;
		s<<=(FBITS-n);
		s>>=(FBITS-n);
		r=s;
	}
	return r;
}

static void unpack(UFULLWORD w,BYTE *ptr,int n)
{ /* unpack n bits from FULLWORD w into bytes */
	int bytes=nbytes(n);
	while (bytes>0)
	{
		*ptr++=(BYTE)w;
		w/=256;
		bytes--;
	}
}

static void update(UFULLWORD *r,int index,UFULLWORD value)
{ /* some registers are NBITS, other are 32-bit */
	r[index]=value&MASK;
}

static SFULLWORD extend(UFULLWORD r)
{ /* sign extend the value r */
	SFULLWORD s=r;
	if (NBITS>=FBITS) return s;
	s<<=(FBITS-NBITS);
	s>>=(FBITS-NBITS);
	return s;
}

int simulator(int &processor_status,int &interrupt_status,WORD32 &pc,WORDFULL *r,WORD32 gprs,WORDFULL *s,WORD32 sprs,WORDFULL *cam,BYTE *code,WORD32 codesize,BYTE *data,BYTE *data_status,WORD32 datasize,BYTE *datax,WORD32 dataxsize,WORD32 &result,WORD32 &regaddr,WORD32 &xmemaddr)
{
//	char txt[100];
	int i,inum,nb,err=0;
	BOOL RD;
	UFULLWORD answer;
	WORD32 instruct,byte;
	WORD32 flags,shift;
	WORD32 opcode,rs1,rs2,rd,imm,func,offset;
	WORDFULL lh,rh1,rh2;
	short shalf;
	unsigned short uhalf;
	SWORD   snum;
	WORD32 unum,addr;
	signed char sbyte;
	result=0;
	if (interrupt_status==INACTIVE && (inum=hot_interrupt(&s[1]))>=0)
	{ // react to interrupt
		s[2]=pc;
		pc=4+inum*4;		
		interrupt_status=ACTIVE;
		return SIM_OK;
	}
	if (processor_status==HALTED) return SIM_OK;
	if (pc>=codesize) return SIM_NOCODEMEM;
	instruct=pack32(&code[pc]);
	pc=pc+4;
	rd=32;    // no such register
	opcode=(instruct>>26);
	switch (opcode)
	{
// I-TYPE Instructions
	case I_ADDI:		// ADDI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);			// now sign extend..
		shalf=(short)imm;
		answer=r[rs1]+(SFULLWORD)shalf;

		update(r,rd,answer);
		break;
	case I_ADDUI:		// ADDUI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}

		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=r[rs1]+(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_SUBUI:		// SUBUI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=r[rs1]-(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_ANDI:		// ANDI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=r[rs1]&(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_ANDXI:		// ANDXI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=swap(r[rs1])&(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_ORI:		// ORI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=r[rs1]|(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_ORXI:		// ORXI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=swap(r[rs1])|(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_XORI:		// XORI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=r[rs1]^(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;
	case I_XORXI:		// XORXI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;			// r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=swap(r[rs1])^(UFULLWORD)uhalf;
		update(r,rd,answer);
		break;

	case I_LHI:		// LHI
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		answer=(UFULLWORD)uhalf<<HBITS;
		update(r,rd,answer);
		break;

	case I_SETBI:
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		unum=(WORD32)uhalf;
		answer=r[rs1]|((UFULLWORD)1<<unum);
		update(r,rd,answer);
		break;
	case I_CLRBI:
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		unum=(WORD32)uhalf;
		answer=r[rs1]&(~((UFULLWORD)1<<unum));
		update(r,rd,answer);
		break;
	case I_SLLI:		// SLLI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		unum=(WORD32)uhalf;
		answer=r[rs1]<<unum;
		update(r,rd,answer);
		break;
	case I_SRLI:		// SRLI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		unum=(WORD32)uhalf;
		answer=r[rs1]>>unum;
		update(r,rd,answer);
		break;
	case I_SRAI:					// SRAI - this is tricky
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		unum=(WORD32)uhalf;
		answer=extend(r[rs1])>>unum;
		update(r,rd,answer);
		break;
	case I_SEQI:		// SEQI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		if (extend(r[rs1])==(SFULLWORD)shalf)	answer=1;
		else									answer=0;
		update(r,rd,answer);
		break;
	case I_SEQUI:		// SEQUI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		if (r[rs1]==(UFULLWORD)uhalf)		answer=1;
		else								answer=0;
		update(r,rd,answer);
		break;

	case I_SLTI:		// SLTI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		if (extend(r[rs1])<(SFULLWORD)shalf)	answer=1;
		else									answer=0;
		update(r,rd,answer);
		break;
	case I_SLEI:		// SLEI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		if (extend(r[rs1])<=(SFULLWORD)shalf)	answer=1;
		else									answer=0;
		update(r,rd,answer);
		break;
	case I_SLTUI:		// SLTUI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		if (r[rs1]<(UFULLWORD)uhalf)	answer=1;
		else							answer=0;
		update(r,rd,answer);
		break;
	case I_SLEUI:		// SLEUI
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		uhalf=(unsigned short)imm;
		if (r[rs1]<=(UFULLWORD)uhalf)	answer=1;
		else							answer=0;
		update(r,rd,answer);
		break;

// Load/Store Instructions

	case I_LB:		// LB
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=LOADS;
		if (addr>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		if (data_status[addr]==0) err=SIM_NOVALUEMEM;
		sbyte=data[addr];
		answer=(SFULLWORD)sbyte;
		update(r,rd,answer); 
		break;
	case I_LBU:		// LBU
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=LOADS;
		if (addr>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		if (data_status[addr]==0) err=SIM_NOVALUEMEM;
		answer=(UFULLWORD)data[addr];
		update(r,rd,answer);
		break;

	case I_LH:		// LH
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=LOADS;
		nb=nbytes(HBITS);
		if (addr+nb-1>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		for (i=0;i<nb;i++)	if (data_status[addr+i]==0)	err=SIM_NOVALUEMEM;
		answer=pack(&data[addr],HBITS,TRUE);
		update(r,rd,answer);
	
		break;
	case I_LHU:		// LHU
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=LOADS;
		nb=nbytes(HBITS);
		if (addr+nb-1>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		for (i=0;i<nb;i++)	if (data_status[addr+i]==0)	err=SIM_NOVALUEMEM;	
		answer=pack(&data[addr],HBITS,FALSE);
		update(r,rd,answer);
		break;

	case I_LW:		// LW
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		if (rd>=gprs)
		{
			err=SIM_NOREG;
			break;
		}
		if (rd==0) break;          // r[0] cannot be changed
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)((SFULLWORD)r[rs1]+(SFULLWORD)shalf);
		result=LOADS;
		nb=nbytes(NBITS);
		if (addr+nb-1>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		for (i=0;i<nb;i++)	if (data_status[addr+i]==0)	err=SIM_NOVALUEMEM;
		answer=pack(&data[addr],NBITS,FALSE);
		update(r,rd,answer);
		break;
	case I_SB:		// SB
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=STORES;
		if (addr>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		data[addr]=(unsigned char)r[rd];
		data_status[addr]=1;
		break;
	case I_SH:		// SH
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=STORES;
		nb=nbytes(HBITS);
		if (addr+nb-1>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		unpack(r[rd],&data[addr],HBITS);
		for (i=0;i<nb;i++) data_status[addr+i]=1;
		break;
	case I_SW:		// SW
		rs1=(instruct>>16)&0x1F;
		rd=(instruct>>21)&0x1F;
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		addr=(WORD32)(extend(r[rs1])+(SFULLWORD)shalf);
		result=STORES;
		nb=nbytes(NBITS);

		if (addr+nb-1>=datasize) 
		{
			err=SIM_NODATAMEM;
			break;
		}
		unpack(r[rd],&data[addr],NBITS);
		for (i=0;i<nb;i++) data_status[addr+i]=1;
		break;
	case I_BEQZ:		// BEQZ
		rs1=(instruct>>16)&0x1F;
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		snum=(SWORD)shalf;
		if (r[rs1]==0)
		{ // branch taken
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
	case I_BNEZ:		// BNEZ
		rs1=(instruct>>16)&0x1F;
		imm=(instruct&0xFFFF);
		shalf=(short)imm;
		snum=(SWORD)shalf;
		if (r[rs1]!=0)
		{ // branch taken
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
	case I_HALT:		// HALT
		processor_status=HALTED;
		break;

	case I_JR:		// JR
		rs1=(instruct>>16)&0x1F;
		pc=(WORD32)r[rs1];
		result=BRANCHTAKEN;
		break;

	case I_JALR:		// JALR
		rs1=(instruct>>16)&0x1F;
		s[3]=pc;
		pc=(WORD32)r[rs1];
		result=JUMPLINK|BRANCHTAKEN;
		break;

	case I_RET:		// RET
		pc=(WORD32)s[3];
		result=BRANCHTAKEN;
		break;

	case I_BEQB0I:		//  BEQBxI
	case I_BEQB1I:
	case I_BEQB2I:
	case I_BEQB3I:
		shift=8*(opcode-0x30);
		rs1=(instruct>>23)&0x7;
		byte=(instruct>>15)&0xFF;
		offset=(instruct&0x7FFF);
		if (offset&0x4000)
		{ // its negative
			offset|=0xFFFF8000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>shift)&0xFF)==(WORDFULL)byte)
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BNEB0I:		//  BNEBxI
	case I_BNEB1I:
	case I_BNEB2I:
	case I_BNEB3I:
		shift=8*(opcode-0x34);
		rs1=(instruct>>23)&0x7;
		byte=(instruct>>15)&0xFF;
		offset=(instruct&0x7FFF);
		if (offset&0x4000)
		{ // its negative
			offset|=0xFFFF8000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>shift)&0xFF)!=(WORDFULL)byte)
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BBS:
		rs1=(instruct>>21)&0x1F;
		byte=(instruct>>15)&0x1F;
		offset=(instruct&0x7FFF);
		if (offset&0x4000)
		{ // its negative
			offset|=0xFFFF8000;
		}
		snum=(SWORD)offset;
		if ((r[rs1]&((UFULLWORD)1<<byte))!=0)
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
	case I_BEQB0:		// BEQBx
	case I_BEQB1:
	case I_BEQB2:
	case I_BEQB3:
		shift=8*(opcode-0x38);
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>shift)&0xFF)==((r[rs2]>>shift)&0xFF))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BNEB0:		// BNEBx
	case I_BNEB1:
	case I_BNEB2:
	case I_BNEB3:
		shift=8*(opcode-0x3C);
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>shift)&0xFF)!=((r[rs2]>>shift)&0xFF))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
#ifdef B32
	case I_BEQHU:		// BEQHU
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>16)&0xFFFF)==((r[rs2]>>16)&0xFFFF))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BNEHU:		// BNEHU
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (((r[rs1]>>HBITS)&HALF_MASK)!=((r[rs2]>>HBITS)&HALF_MASK))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BEQHL:		// BEQHL
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if ((r[rs1]&HALF_MASK)==(r[rs2]&HALF_MASK))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

	case I_BNEHL:		// BNEHL
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if ((r[rs1]&HALF_MASK)!=(r[rs2]&HALF_MASK))
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
#endif
	case I_BEQW:		// BEQW
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (r[rs1]==r[rs2])
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;
	
	case I_BNEW:		// BNEW
		rs1=(instruct>>21)&0x1F;
		rs2=(instruct>>16)&0x1F;
		offset=(instruct&0xFFFF);
		if (offset&0x8000)
		{ // its negative
			offset|=0xFFFF0000;
		}
		snum=(SWORD)offset;
		if (r[rs1]!=r[rs2])
		{
			pc=(SWORD)pc+snum;
			result=BRANCHTAKEN;
		}
		else result=BRANCHNOTTAKEN;
		break;

// J-TYPE Instructions
	case I_J:		// J
		offset=(instruct&0x3FFFFFF);
		if (offset&0x2000000)
		{ // its negative
			offset|=0xFC000000;
		}
		snum=(SWORD)offset;
		pc=(SWORD)pc+snum;
		result=BRANCHTAKEN;
		break;
	case I_JAL:		// JAL
		offset=(instruct&0x3FFFFFF);
		if (offset&0x2000000)
		{ // its negative
			offset|=0xFC000000;
		}
		s[3]=pc;
		snum=(SWORD)offset;
		pc=(SWORD)pc+snum;
		result=JUMPLINK|BRANCHTAKEN;
		break;
	case I_TRAP:		// TRAP
		s[2]=pc;
		pc=4;
		result=BRANCHTAKEN;
		interrupt_status=ACTIVE;
		break;
	case I_RFE:		// RFE
		pc=(WORD32)s[2];
		result=BRANCHTAKEN;
		interrupt_status=INACTIVE;
		break;

	case 0x00:		// R-R Instructions!
		func=(instruct&0x3F);
		flags=(instruct>>6)&0x07;  // r or s registers?
		RD=FALSE;
		if (flags&0x01) RD=TRUE;
		rs1=(instruct>>16)&0x1F;
		rs2=(instruct>>11)&0x1F;
		rd =(instruct>>21)&0x1F;
		
		rh1=r[rs1]; if (flags&0x04) rh1=s[rs1];
		rh2=r[rs2]; if (flags&0x02) rh2=s[rs2];
		switch (func)
		{
// R-TYPE Instructions
		case R_ADD:		// ADD
			lh=rh1+rh2;
			break;
		case R_SUB:		// SUB
			lh=rh1-rh2;
			break;
		case R_AND:		// AND
			lh=rh1&rh2;
			break;
		case R_ANDX:	// ANDX
			lh=swap(rh1)&rh2;
			break;
		case R_ANDS:	// ANDS
			lh=swap(rh1&rh2);
			break;
		case R_OR:		// OR
			lh=rh1|rh2;;
			break;
		case R_ORX:		// ORX
			lh=swap(rh1)|rh2;
			break;
		case R_ORS:		// ORS
			lh=swap(rh1|rh2);
			break;
		case R_XOR:		// XOR
			lh=rh1^rh2;
			break;
		case R_XORX:
			lh=swap(rh1)^rh2; // XORX - swap halves of rh1
			break;
		case R_XORS:
			lh=swap(rh1^rh2); // XORS - swap halves of result
			break;
		case R_SETB:
			lh=rh1|((UFULLWORD)1<<rh2);
			break;
		case R_CLRB:
			lh=rh1&(~((UFULLWORD)1<<rh2));
			break;
		case R_SLL:		// SLL
			lh=(rh1<<rh2);
			break;
		case R_SRL:		// SRL
			lh=(rh1>>rh2);
			break;
		case R_SRA:		// SRA
			lh=(extend(rh1)>>rh2);
			break;
		case R_SLT:		// SLT
			if (extend(rh1)<extend(rh2))	lh=1;
			else							lh=0;
			break;
		case R_SLTU:		// SLTU
			if (rh1<rh2)	lh=1;
			else			lh=0;
			break;
		case R_SLE:		// SLE
			if (extend(rh1)<=extend(rh2))	lh=1;
			else							lh=0;
			break;
		case R_SLEU:		// SLEU
			if (rh1<=rh2)	lh=1;
			else			lh=0;
			break;
		case R_SEQ:		// SEQ
			if (rh1==rh2)	lh=1;
			else			lh=0;
			break;
	
		case R_CAMWR:	// CAM write
			i=(int)(rh2&0xF);
			if (i<CAM_SIZE) cam[i]=rh1;
			lh=0;
			break;
		case R_CAMLU:	// CAM look-up
			lh=rh2;
			for (i=0;i<CAM_SIZE;i++)
			{
				if (rh1==cam[i])
				{
					lh&=(UFULLWORD)(~4);		// clear bit 2
					break;
				}
			}
			break;
	
		case R_L256:
			lh=0;
			if (rh2+31>=dataxsize) err=SIM_NODATAXMEM;

			regaddr=(WORD32)(rh1&0x1F);
			xmemaddr=(WORD32)rh2;

			result=XMEMLOAD256;
			break;
		case R_L128:
			lh=0;
			if (rh2+15>=dataxsize) err=SIM_NODATAXMEM;

			regaddr=(WORD32)(rh1&0x1F);
			xmemaddr=(WORD32)rh2;

			result=XMEMLOAD128;
			break;
		case R_S256:
			lh=0;
			if (rh2+31>=dataxsize) err=SIM_NODATAXMEM;

			regaddr=(WORD32)(rh1&0x1F);
			xmemaddr=(WORD32)rh2;

			result=XMEMSTORE256;
			break;
		case R_S128:
			lh=0;
			if (rh2+15>=dataxsize) err=SIM_NODATAXMEM;

			regaddr=(WORD32)(rh1&0x1F);
			xmemaddr=(WORD32)rh2;

			result=XMEMSTORE128;
			break;
	
		default:
			err=SIM_ILLEGAL;
			break;
		}
		if (err) break;
		if (!RD)
		{
			if (rd>=gprs) 
			{
				err=SIM_NOREG;
				break;
			}
			if (rd!=0) r[rd]=(lh&MASK); // r[0] not modified
		}
		else
		{
			if (rd>=sprs)
			{
				err=SIM_NOREG;
				break;
			}
			s[rd]=(lh&MASK);
			rd=32;
		}
		break;
	default:
		err=SIM_ILLEGAL;
		break;
	}
	result|=(rd<<16);
	return err;
}

// Have a look at the next instruction.
// Determine which registers it needs to read, 
// and if its an I-TYPE Branch (which needs the register earlier) 

int peek_at_next(WORD32 instruct,int &r1,int&r2)
{
	int res=0;
	WORD32 flags,opcode=(instruct>>26);

	r1=r2=32;  // no such register
	switch (opcode)
	{
	case I_ADDI:
	case I_ADDUI:
	case I_SUBUI:
	case I_ANDI:
	case I_ANDXI:
	case I_ORI:
	case I_ORXI:
	case I_XORI:
	case I_XORXI:
	case I_SLLI:
	case I_SRLI:
	case I_SRAI:
	case I_SEQI:
	case I_SLTI:
	case I_SLEI:
	case I_SLTUI:
	case I_SLEUI:
	case I_LB:
	case I_LBU:
	case I_LH:
	case I_LHU:
	case I_LW:
		r1=(instruct>>16)&0x1F;
		res=REGREAD;
		break;
	case I_BEQW:  // new branch instructions
	case I_BNEW:
	case I_BEQHU:
	case I_BNEHU:
	case I_BEQHL:
	case I_BNEHL:
	case I_BEQB0:
	case I_BEQB1:
	case I_BEQB2:
	case I_BEQB3:
	case I_BNEB0:
	case I_BNEB1:
	case I_BNEB2:
	case I_BNEB3:
		r1=(instruct>>16)&0x1F;
		r2=(instruct>>21)&0x1F;
		res=ITYPE_BRANCH;
		break;
	case I_SB:
	case I_SH:
	case I_SW:
		r1=(instruct>>16)&0x1F;
		r2=(instruct>>21)&0x1F;
		res=REGREAD;
		break;
	case I_BEQB0I: // new branch instructions
	case I_BEQB1I:
	case I_BEQB2I:
	case I_BEQB3I:
	case I_BNEB0I:
	case I_BNEB1I:
	case I_BNEB2I:
	case I_BNEB3I:
		r1=(instruct>>23)&0x07;
		res=ITYPE_BRANCH;
		break;
	case I_BBS:
		r1=(instruct>>21)&0x1F;
		res=ITYPE_BRANCH;
		break;
	case I_BEQZ:
	case I_BNEZ:
	case I_JR:
	case I_JALR:
		r1=(instruct>>16)&0x1F;
		res=ITYPE_BRANCH;
		break;
	case 0x00:
		flags=(instruct>>6)&0x07;
		if (!(flags&0x04)) r1=(instruct>>16)&0x1F;
		if (!(flags&0x02)) r2=(instruct>>11)&0x1F;
		res=REGREAD;
		break;
	default:
		break;
	}	
	return res;
}
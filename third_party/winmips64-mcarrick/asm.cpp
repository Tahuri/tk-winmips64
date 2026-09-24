/*
 * MIPS64 assembler
 * M.Scott
 * March 2006
 *
 * Compile from the command line as 
 * d:>cl /O2 /GX asm.cpp
 */

#include <stdio.h>
#include <string.h>
#include <stdlib.h>
#include <ctype.h>

#define FALSE 0
#define TRUE  1

#define ENDLINE 1
#define SPACE   2
#define COMMENT 3

typedef unsigned char BYTE;
typedef unsigned __int64 WORD64;
typedef __int64 SIGNED64;
typedef unsigned long WORD32;
typedef long SIGNED32;
typedef short SIGNED16;
typedef int BOOL;

union
{
    WORD64 u;
    SIGNED64 s;
    double d;
} typedef DOUBLE64;

#define R_TYPE 1
#define I_TYPE 2
#define J_TYPE 3
#define F_TYPE 4
#define M_TYPE 5
#define B_TYPE 6

#define NOP 0
#define LOAD  1
#define STORE 2
#define REG1I 3
#define REG2I 4
#define REG2S 5
#define JUMP  6
#define JREG  7
#define HALT  8
#define REG3F 9
#define BRANCH 10
#define REG3   11
#define REGID  12
#define FLOAD  13
#define FSTORE 14
#define JREGN  15
#define REG2F  16
#define REG3X  17
#define REGDI  18
#define REG2C  19
#define BC     20

#define I_SPECIAL       0x00
#define I_COP1          0x11
#define I_DOUBLE        0x11
#define I_MTC1          0x04
#define I_MFC1          0x00
#define I_BC            0x08
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
#define F_CVT_D_L       0x21
#define F_CVT_L_D       0x25
#define F_C_LT_D        0x3C
#define F_C_LE_D        0x3E
#define F_C_EQ_D        0x32

typedef struct
{
    char *name;
    BYTE type;
    BYTE subtype;
    WORD32 op_code;
} op_code_info;

static char *directives[]=
{".space",
 ".asciiz",
 ".align",
 ".word",
 ".byte",
 ".ascii",
 ".global",
 ".data",
 ".text",
 ".org",
 ".word32", /* 10 */
 ".word16", /* 11 */
 ".double", /* 12 */
 ".code",
 NULL};

#define SI(x) ((WORD32)x<<26)
#define SR(x) ((WORD32)x | I_SPECIAL << 26)
#define SF(x) ((WORD32)x | I_COP1 << 26 | I_DOUBLE << 21)
#define SMTC1 (I_COP1 << 26 | I_MTC1 << 21)
#define SMFC1 (I_COP1 << 26 | I_MFC1 << 21)
#define SBC1F (I_COP1 << 26 | I_BC << 21)
#define SBC1T (I_COP1 << 26 | I_BC << 21 | 1 << 16)  

static op_code_info codes[]={
{"lb",     I_TYPE, LOAD,   SI(I_LB)},
{"lbu",    I_TYPE, LOAD,   SI(I_LBU)},
{"sb",     I_TYPE, STORE,  SI(I_SB)},
{"lh",     I_TYPE, LOAD,   SI(I_LH)},
{"lhu",    I_TYPE, LOAD,   SI(I_LHU)},
{"sh",     I_TYPE, STORE,  SI(I_SH)},
{"lw",     I_TYPE, LOAD,   SI(I_LW)},   
{"lwu",    I_TYPE, LOAD,   SI(I_LWU)},   
{"sw",     I_TYPE, STORE,  SI(I_SW)},
{"ld",     I_TYPE, LOAD,   SI(I_LD)},
{"sd",     I_TYPE, STORE,  SI(I_SD)},
{"l.d",    I_TYPE, FLOAD,  SI(I_L_D)},
{"s.d",    I_TYPE, FSTORE, SI(I_S_D)},
{"halt",   I_TYPE, HALT,   SI(I_HALT)},

{"daddi",  I_TYPE, REG2I,  SI(I_DADDI)},
{"daddui", I_TYPE, REG2I,  SI(I_DADDIU)},
{"andi",   I_TYPE, REG2I,  SI(I_ANDI)},
{"ori",    I_TYPE, REG2I,  SI(I_ORI)},
{"xori",   I_TYPE, REG2I,  SI(I_XORI)},
{"lui",    I_TYPE, REG1I,  SI(I_LUI)},

{"slti",   I_TYPE, REG2I,  SI(I_SLTI)},
{"sltiu",  I_TYPE, REG2I,  SI(I_SLTIU)},

{"beq",    I_TYPE, BRANCH,  SI(I_BEQ)},
{"bne",    I_TYPE, BRANCH,  SI(I_BNE)},
{"beqz",   I_TYPE, JREGN,   SI(I_BEQZ)},
{"bnez",   I_TYPE, JREGN,   SI(I_BNEZ)},


{"j",      J_TYPE, JUMP,   SI(I_J)},
{"jr",     R_TYPE, JREG,   SR(R_JR)},
{"jal",    J_TYPE, JUMP,   SI(I_JAL)},
{"jalr",   R_TYPE, JREG,   SR(R_JALR)},

{"dsll",   R_TYPE, REG2S,  SR(R_DSLL)},
{"dsrl",   R_TYPE, REG2S,  SR(R_DSRL)},
{"dsra",   R_TYPE, REG2S,  SR(R_DSRA)},
{"dsllv",  R_TYPE, REG3,   SR(R_DSLLV)},
{"dsrlv",  R_TYPE, REG3,   SR(R_DSRLV)},
{"dsrav",  R_TYPE, REG3,   SR(R_DSRAV)},
{"movz",   R_TYPE, REG3,   SR(R_MOVZ)},
{"movn",   R_TYPE, REG3,   SR(R_MOVN)},
{"nop",    R_TYPE, NOP,    SR(R_NOP)},
{"and",    R_TYPE, REG3,   SR(R_AND)},
{"or",     R_TYPE, REG3,   SR(R_OR)},
{"xor",    R_TYPE, REG3,   SR(R_XOR)},
{"slt",    R_TYPE, REG3,   SR(R_SLT)},
{"sltu",   R_TYPE, REG3,   SR(R_SLTU)},

{"dadd",   R_TYPE, REG3,   SR(R_DADD)},
{"daddu",  R_TYPE, REG3,   SR(R_DADDU)},
{"dsub",   R_TYPE, REG3,   SR(R_DSUB)},
{"dsubu",  R_TYPE, REG3,   SR(R_DSUBU)},

{"dmul",   R_TYPE, REG3X,   SR(R_DMUL)},
{"dmulu",  R_TYPE, REG3X,   SR(R_DMULU)},
{"ddiv",   R_TYPE, REG3X,   SR(R_DDIV)},
{"ddivu",  R_TYPE, REG3X,   SR(R_DDIVU)},

{"add.d",  F_TYPE, REG3F,   SF(F_ADD_D)},
{"sub.d",  F_TYPE, REG3F,   SF(F_SUB_D)},
{"mul.d",  F_TYPE, REG3F,   SF(F_MUL_D)},
{"div.d",  F_TYPE, REG3F,   SF(F_DIV_D)},
{"mov.d",  F_TYPE, REG2F,   SF(F_MOV_D)},
{"cvt.d.l",F_TYPE, REG2F,   SF(F_CVT_D_L)},
{"cvt.l.d",F_TYPE, REG2F,   SF(F_CVT_L_D)},
{"c.lt.d", F_TYPE, REG2C,   SF(F_C_LT_D)},
{"c.le.d", F_TYPE, REG2C,   SF(F_C_LE_D)},
{"c.eq.d", F_TYPE, REG2C,   SF(F_C_EQ_D)},

{"bc1f",   B_TYPE, BC,     SBC1F},
{"bc1t",   B_TYPE, BC,     SBC1T},
{"mtc1",   M_TYPE, REGID,  SMTC1},
{"mfc1",   M_TYPE, REGDI,  SMFC1},

{NULL,    0,      0,      0}
};

#define CODESIZE 16384
#define DATASIZE 16384

#define SYMTABSIZE 1000

#define CODE 1
#define DATA 2

int NBITS;   /* =64  */
int STEP;
int MASK;
int CODEORDATA;

BYTE codemem[CODESIZE];       /* Bytes generated */
BYTE datamem[DATASIZE];       /* Data area       */
WORD32 codeptr;
WORD32 dataptr;

typedef struct
{
    char *symb;
    BYTE type;
    WORD32 value;
} symbol_table;

symbol_table code_table[SYMTABSIZE];
symbol_table data_table[SYMTABSIZE];
int code_symptr;
int data_symptr;

static WORD64 pack(BYTE *b)
{
    int i;
    WORD64 r=0;
    for (i=STEP-1;i>=0;i--)
    {
        r<<=8;
        r|=b[i];    
    }
    return r;
}

static void unpack(WORD64 a,BYTE *b)
{
    int i;
    for (i=0;i<STEP;i++)
    {
        b[i]=(BYTE)a;
        a>>=8;
    }
}

void printword32(WORD32 ptr)
{
    printf("%08x",ptr);
}

void printword(WORD64 val)
{
    int i;
    BYTE a[8];
    val&=MASK;
    unpack(val,a);
    for (i=0;i<STEP;i++) printf("%02x",a[STEP-i-1]);
}

void printdouble(double db)
{
    printf("%016le",db);
}

int delimiter(int c)
{
    if (c==';' || c=='#') return COMMENT;
    if (c==0 || c=='\n' ) return ENDLINE;
    if (c==' ' || c==9)  return SPACE;
    return 0;
}

int compare(char *ptr1,char *ptr2)
{ /* compare strings, up to 0 in the second with all of the first
     returns number of bytes successfully compared                */
    int incr=0;
    if (ptr1==NULL) return 0;
    while (*ptr2==*ptr1)
    {
        if (*ptr2==0) break;
        ptr1++; ptr2++; incr++;
    }    
    if (*ptr2==0 && !isalnum(*ptr1)) return incr;
    return 0;
         
}

int is_symbol(char *start)
{
    int len=0;
    char *ptr=start;
    while (!delimiter(*ptr)) { ptr++; len++; }
    ptr--; len--;
    if (*ptr==':') return len; 
    return 0;
}

char *eatwhite(char *ptr)
{
    if (ptr==NULL) return NULL;
    while (*ptr==' ' || *ptr==9) ptr++;  
    if (*ptr==0 || *ptr==';' || *ptr=='#' || *ptr=='\n') return NULL;
    return ptr; 
}

int getreg(char *&ptr)
{
    int n,ch,type;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return (-1);
    ch=tolower(*ptr);
    if (ch!='r' && ch!='$') return (-1);
    ptr++;
    if (ch=='r' && !isdigit(*ptr)) return (-1);
/* check for register pseudo-name */
    type=0;
    if (strncmp(ptr,"zero",4)==0) {n=0; ptr+=4; return n;}
    if (strncmp(ptr,"at",2)==0) {n=1; ptr+=2; return n;}
    if (strncmp(ptr,"gp",2)==0) {n=28; ptr+=2; return n;}
    if (strncmp(ptr,"sp",2)==0) {n=29; ptr+=2; return n;}
    if (strncmp(ptr,"fp",2)==0) {n=30; ptr+=2; return n;}      
    if (strncmp(ptr,"ra",2)==0) {n=31; ptr+=2; return n;} 
    if (strncmp(ptr,"v",1)==0) {type=1; ptr++;}
    if (strncmp(ptr,"a",1)==0) {type=2; ptr++;}
    if (strncmp(ptr,"t",1)==0) {type=3; ptr++;}
    if (strncmp(ptr,"s",1)==0) {type=4; ptr++;}
    if (strncmp(ptr,"k",1)==0) {type=5; ptr++;}

    n=(int)atoi(ptr);
    ptr++;
    while (isdigit(*ptr)) ptr++;

    switch (type)
    {
    case 1:
        if (n<0 || n>1) return -1;
        n+=2;
        break;
    case 2:
        if (n<0 || n>3) return -1;
        n+=4;
        break;
    case 3:
        if (n<0 || n>9) return -1;
        if (n<8) n+=8;
        else n+=16;
        break;
    case 4:
        if (n<0 || n>7) return -1;
        n+=16;
        break;
    case 5:
        if (n<0 || n>1) return -1;
        n+=26;
        break;

    default: break;
    }

    if (n<0 || n>31) return (-1);
    return n;    
}

int fgetreg(char *&ptr)
{
    int n;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return (-1);
    if (tolower(*ptr)!='f') return (-1);
    ptr++;
    if (!isdigit(*ptr)) return (-1);
    n=(int)atoi(ptr);
    ptr++;
    while (isdigit(*ptr)) ptr++;
    if (n<0 || n>31) return (-1);
    return n;    
}

WORD64 strtoint64(char *&ptr)
{
    WORD64 n=0;
    int ch,s=0;
    int base=10;

    while (*ptr==' ' || *ptr==9) ptr++;
    if (*ptr=='-') {s=1; ptr++;}
    else if (*ptr=='+') ptr++;
    while (*ptr==' ' || *ptr==9) ptr++;
    if (*ptr=='0')
    {
        ptr++;
        if (*ptr=='x' || *ptr=='X') 
        {
            base=16;
            ptr++;
        }
        else
            base=8;
    }
    for (;;)
    {
        ch=*ptr;
        if (base==8)
        {
            if (ch<'0' || ch>'7') break;
            n=n*base+(ch-'0'); 
        }
        if (base==10)
        {
            if (ch<'0' || ch>'9') break;
            n=n*base+(ch-'0');
        }
        if (base==16)
        {
           if ((ch<'0' || ch>'9') && (ch<'A' || ch>'F') && (ch<'a' || ch>'f'))
               break;
           if (ch>='0' && ch<='9') n=n*base+(ch-'0');
           if (ch>='A' && ch<='F') n=n*base+10+(ch-'A');
           if (ch>='a' && ch<='f') n=n*base+10+(ch-'a');
        }
        ptr++;
    }
    if (s==1) return ((~n)+1);  /* -n */
    else return n;
}

BOOL getfullnum(char *&ptr,WORD64 *num)
{
    WORD64 m;
    *num=0;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return FALSE;
//    if (*ptr=='#') ptr++;   /* skip over # */
    if (!isdigit(*ptr) && *ptr!='-' && *ptr!='+') return FALSE;

    m=strtoint64(ptr);

    *num=m;

    return TRUE;
}


BOOL getdouble(char *&ptr,double *num)
{
    double m;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return FALSE;
//    if (*ptr=='#') ptr++;   /* skip over # */
    if (!isdigit(*ptr) && *ptr!='.' && *ptr!='-' && *ptr!='+') return FALSE;

    m=strtod(ptr,&ptr);

    *num=m;

    return TRUE;
}

BOOL getnum(char *&ptr,WORD32 *n)
{
    int sign=1;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return FALSE;
//    if (*ptr=='#') ptr++;   /* skip over # */
    if (!isdigit(*ptr) && *ptr!='-' && *ptr!='+') return FALSE;

    if (*ptr=='-') {sign=-1; ptr++;}
    if (*ptr=='+') ptr++;

    *n=(int)strtoul(ptr,&ptr,0);

    *n*=sign;

    while (isdigit(*ptr)) ptr++;

    return TRUE;
}

/* 
   get symbol from a symbol table
   This should use a hash table....
*/

BOOL getsym(symbol_table *table,int size,char *&ptr,WORD32 *n)
{
    int k,incr;
    WORD32 m;
    char *sptr;
    ptr=eatwhite(ptr);
    if (ptr==NULL) return FALSE;
    if (getnum(ptr,&m))
    { /* its a number! */
        *n=m;
        return TRUE;
    }
    for (k=0;k<size;k++)
    {
        sptr=table[k].symb;
        incr=compare(ptr,sptr);
        if (incr)
        {
            *n=table[k].value;
/* check for + or - an integer */
             ptr+=incr;
             if (eatwhite(ptr)==NULL) return TRUE;
             ptr=eatwhite(ptr);
             if (*ptr=='-' || *ptr=='+')
             {
                 if (getnum(ptr,&m)) *n+=m;
             }
             return TRUE;
        }
    }
    return FALSE;
}


char *skipover(char *ptr,char c)
{ /* skip over anything  to get to c */

    int res;
    while (*ptr!=c)
    {
        res=delimiter(*ptr);
        if (res==ENDLINE || res==COMMENT) return NULL;
        if (c==',' && *ptr=='(') return NULL;
        ptr++;
    }
    ptr++;
    return ptr;
}

char *skip(char *ptr,char c)
{ /* skip over next to c */


	ptr=eatwhite(ptr);
	if (ptr==NULL) return NULL;
	if (*ptr!=c) return NULL;
	ptr++;
	return ptr;


/*
    int res;
    while (*ptr!=c)
    {
        res=delimiter(*ptr);
        if (res==ENDLINE || res==COMMENT) return NULL;
        if (c==',' && *ptr=='(') return NULL;
        ptr++;
    }
    ptr++;
    return ptr;
*/
}

static SIGNED16 pack16(BYTE *b)
{ /* pack bytes into 16-bit type */
    return ((SIGNED16)b[1]<<8) || (SIGNED16)b[0];
}

static void unpack16(SIGNED16 a,BYTE *b)
{ /* unpack bytes from 16-bit type */
    b[0]=(BYTE)a;
    b[1]=(BYTE)(a>>8);
}

static WORD32 pack32(BYTE *b)
{ /* pack bytes into a 32-bit Word */
    return ((WORD32)b[3]<<24)|((WORD32)b[2]<<16)|((WORD32)b[1]<<8)|(WORD32)b[0];
}

static void unpack32(WORD32 a,BYTE *b)
{ /* unpack bytes from a word */
    b[0]=(BYTE)a;
    b[1]=(BYTE)(a>>8);
    b[2]=(BYTE)(a>>16);
    b[3]=(BYTE)(a>>24);
}

BOOL in_range(WORD32 num,WORD32 mask)
{ /* check that num will fit into the number of bits in mask */
    int n=num;
    if (n>=0 && (num&mask)!=num) return FALSE;
    if (n<0 && (num|mask)!=-1) return FALSE;
    return TRUE;
}


BOOL getcodesym(char *&ptr,WORD32 *m)
{
    return getsym(code_table,code_symptr,ptr,m);
}

BOOL getdatasym(char *&ptr,WORD32 *m)
{
    return getsym(data_table,data_symptr,ptr,m);
}

/* align data objects */

int alignment(int ptr,int num)
{
    int t,r=ptr;
    t=r%num;
    if (t>0) r+=(num-t);
    return r;
}

int instruction(char *start)
{
    int i=0;
    char text[10];    /* all instructions are 6 chars or less */
    char *ptr=start;
    if (ptr==NULL) return (-1); 
    while (i<10 && !delimiter(*ptr)) 
    {
        text[i++]=tolower(*ptr);
        ptr++;
    }
    if (i>9) return (-1);
    text[i]=0;    /* terminate it */

    for (i=0;;i++)
    {
        if (codes[i].name==NULL) return (-1);
        if (strcmp(text,codes[i].name)==0) break;
    }
    return i;
}

BOOL directive(int pass,char *ptr,char *line)
{ /* process assembler directives. return number of bytes consumed  */
    
    BOOL zero,bs;
    WORD32 num,m;
    WORD64 fw;
    DOUBLE64 db;
    int k,i,ch;
    int sc;
    BYTE b[10];
    char *iptr;

    if (ptr==NULL || *ptr!='.') return FALSE;
    
    for (k=0;;k++)
    {
        if (directives[k]==NULL) return FALSE;
        if (!compare(ptr,directives[k]))  continue;
        break;
    }
    while (!delimiter(*ptr)) ptr++;
    zero=TRUE;
    switch (k)
    {
    case 0:     /* .space */
        if (CODEORDATA==CODE) return FALSE;
        if (!getnum(ptr,&num)) return FALSE;
        if (num==0) return FALSE;
        if (CODEORDATA==DATA)
        {
            if (pass==2)
            {
                printword32(dataptr);
                printf("          %s",line);
            }
            dataptr+=num;
        //    if (dataptr>DATASIZE) return FALSE;
        }
        return TRUE;
        break;
    case 5:     /* .ascii  */
        zero=FALSE;
    case 1:     /* .asciiz */
        if (CODEORDATA==CODE) return FALSE;
        ptr=eatwhite(ptr);
        if (ptr==NULL) return (-1);
        if (*ptr!='"') return (-1);        
        sc=*ptr;    /* character to indicate end of string */
        *ptr++;
        num=0;
        iptr=ptr;
        bs=FALSE;
        while (*iptr!=sc)
        {
            if (delimiter(*iptr)==ENDLINE) return FALSE;
            if (bs)
            {
                num++;
                bs=FALSE;
            }
            else
            {
                if (*ptr=='\\') bs=TRUE;
                else num++;
            }

            iptr++; 
        }
        if (zero) num++;              /* trailing 0 needed */
        if (CODEORDATA==DATA)
        {
            if (pass==1) 
            {
                dataptr+=num;
                if (dataptr>DATASIZE) return FALSE;
            }
            if (pass==2)
            {
                printword32(dataptr);
                printf("          %s",line);            
                if (zero) *iptr=0;     /* stuff in a zero */
                m=0;
                printf("         ");
                bs=FALSE;
                while (m<num)
                {
                    if (m>0 && m%4==0) printf("\n         ");              
                    m++; 

                    if (bs)
                    {
                        if (*ptr=='n') ch='\n';
                        else ch=*ptr;
                        datamem[dataptr++]=ch;
                        printf("%02x",ch);
                        bs=FALSE;
                    }
                    else
                    {
                        if (*ptr=='\\') bs=TRUE;
                        else
                        {
                            datamem[dataptr++]=*ptr;
                            printf("%02x",*ptr);
                        }
                    } 
                    
                    ptr++;
                }
                printf("\n");
            }
        }  
        return TRUE;
    case 2:            /* .align */
        if (!getnum(ptr,&num)) return FALSE;
        if (num<2 || num>16) return FALSE;
        if (CODEORDATA==CODE) return FALSE;
        
        if (CODEORDATA==DATA)
        {
            dataptr=alignment(dataptr,num);
            if (pass==2)
            {
                printword32(dataptr);
                printf("          %s",line);
            }
        }
        
        return TRUE;
    case 3:            /* .word (This is the processor word length) */
        if (CODEORDATA==CODE) return FALSE;
        if (pass==1)
        {
            if (CODEORDATA==DATA)
            {
                do {
                    dataptr+=STEP;
                }
                while ((ptr=skipover(ptr,','))!=NULL);
                if (dataptr>DATASIZE) return FALSE;
            }
        }
        if (pass==2)
        {
            if (!getfullnum(ptr,&fw)) return FALSE;  
            if (CODEORDATA==DATA)
            {
                printword32(dataptr);
                printf(" ");
                printword(fw);
                printf(" %s",line);
                
                unpack(fw,b);       
                for (i=0;i<STEP;i++) datamem[dataptr++]=b[i];
                while ((ptr=skip(ptr,','))!=NULL)
                {
                    if (!getfullnum(ptr,&fw)) return FALSE;
                    printf("         ");
                    printword(fw);
                    printf("\n");
                    unpack(fw,b);
                    for (i=0;i<STEP;i++) datamem[dataptr++]=b[i];
                }
            }
        }
        return TRUE;

    case 12:            /* .double (64-bits) */
        if (CODEORDATA==CODE) return FALSE;
        if (pass==1)
        {
            if (CODEORDATA==DATA)
            {
                do {
                    dataptr+=STEP;
                }
                while ((ptr=skipover(ptr,','))!=NULL);
                if (dataptr>DATASIZE) return FALSE;
            }
        }
        if (pass==2)
        {
            if (!getdouble(ptr,&db.d)) return FALSE;  
            if (CODEORDATA==DATA)
            {
                printword32(dataptr);
                printf(" ");
                printdouble(db.d);
                printf(" %s",line);
                unpack(db.u,b);
                for (i=0;i<STEP;i++) datamem[dataptr++]=b[i];
                while ((ptr=skip(ptr,','))!=NULL)
                {
                    if (!getdouble(ptr,&db.d)) return FALSE;
                    printf("         ");
                    printdouble(db.d);
                    printf("\n");
                    unpack(db.u,b);
                    for (i=0;i<STEP;i++) datamem[dataptr++]=b[i];
                }
            }
        }
        return TRUE;
    case 10:            /* .word32 */
                        /* Note that all symbols are of this type */
        if (CODEORDATA==CODE) return FALSE;
        if (pass==1)
        {
            if (CODEORDATA==DATA)
            {
                do {
                   dataptr+=4;
                }
                while ((ptr=skipover(ptr,','))!=NULL);
                if (dataptr>DATASIZE) return FALSE;
            }
        }
        if (pass==2)
        {
            if (!getdatasym(ptr,&num)) return FALSE;
            if (CODEORDATA==DATA)
            {
                printword32(dataptr);
                printf(" %08x %s",num,line);
                unpack32(num,b);
                for (i=0;i<4;i++) datamem[dataptr++]=b[i];
                while ((ptr=skip(ptr,','))!=NULL)
                {
                    if (!getdatasym(ptr,&num)) return FALSE;
                    printf("         %08x\n",num);
                    unpack32(num,b);
                    for (i=0;i<4;i++) datamem[dataptr++]=b[i];
                }
            }
        }
        return TRUE;
    case 11:            /* .word16 */
        if (CODEORDATA==CODE) return FALSE;
        if (pass==1)
        {
            if (CODEORDATA==DATA)
            {
                do {
                   dataptr+=2;
                }
                while ((ptr=skipover(ptr,','))!=NULL);
                if (dataptr>DATASIZE) return FALSE;
            }
        }
        if (pass==2)
        {
            if (!getnum(ptr,&num)) return FALSE;
            if (!in_range(num,0xffff)) return FALSE;
            if (CODEORDATA==DATA)
            {
                printword32(dataptr);
                printf(" %04x     %s",num,line);
                unpack16((SIGNED16)num,b);
                for (i=0;i<2;i++) datamem[dataptr++]=b[i];
                while ((ptr=skip(ptr,','))!=NULL)
                {
                    if (!getnum(ptr,&num)) return FALSE;
                    if (!in_range(num,0xffff)) return FALSE;
                    printf("         %04x\n",num);
                    unpack16((SIGNED16)num,b);
                    for (i=0;i<2;i++) datamem[dataptr++]=b[i];
                }
            }
        }
        return TRUE;
    case 4:            /* .byte */
        if (CODEORDATA==CODE) return FALSE;
        if (pass==1)
        {
            if (CODEORDATA==DATA)
            {
                do {
                    dataptr+=1;
                }
                while ((ptr=skipover(ptr,','))!=NULL);
                if (dataptr>DATASIZE) return FALSE;
            }
        }
        if (pass==2)
        {
            if (!getnum(ptr,&num)) return FALSE;
            if (!in_range(num,0xff)) return FALSE;   
            if (CODEORDATA==DATA)
            {
                printword32(dataptr);
                printf(" %02x       %s",num,line);
                datamem[dataptr++]=(BYTE)num;
                while ((ptr=skip(ptr,','))!=NULL)
                {
                    if (!getnum(ptr,&num)) return FALSE;
                    if (!in_range(num,0xff)) return FALSE;   
                    printf("         %02x\n",num);
                    datamem[dataptr++]=(BYTE)num;
                }
            }
        }
        return TRUE;
    case 6:      /* .global */
    case 7:      /* .data   */
        CODEORDATA=DATA;
        if (pass==1)
        {  /* should be nothing after it */
            if (eatwhite(ptr)!=NULL) return FALSE;
        }
        if (pass==2)
        {
            printword32(dataptr);
            printf("          %s",line);
        }
        return TRUE;
    case 13:
    case 8:      /* .text or .code */
        CODEORDATA=CODE;
        if (pass==1)
        {
            if (eatwhite(ptr)!=NULL) return FALSE;
        }
        if (pass==2)
        {
            printword32(codeptr);
            printf("          %s",line);
        }
        return TRUE;

   case 9:    /* .org  */
        if (CODEORDATA==DATA)
        {
            if (!getnum(ptr,&num)) return FALSE;
            if (num<dataptr /* || num>DATASIZE */) return FALSE;
           
            dataptr=alignment(num,STEP);
            if (pass==2)
            {
                printword32(dataptr);
                printf("          %s",line);
            }  
            return TRUE;
        }
        if (CODEORDATA==CODE)
        {
            if (!getnum(ptr,&num)) return FALSE;
            if (num<codeptr || num>CODESIZE) return FALSE;
            codeptr=alignment(num,4);
            if (pass==2)
            {
                printword32(codeptr);
                printf("          %s",line);
            }
            return TRUE;
        }
        return FALSE;
    
    default:
        break;
    }
    return FALSE;
}

/* fill in symbol tables and check for syntax errors */

int first_pass(char *line,int lineptr)
{
    int i,len;
    char *ptr=line;

    ptr=eatwhite(ptr);
    if (ptr==NULL) return 0;
    if (delimiter(*ptr)) return 0;  /* blank line */

    while ((len=is_symbol(ptr))>0)
    {
      /*  copy symbol into symbol table
          advance over symbol           */
        
        if (CODEORDATA==0)
        {
            printf("Pass 1 - Error on line %d\n",lineptr);
            return 1;
        }
        if (CODEORDATA==CODE)
        {
            if (code_symptr>=SYMTABSIZE)
            {
                printf("Pass 1 - Error on line %d\n",lineptr);
                printf("Code symbol table overflow\n");
                return 1;
            } 
            code_table[code_symptr].symb=(char *)malloc(len+1);
            for (i=0;i<len;i++) code_table[code_symptr].symb[i]=*ptr++;
            code_table[code_symptr].symb[len]=0;
            code_table[code_symptr].value=codeptr;
            ptr++;           /* skip over the ":"  */
            code_symptr++;
        }
        if (CODEORDATA==DATA)
        {
            if (data_symptr>=SYMTABSIZE)
            {
                printf("Pass 1 - Error on line %d\n",lineptr);
                printf("Data symbol table overflow\n");
                return 1;
            } 
            data_table[data_symptr].symb=(char *)malloc(len+1);
            for (i=0;i<len;i++) data_table[data_symptr].symb[i]=*ptr++;
            data_table[data_symptr].symb[len]=0;
            data_table[data_symptr].value=dataptr;
            ptr++;           /* skip over the ":"  */
            data_symptr++;
        }
        ptr=eatwhite(ptr);
        if (ptr==NULL) return 0;
    }
    if (instruction(ptr)>=0)
    {
        /* instruction found             */
        /* increase instruction pointer  */
        if (CODEORDATA!=CODE)
        {
            printf("Pass 1 - Error on line %d\n",lineptr);
            return 1;
        }
        codeptr=alignment(codeptr,4);
        codeptr+=4;
        if (codeptr>CODESIZE)
        {
            printf("Pass 1 - Error on line %d\n",lineptr);
            printf("No such memory location\n"); 
            return 1;
        }
        return 0;
    }
    if (directive(1,ptr,line)) return 0;

    printf("Pass 1 - Error on line %d\n",lineptr);
    
    return 1;
}

int second_pass(char *line,int lineptr)
{
    WORD32 w,byte;
    WORD32 op,code_word=0;
    WORD32 flags;
    BYTE b[4];
    int i,instruct;
    int rs,rt,rd,sub,type;
    BOOL sign,error=TRUE;
    
    char *ptr=line;
    ptr=eatwhite(ptr);
    if (ptr==NULL) 
    {
        printf("                  %s",line);
        return 0;  /* blank line */
    }

/* skip over any symbols on the line */
    while (is_symbol(ptr))
    {
        while (*ptr!=':') ptr++;
        ptr++;
        ptr=eatwhite(ptr);
        if (ptr==NULL) break;
    } 

    instruct=instruction(ptr);
    if (instruct<0)
    { /* no instruction on the line, perhaps a directive? */
        
        if (!directive(2,ptr,line))
        { /* its not a good directive - so just copy it out */
            if (CODEORDATA==CODE)
            {
                printword32(codeptr);
                printf("          %s",line);
            }
            if (CODEORDATA==DATA) 
            {
        
                if (ptr==NULL)
                {                
                    printword32(dataptr);
                    printf("          %s",line);
                }
                else
                {
                    printf(" ????????          %s",line);
                    return 1;
                }
            }
        }
        return 0;
    }

/* instruction found - parse it out */

    op=codes[instruct].op_code;
    sub=codes[instruct].subtype;
    type=codes[instruct].type;
    while (!delimiter(*ptr)) ptr++; /* advance over instuction */

    rs=rt=rd=0;
    w=0; byte=0; flags=0;
    sign=TRUE;
    switch (sub)
    {
  case NOP:
  case HALT:
        ptr=eatwhite(ptr); if(ptr!=NULL) break;
        error=FALSE;
        break;    
   
  case STORE:
  case LOAD:
        rt=getreg(ptr);    if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over ,  */
        ptr=eatwhite(ptr);
        if (*ptr=='(') w=0;
        else if (!getdatasym(ptr,&w)) break;
        ptr=skip(ptr,'('); if (eatwhite(ptr)==NULL) break;  /* skip over (  */
        ptr=eatwhite(ptr);
        rs=getreg(ptr);    if (rs<0) break;
        ptr=skip(ptr,')');                                  /* skip over )  */
        if (ptr==NULL) break;
        if (eatwhite(ptr)!=NULL) break;
        error=FALSE;
        break;    
  case FSTORE:
  case FLOAD:
        rt=fgetreg(ptr);    if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        if (*ptr=='(') w=0;
        else if (!getdatasym(ptr,&w)) break;
        ptr=skip(ptr,'('); if (eatwhite(ptr)==NULL) break;  /* skip over ( */
        ptr=eatwhite(ptr);
        rs=getreg(ptr);    if (rs<0) break;
        ptr=skip(ptr,')');                                  /* skip over ) */
        if (ptr==NULL) break;
        if (eatwhite(ptr)!=NULL) break;
        error=FALSE;
        break;    

  case REG2I:
        rt=getreg(ptr); if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        rs=getreg(ptr); if (rs<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        if (!getdatasym(ptr,&w)) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;    

  case REG1I:
        rt=getreg(ptr); if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        if (!getdatasym(ptr,&w)) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;    

  case JREG:
        rt=getreg(ptr); if (rt<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;    

  case REG3:
  case REG3X:
        rd=getreg(ptr); if (rd<0) break; 
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over ,  */
        ptr=eatwhite(ptr);
        rs=getreg(ptr); if (rs<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over ,  */
        ptr=eatwhite(ptr);
        rt=getreg(ptr); if (rt<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  case REG3F:
        rd=fgetreg(ptr); if (rd<0) break; 
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over ,  */
        ptr=eatwhite(ptr);
        rs=fgetreg(ptr); if (rs<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over ,  */
        ptr=eatwhite(ptr);
        rt=fgetreg(ptr); if (rt<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  case REG2F:
        rd=fgetreg(ptr); if (rd<0) break; 
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        rs=fgetreg(ptr); if (rs<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  case REG2C:
        rs=fgetreg(ptr); if (rs<0) break; 
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        rt=fgetreg(ptr); if (rt<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  case REGID:
  case REGDI:
        rt=getreg(ptr); if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        rd=fgetreg(ptr); if (rd<0) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;
   
  case REG2S:
        rd=getreg(ptr); if (rd<0) break; 
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        rs=getreg(ptr); if (rs<0) break;
        ptr=skip(ptr,','); if (ptr==NULL) break;  /* skip over , */
        ptr=eatwhite(ptr);
        if (!getdatasym(ptr,&flags)) break;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;
       
  case JUMP:
  case BC:
        if (!getcodesym(ptr,&w)) break;
        w-=(codeptr+4);   /* relative jump */
        w=(SIGNED32)w/4;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;    

  case BRANCH:
        rt=getreg(ptr); if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;
        ptr=eatwhite(ptr);
        rs=getreg(ptr); if (rs<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;
        ptr=eatwhite(ptr);
        if (!getcodesym(ptr,&w)) break;    

        w-=(codeptr+4);
        w=(SIGNED32)w/4;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  case JREGN:
        rt=getreg(ptr); if (rt<0) break;
        ptr=skip(ptr,','); if (eatwhite(ptr)==NULL) break;
        ptr=eatwhite(ptr);
        if (!getcodesym(ptr,&w)) break;    
        w-=(codeptr+4);
        w=(SIGNED32)w/4;
        ptr=eatwhite(ptr); if (ptr!=NULL) break;
        error=FALSE;
        break;

  default: 
        error=FALSE;
        break;
    }

    if (!error) switch (type)
    {
   case I_TYPE:
        if (in_range(w,0xffff)) 
            code_word=(op | rs<<21 | rt<<16 | (w&0xffff));
        else error=TRUE;   
        break;
   case R_TYPE:
        if (in_range(flags,0x1F))
            code_word=(op | rs<<21 | rt << 16 | rd<<11 | flags<<6);
        else error=TRUE;
        break;
   case J_TYPE:
        if (in_range(w,0x3ffffff)) code_word=(op | w&0x3ffffff);
        else error=TRUE;
        break;
   case F_TYPE:
        code_word=(op | rs << 11 | rt << 16 | rd << 6);
        break;
   case M_TYPE:
        code_word=(op | rt << 16 | rd << 11);
        break;
   case B_TYPE:
        if (in_range(w,0xffff)) code_word=(op | w&0xffff);
        else error=TRUE;
        break;
    }

    codeptr=alignment(codeptr,4);
    if (error) 
    {
        printword32(codeptr);
        printf(" ???????? %s",line);
        for (i=0;i<4;i++) codemem[codeptr++]=0;
        return 1;     
    }

    printword32(codeptr);
    printf(" %08x %s",code_word,line);
    unpack32(code_word,b);
    for (i=0;i<4;i++) codemem[codeptr++]=b[i];
    return 0;    
}

void strip(char *name)
{ /* strip off filename extension */
    int i;
    for (i=0;name[i]!='\0';i++)
    {
        if (name[i]!='.') continue;
        name[i]='\0';
        break;
    }
}

int mygets(char *line,int max,FILE *fp)
{
    int i,ch;
    i=0;
    do
    {
        ch=fgetc(fp);
        if (i==0 && ch==EOF) return 1;
        if (ch==EOF) break;
        if (i<max-3) line[i++]=ch;
        else 
        {
            line[i++]='\n';
            line[i]='\0';
            do
            {
                ch=fgetc(fp);
            } while (ch!=EOF && ch!='\n' && ch!='\0');
            
            return -1;
        }
    } while (ch!='\n' && ch!='\0');
    if (ch!='\n') line[i++]='\n';
    line[i]='\0';
    return 0;
}

#define MAX_LINE 200
char line[MAX_LINE+1];

int main(int argc,char **argv)
{
    FILE *fp,*fc,*fd;
    int j,lineptr,errors,gotline;
    WORD32 i;
    char fname[MAX_LINE+1];
    argc--; argv++;

    if (argc!=1)
    {
        printf("Incorrect usage\n");
        printf("asm <file>\n");
        printf("and <file> is the input file, usually with .s extension\n");
        printf("Output files have the same name and .cod and .dat extensions\n");
        exit(0);
    }

    NBITS=64;
    MASK=-1;

    STEP=8;

    fp=fopen(argv[0],"r");
    if (fp==NULL)
    {
        printf("No such file\n");
        return 0;
    }
    
/* First Pass */
    errors=0;
    CODEORDATA=0;
    codeptr=0;
    dataptr=0;
    code_symptr=0;
    data_symptr=0;
    for (lineptr=1;;lineptr++)
    {
/* printf("line= %d\n",lineptr);  */
        gotline=mygets(line,MAX_LINE,fp); 
        if (gotline==1) break;    // end of file
        if (gotline==-1) 
        { // something got chopped
            errors++;
            printf("Pass 1 - Error on line %d\n",lineptr);
            printf("Line too long\n");
        }
        errors+=first_pass(line,lineptr);
        dataptr=alignment(dataptr,STEP);
    }
    fclose(fp);
    printf("Pass 1 completed with %d errors\n",errors);
    if (errors) return 0;
    
    fp=fopen(argv[0],"r");
    errors=0;
    CODEORDATA=0;
    codeptr=0;
    dataptr=0;
    for (lineptr=1;;lineptr++)
    {
        gotline=mygets(line,MAX_LINE,fp); 
        if (gotline==1) break;    // end of file
        if (gotline==-1)
        { // something got chopped
            errors++;
            printf("Pass 2 - Error on line %d\n",lineptr);
            printf("Line too long\n");
        }    
        errors+=second_pass(line,lineptr);
        dataptr=alignment(dataptr,STEP);
    }
    fclose(fp);
    printf("\nPass 2 completed with %d errors\n",errors);
    if (errors) return 0;

    printf("Code Symbol Table\n");
    for (j=0;j<code_symptr;j++)
    {
        printf("%20s = ",code_table[j].symb);
        printword32(code_table[j].value);
        printf("\n");
    }

    printf("Data Symbol Table\n");
    for (j=0;j<data_symptr;j++)
    {
        printf("%20s = ",data_table[j].symb);
        printword32(data_table[j].value);
        printf("\n");
    }

    strcpy(fname,argv[0]);
    strip(fname);
    strcat(fname,".cod");
    fc=fopen(fname,"w");

    for (i=0;i<codeptr;i+=4)
        fprintf(fc,"%08x\n",pack32(&codemem[i]));

    fclose(fc);

    strip(fname);
    strcat(fname,".dat");
    fd=fopen(fname,"w");
    for (i=0;i<dataptr;i+=STEP)
    {
        for (j=0;j<STEP;j++) fprintf(fd,"%02x",datamem[i+7-j]);
        fprintf(fd,"\n");
    }
    fclose(fd);

    return 0;
}


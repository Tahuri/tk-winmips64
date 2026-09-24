// ORACLE: non-UI subset of CWinMIPS64Doc (third_party/winmips64-andoni/src/WinMIPS64Doc.h).
// Member names and types are kept identical to the original so that doc.cpp can
// stay a near-verbatim copy of WinMIPS64Doc.cpp.
#pragma once

#include <vector>

#include "stdafx.h"
#include "utils.h"

#define MIN_CODEBITS 8
#define MAX_CODEBITS 13
#define MIN_DATABITS 4
#define MAX_DATABITS 11
#define MIN_ADD_LATENCY 2
#define MAX_ADD_LATENCY 8
#define MIN_MUL_LATENCY 2
#define MAX_MUL_LATENCY 8
#define MIN_DIV_LATENCY 10
#define MAX_DIV_LATENCY 30

#define SYMTABSIZE 1000

#define GSXY 50 // from IOView.h

// Minimal CFile / CStdioFile replacement. CStdioFile::Open without typeBinary
// opens the stream in *text* mode on Windows: "\r\n" is read as "\n" and a
// Ctrl-Z (0x1A) byte acts as end of file. Read() emulates exactly that.
class CFile
{
public:
	enum { modeRead = 0 };
	CFile() : fp(NULL), pending(-1), eof(false) {}
	~CFile() { Close(); }
	BOOL Open(const CString& name, int /*mode*/)
	{
		Close();
		fp = fopen(name.c_str(), "rb");
		pending = -1;
		eof = false;
		return fp != NULL;
	}
	void Close()
	{
		if (fp) fclose(fp);
		fp = NULL;
	}
	unsigned int Read(void* buf, unsigned int n);

private:
	FILE* fp;
	int pending;
	bool eof;
};
typedef CFile CStdioFile;

struct AsmError
{
	int line;
	std::string text;
};

struct OracleConfig
{
	int codebits = 10, databits = 10;
	int add = 4, mul = 7, div = 24;
	BOOL forwarding = TRUE, delay_slot = FALSE, btb = FALSE;
};

class CWinMIPS64Doc
{
public:
	explicit CWinMIPS64Doc(const OracleConfig& cfg);
	~CWinMIPS64Doc();

	// Attributes (as in the original)
	CString* codelines;
	CString* datalines;
	CString* assembly;
	CString* mnemonic;

	unsigned int CODESIZE;
	unsigned int DATASIZE;

	processor cpu;
	pipeline pipe;

	BOOL forwarding;
	BOOL delay_slot;
	BOOL branch_target_buffer;
	BOOL registers_as_numbers;

	symbol_table code_table[SYMTABSIZE];
	symbol_table data_table[SYMTABSIZE];
	unsigned int codeptr;
	unsigned int dataptr;

	unsigned int code_symptr;
	unsigned int data_symptr;
	int CODEORDATA;
	unsigned int cycles;
	unsigned int instructions;
	unsigned int loads;
	unsigned int stores;
	unsigned int branch_taken_stalls;
	unsigned int branch_misprediction_stalls;
	unsigned int raw_stalls;
	unsigned int waw_stalls;
	unsigned int war_stalls;
	unsigned int structural_stalls;

	int multi;
	unsigned int ADD_LATENCY;
	unsigned int MUL_LATENCY;
	unsigned int DIV_LATENCY;

	BOOL simulation_running;
	BOOL restart;
	int stall_type;
	int stalls;
	int amount;

	record history[50];
	WORD32 entries;
	WORD32 offset;

	// ORACLE additions
	std::string last_msg;          // status-bar text built by process_result (English canonical)
	RESULT last_result;            // RESULT after update_history of the last cycle
	std::vector<AsmError> asm_errors;

	int mygets(char*, int, CFile*);
	BOOL openit(CString);
	int openfile(CString);
	int one_cycle(pipeline*, processor*, BOOL);
	void OnFileReset();
	void OnFullReset();
	void keyboard_input(const std::string& text); // IOView::OnChar emulation

protected:
	BOOL getcodesym(char*&, WORD32*);
	BOOL getdatasym(char*&, WORD32*);
	int instruction(char*);
	BOOL directive(int, char*, char*);
	int first_pass(char*, int);
	int second_pass(char*, int);
	void process_result(RESULT*, BOOL);
	void clear();
	void check_stalls(int, const char*, int, char*);
	void update_history(pipeline*, processor*, RESULT*);
	int update_io(processor*);
	void UpdateAllViews(void*, long = 0) {} // UI refresh: no-op
	void asm_error(int lineptr, const char* line);
	CString io_line; // CIOView::line (characters typed so far)
};

/*
 * Copyright 2026 tk-winmips64 contributors
 * SPDX-License-Identifier: Apache-2.0
 *
 * Derived from WinMIPS64 by Mike Scott (Apache-2.0,
 * https://github.com/mcarrickscott/WinMIPS64) and the fork by Andoni
 * Zubimendi (https://github.com/AndoniZubimendi/WinMIPS64).
 * Modified: adapted to build outside Windows/MFC as a trace oracle. See NOTICE.
 */

// ORACLE driver: assembles a program with the original WinMIPS64 assembler,
// runs it cycle by cycle with the original pipeline, and prints the golden
// trace defined in docs/CONTRACT.md section 1.
#include <string>
#include <vector>
#include <fstream>

#include "doc.h"

static void usage()
{
	fprintf(stderr,
		"usage: oracle <prog.s> [--codebits N] [--databits N] [--add N] [--mul N] [--div N]\n"
		"                       [--forwarding 0|1] [--delayslot 0|1] [--btb 0|1]\n"
		"                       [--input FILE] [--max N]\n");
}

static bool parse_int(const char* s, int* out)
{
	char* e;
	long v = strtol(s, &e, 10);
	if (*s == 0 || *e != 0) return false;
	*out = (int)v;
	return true;
}

static uint32_t fnv1a(const unsigned char* p, size_t n)
{
	uint32_t h = 2166136261u;
	for (size_t i = 0; i < n; i++)
	{
		h ^= p[i];
		h *= 16777619u;
	}
	return h;
}

static std::string escape(const std::string& s)
{
	std::string r;
	char buf[8];
	for (size_t i = 0; i < s.size(); i++)
	{
		unsigned char c = (unsigned char)s[i];
		switch (c)
		{
		case '\n': r += "\\n"; break;
		case '\t': r += "\\t"; break;
		case '\\': r += "\\\\"; break;
		case '"': r += "\\\""; break;
		default:
			if (c < 0x20 || c >= 0x7f)
			{
				snprintf(buf, sizeof(buf), "\\x%02x", c);
				r += buf;
			}
			else r += (char)c;
		}
	}
	return r;
}

static void hexbytes(FILE* o, const char* tag, const BYTE* p, unsigned n)
{
	fputs(tag, o);
	fputc(' ', o);
	for (unsigned i = 0; i < n; i++) fprintf(o, "%02x", p[i]);
	fputc('\n', o);
}

static void print_regs(FILE* o, const char* tag, const reg* r)
{
	fputs(tag, o);
	for (int i = 0; i < 64; i++)
		fprintf(o, " %016" PRIx64 ":%d", (uint64_t)r[i].val, (int)r[i].source);
	fputc('\n', o);
}

static void print_cycle(FILE* o, CWinMIPS64Doc& d, int ret)
{
	processor& cpu = d.cpu;
	pipeline& pipe = d.pipe;
	const RESULT& x = d.last_result;
	int i;

	fprintf(o, "C %u RET %d CPU %d PC %08x\n", d.cycles, ret, cpu.status, cpu.PC);
	print_regs(o, "R", cpu.rreg);
	print_regs(o, "W", cpu.wreg);
	fprintf(o, "FCC %d\n", cpu.fp_cc);

	fprintf(o, "P IF %d,%08x ID %d,%08x EX %d,%08x A", pipe.active, cpu.PC,
		pipe.if_id.active, pipe.if_id.IR, pipe.integer.active, pipe.integer.IR);
	for (i = 0; i < pipe.ADD_LATENCY; i++)
		fprintf(o, "%c%d,%08x", i ? ';' : ' ', pipe.a[i].active, pipe.a[i].IR);
	fprintf(o, " M");
	for (i = 0; i < pipe.MUL_LATENCY; i++)
		fprintf(o, "%c%d,%08x", i ? ';' : ' ', pipe.m[i].active, pipe.m[i].IR);
	fprintf(o, " DIV %d,%08x MEM %d,%08x WB %d,%08x\n", pipe.div.active, pipe.div.IR,
		pipe.ex_mem.active, pipe.ex_mem.IR, pipe.mem_wb.active, pipe.mem_wb.IR);

	fprintf(o, "X %d %d %d %d %d %d A", x.IF, x.ID, x.EX, x.MEM, x.WB, x.DIVIDER);
	for (i = 0; i < pipe.ADD_LATENCY; i++) fprintf(o, " %d", x.ADDER[i]);
	fprintf(o, " M");
	for (i = 0; i < pipe.MUL_LATENCY; i++) fprintf(o, " %d", x.MULTIPLIER[i]);
	fprintf(o, " RR %d %d %d %d %d %d\n", x.idrr, x.exrr, x.memrr, x.addrr, x.mulrr, x.divrr);

	fprintf(o, "S %u %u %u %u %u %u %u %u %u\n", d.instructions, d.loads, d.stores,
		d.raw_stalls, d.waw_stalls, d.war_stalls, d.structural_stalls,
		d.branch_taken_stalls, d.branch_misprediction_stalls);
	fprintf(o, "MSG %s\n", d.last_msg.c_str());
	fprintf(o, "T %s\n", escape(cpu.Terminal).c_str());

	unsigned char scr[GSXY * GSXY * 4];
	for (i = 0; i < GSXY * GSXY; i++)
	{
		WORD32 v = cpu.screen[i];
		scr[4 * i] = (unsigned char)v;
		scr[4 * i + 1] = (unsigned char)(v >> 8);
		scr[4 * i + 2] = (unsigned char)(v >> 16);
		scr[4 * i + 3] = (unsigned char)(v >> 24);
	}
	fprintf(o, "MEMH %08x %08x %08x\n", fnv1a(cpu.data, cpu.datasize),
		fnv1a(cpu.dstat, cpu.datasize), fnv1a(scr, sizeof(scr)));

	fprintf(o, "H %u", d.entries);
	for (WORD32 e = 0; e < d.entries; e++)
	{
		const record& h = d.history[e];
		unsigned cc = d.cycles - h.start_cycle;
		fprintf(o, " %08x@%u:", h.IR, h.start_cycle);
		for (unsigned k = 0; k <= cc && k < 500; k++)
			fprintf(o, "%s%u.%u.%u", k ? "/" : "", h.status[k].stage, h.status[k].substage, h.status[k].cause);
	}
	fputc('\n', o);
}

int main(int argc, char** argv)
{
	OracleConfig cfg;
	const char* prog = NULL;
	const char* input_file = NULL;
	int maxc = 5000;

	for (int a = 1; a < argc; a++)
	{
		std::string f = argv[a];
		if (f.rfind("--", 0) != 0)
		{
			if (prog) { usage(); return 2; }
			prog = argv[a];
			continue;
		}
		if (a + 1 >= argc) { usage(); return 2; }
		const char* v = argv[++a];
		int n = 0;
		if (f == "--input") { input_file = v; continue; }
		if (!parse_int(v, &n)) { fprintf(stderr, "oracle: bad value for %s: %s\n", f.c_str(), v); return 2; }
		if (f == "--codebits") cfg.codebits = n;
		else if (f == "--databits") cfg.databits = n;
		else if (f == "--add") cfg.add = n;
		else if (f == "--mul") cfg.mul = n;
		else if (f == "--div") cfg.div = n;
		else if (f == "--forwarding") cfg.forwarding = n ? TRUE : FALSE;
		else if (f == "--delayslot") cfg.delay_slot = n ? TRUE : FALSE;
		else if (f == "--btb") cfg.btb = n ? TRUE : FALSE;
		else if (f == "--max") maxc = n;
		else { fprintf(stderr, "oracle: unknown flag %s\n", f.c_str()); usage(); return 2; }
	}
	if (!prog) { usage(); return 2; }

	if (cfg.codebits < MIN_CODEBITS || cfg.codebits > MAX_CODEBITS ||
		cfg.databits < MIN_DATABITS || cfg.databits > MAX_DATABITS ||
		cfg.add < MIN_ADD_LATENCY || cfg.add > MAX_ADD_LATENCY ||
		cfg.mul < MIN_MUL_LATENCY || cfg.mul > MAX_MUL_LATENCY ||
		cfg.div < MIN_DIV_LATENCY || cfg.div > MAX_DIV_LATENCY || maxc < 0)
	{
		fprintf(stderr, "oracle: configuration value out of range\n");
		return 2;
	}
	if (cfg.btb && cfg.delay_slot)
	{
		fprintf(stderr, "oracle: --btb 1 is incompatible with --delayslot 1\n");
		return 2;
	}

	std::vector<std::string> input;
	if (input_file)
	{
		std::ifstream in(input_file, std::ios::binary);
		if (!in) { fprintf(stderr, "oracle: cannot open input file %s\n", input_file); return 2; }
		std::string l;
		while (std::getline(in, l))
		{
			if (!l.empty() && l.back() == '\r') l.pop_back();
			input.push_back(l);
		}
	}

	CWinMIPS64Doc* doc = new CWinMIPS64Doc(cfg);
	CWinMIPS64Doc& d = *doc;
	FILE* o = stdout;

	// openfile(): OnFileReset + memory clear + openit (as File/Open in the GUI)
	int res = d.openfile(prog);
	if (res == 1)
	{
		fprintf(stderr, "oracle: cannot open %s\n", prog);
		return 1;
	}
	if (res != 0)
	{
		fprintf(o, "ASM ERR\n");
		for (size_t k = 0; k < d.asm_errors.size(); k++)
			fprintf(o, "E %d %s\n", d.asm_errors[k].line, d.asm_errors[k].text.c_str());
		delete doc;
		return 0;
	}

	fprintf(o, "ASM OK\n");
	hexbytes(o, "CODE", d.cpu.code, d.cpu.codesize);
	hexbytes(o, "CSTAT", d.cpu.cstat, d.cpu.codesize);
	hexbytes(o, "DATA", d.cpu.data, d.cpu.datasize);
	hexbytes(o, "DSTAT", d.cpu.dstat, d.cpu.datasize);
	for (unsigned k = 0; k < d.CODESIZE / 4; k++) fprintf(o, "CL %u %s\n", k, d.codelines[k].c_str());
	for (unsigned k = 0; k < d.DATASIZE / 8; k++) fprintf(o, "DL %u %s\n", k, d.datalines[k].c_str());

	size_t next_input = 0;
	while (d.cpu.status != HALTED && d.cycles < (unsigned)maxc)
	{
		int ret = d.one_cycle(&d.pipe, &d.cpu, FALSE);
		print_cycle(o, d, ret);
		if (ret == WAITING_FOR_INPUT)
		{
			if (next_input >= input.size())
			{
				fprintf(o, "INPUT EOF\n");
				break;
			}
			d.keyboard_input(input[next_input++]);
		}
	}
	fprintf(o, "END %u %u\n", d.cycles, d.instructions);
	delete doc;
	return 0;
}

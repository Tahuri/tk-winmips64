/*
 * Copyright 2026 tk-winmips64 contributors
 * SPDX-License-Identifier: Apache-2.0
 *
 * Derived from WinMIPS64 by Mike Scott (Apache-2.0,
 * https://github.com/mcarrickscott/WinMIPS64) and the fork by Andoni
 * Zubimendi (https://github.com/AndoniZubimendi/WinMIPS64).
 * Modified: adapted to build outside Windows/MFC as a trace oracle. See NOTICE.
 */

// ORACLE shim: replaces the MFC precompiled header (StdAfx.h) of WinMIPS64.
//
// It provides just enough of the Win32/MFC/MSVC-CRT surface for the upstream
// pipeline.cpp and utils.cpp (compiled verbatim) and for tools/oracle/src/doc.cpp.
// Every place where the behaviour of the Windows CRT differs from the host libc in
// a way the simulator can observe is emulated here and listed in README.md.
#pragma once

// All standard headers are included BEFORE mytypes.h, because mytypes.h defines
// short macros (READ, WRITE, ADD, DIV, OK, EMPTY, ...) that would break them.
#include <cctype>
#include <cerrno>
#include <cinttypes>
#include <cmath>
#include <cstdarg>
#include <cstdint>
#include <cstdio>
#include <cstdlib>
#include <cstring>
#include <string>
#include <strings.h>

// ---------------------------------------------------------------- Win32 types
typedef uint32_t DWORD32;
#ifndef MAX_PATH
#define MAX_PATH 260
#endif
#define RGB(r, g, b) \
	((uint32_t)(((uint8_t)(r) | ((uint32_t)((uint8_t)(g)) << 8)) | (((uint32_t)(uint8_t)(b)) << 16)))

// ------------------------------------------------------------- MSVC "secure" CRT
#define sprintf_s snprintf

inline int strcpy_s(char* dst, size_t n, const char* src)
{
	size_t l = strlen(src);
	if (l >= n) { if (n) dst[0] = 0; return ERANGE; }
	memcpy(dst, src, l + 1);
	return 0;
}

inline int strcat_s(char* dst, size_t n, const char* src)
{
	size_t d = strnlen(dst, n);
	size_t l = strlen(src);
	if (d + l >= n)
	{
		// The MSVC CRT would call the invalid-parameter handler (terminates the
		// process). The oracle aborts loudly instead of silently truncating.
		fprintf(stderr, "oracle: strcat_s overflow (%zu+%zu >= %zu)\n", d, l, n);
		abort();
	}
	memcpy(dst + d, src, l + 1);
	return 0;
}

inline int _strnicmp(const char* a, const char* b, size_t n) { return strncasecmp(a, b, n); }

// ------------------------------------------------------------------ __int64
// Only one use of __int64 survives in the upstream sources once mytypes.h is
// patched: `fpR.s = (__int64)fpA.d;` (CVT.L.D in pipeline.cpp). The original
// runs on x86/x64 where the conversion is CVTTSD2SI: NaN and out-of-range values
// give 0x8000000000000000 ("integer indefinite"). arm64 saturates instead, so the
// cast is routed through this type to reproduce the x86 result on every host.
struct oracle_x86_int64
{
	int64_t v;
	oracle_x86_int64(double d)
	{
		if (std::isnan(d) || d >= 9223372036854775808.0 || d < -9223372036854775808.0)
			v = INT64_MIN;
		else
			v = (int64_t)d;
	}
	operator int64_t() const { return v; }
};
#define __int64 oracle_x86_int64

// ----------------------------------------------------------------- strtoul
// On Windows `unsigned long` is 32 bits, so strtoul() saturates at 0xFFFFFFFF
// (ERANGE) for any magnitude that does not fit in 32 bits. A leading '-' negates
// the (32-bit) value, as in the C standard.
inline unsigned long oracle_win32_strtoul(const char* s, char** end, int base)
{
	const char* p = s;
	while (isspace((unsigned char)*p)) p++;
	bool neg = false;
	if (*p == '-') { neg = true; p++; }
	else if (*p == '+') p++;
	if (*p == '-' || *p == '+')
	{ // strtoull would accept a second sign; Windows strtoul does not
		if (end) *end = (char*)s;
		return 0;
	}
	char* e;
	errno = 0;
	unsigned long long v = strtoull(p, &e, base);
	if (e == p)
	{ // no conversion
		if (end) *end = (char*)s;
		return 0;
	}
	if (end) *end = e;
	if (errno == ERANGE || v > 0xFFFFFFFFULL) return 0xFFFFFFFFUL;
	uint32_t r = (uint32_t)v;
	if (neg) r = (uint32_t)(0u - r);
	return r;
}
#define strtoul oracle_win32_strtoul

// ------------------------------------------------------------------- ctype
// The assembler passes plain (signed) char to isalnum/isdigit/tolower. For
// bytes >= 0x80 that is a negative int, which is undefined behaviour and
// differs between CRTs. The oracle pins the "C" locale ASCII classification:
// only 0..127 can be alnum/digit/upper; everything else is "not a letter".
inline int oracle_isdigit(int c) { return c >= '0' && c <= '9'; }
inline int oracle_isalpha(int c) { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z'); }
inline int oracle_isalnum(int c) { return oracle_isdigit(c) || oracle_isalpha(c); }
inline int oracle_tolower(int c) { return (c >= 'A' && c <= 'Z') ? c + ('a' - 'A') : c; }
#define isdigit(c) oracle_isdigit(c)
#define isalnum(c) oracle_isalnum(c)
#define tolower(c) oracle_tolower(c)

// ----------------------------------------------------------------- CString
// Minimal stand-in for MFC CString (ANSI build).
class CString : public std::string
{
public:
	CString() {}
	CString(const char* s) : std::string(s ? s : "") {}
	CString(const char* s, int n) : std::string(s, (size_t)n) {}
	CString(const std::string& s) : std::string(s) {}
	CString& operator=(const char* s) { assign(s ? s : ""); return *this; }
	CString& operator=(const std::string& s) { assign(s); return *this; }
	int GetLength() const { return (int)size(); }
	void Delete(int i, int n = 1) { if (i >= 0 && (size_t)i < size()) erase((size_t)i, (size_t)n); }
	int Find(char c, int start = 0) const
	{
		size_t r = find(c, (size_t)start);
		return r == npos ? -1 : (int)r;
	}
	void MakeLower()
	{
		for (size_t i = 0; i < size(); i++) (*this)[i] = (char)oracle_tolower((*this)[i]);
	}
	const char* GetString() const { return c_str(); }
};

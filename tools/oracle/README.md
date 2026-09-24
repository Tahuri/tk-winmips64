# WinMIPS64 reference oracle

`oracle` is the original WinMIPS64 v1.60 simulator (`third_party/winmips64-andoni/src`), built as a
command-line program for macOS and Linux. It prints the golden trace that `docs/CONTRACT.md` §1
defines. The Go port (`wmips trace`) must produce the same bytes.

## Build and run

```sh
make -C tools/oracle            # output: tools/oracle/build/oracle
make -C tools/oracle check      # upstream files are verbatim + I/O self-test
tools/oracle/build/oracle testdata/programs/simple.s --btb 1 --max 5000
tools/oracle/gen-golden.sh      # regenerate testdata/golden/ (about 2 minutes)
```

Requirements: clang >= 16 or gcc >= 12 (for `-ftrivial-auto-var-init=zero`). Tested with Apple
clang 21 (darwin/arm64) and gcc 13 (Linux, Docker `gcc:13`). The two builds give byte-identical
traces for all 195 golden files.

## Layout

| Path | Content |
|---|---|
| `shim/stdafx.h` | Replaces the MFC precompiled header: `CString`, `RGB`, `DWORD32`, `sprintf_s`, `strcpy_s`, `strcat_s`, `_strnicmp`, and emulations of Windows CRT behaviour (see below). |
| `shim/mytypes.h` | Copy of upstream `mytypes.h`. Only the four integer typedefs are patched (see below). |
| `build/upstream/` | `pipeline.cpp`, `utils.cpp`, `utils.h`: **verbatim** copies that the Makefile makes. `make verify-upstream` proves it with `cmp`. |
| `src/doc.h`, `src/doc.cpp` | Non-UI methods of `CWinMIPS64Doc`, copied from `WinMIPS64Doc.cpp`. Each change has an `ORACLE:` comment. |
| `src/main.cpp` | Command-line flags and trace printer. |
| `gen-golden.sh` | Regenerates `testdata/golden/`. |
| `tests/io.s`, `tests/io.input`, `tests/io.default.trace` | Self-test for memory-mapped I/O and keyboard input. No program in `testdata/programs` uses I/O. |

The Makefile copies the upstream files to `build/upstream/` before it compiles them. This is
necessary because `#include "..."` looks first in the directory of the including file. From
`third_party/`, the includes would find the MFC `StdAfx.h` and the unpatched `mytypes.h`.

## Changes to upstream code

### `mytypes.h` (the only patched upstream header)

| Original | Oracle | Reason |
|---|---|---|
| `typedef unsigned long WORD32;` | `uint32_t` | `long` has 32 bits on Windows (LLP64) and 64 bits on macOS/Linux (LP64). |
| `typedef long SIGNED32;` | `int32_t` | Same reason. |
| `typedef unsigned __int64 WORD64;` | `uint64_t` | `__int64` is an MSVC keyword. |
| `typedef __int64 SIGNED64;` | `int64_t` | Same reason. |

`pipeline.cpp`, `utils.cpp` and `utils.h` have no changes.

### Windows CRT behaviour that the shim emulates

1. **`(__int64)double`** (`CVT.L.D`). x86 `CVTTSD2SI` gives `0x8000000000000000` for NaN and out-of-range values. arm64 saturates. The shim macro `__int64` sends the cast through `oracle_x86_int64`.
2. **`strtoul`**. On Windows it returns 32 bits and saturates at `0xFFFFFFFF`. This affects `getnum()`: `.word32`, `.byte`, `.space`, `.org`, `.align`, and immediates found through `getsym`.
3. **`isdigit`/`isalnum`/`tolower`**. These use ASCII classification in the "C" locale. A byte >= 0x80 is never a letter or a digit. With a signed `char`, this is undefined behaviour in the original.
4. **Source-file reading**. `CStdioFile` without `typeBinary` uses Windows text mode. `\r\n` becomes `\n`, and a byte `0x1A` (Ctrl-Z) ends the file.
5. **`printf("%lf")` for CONTROL=3**. The UCRT spelling of non-finite values is `inf`, `-inf`, `nan`, `-nan(ind)` (for `0xFFF8000000000000`), and `nan(snan)`.
6. **`%I64u` / `%I64d`** become `PRIu64` / `PRId64`.
7. **`strcat_s` overflow**. MSVC calls the invalid-parameter handler and the process ends. The shim prints a message and calls `abort()`.

### `doc.cpp` compared to `WinMIPS64Doc.cpp`

- The constructor takes the configuration from the command line and not from `winmips64.ini`. It zeroes all members that the original leaves uninitialised: `pipe`, `history`, the symbol tables, and `cpu`. `branch_target_buffer` has no initial value in the original when no `.ini` file exists. The oracle uses the `--btb` value.
- `one_cycle`: `memset(&result, 0, ...)` before `clock_tick`, as `docs/CONTRACT.md` requires.
- `process_result`: the Spanish status-bar strings are replaced with the English canonical texts of CONTRACT §3. The misaligned-access message has one leading space in the original. The oracle uses two spaces, as all the other messages. `strcat_s` limits of 100 become 300: English messages are longer, and the original would abort when a message exceeds 99 characters. The text goes into `last_msg` and not into the status bar.
- `update_io`, CONTROL=4: the original appends the C string at `&data[addr]`. When the string has no terminating zero, the original reads past the end of data memory. The oracle stops at `DATASIZE`.
- `update_history`: when `cc >= 500`, the original writes past `record::status[500]`. The oracle exits with code 3. This does not occur in any golden trace.
- `openit`: the lines that fail are collected for the `E` lines of the trace. Pass 1 stops at the first error, so there is one `E` line. Pass 2 continues, so there is one `E` line for each failing line. The text is the raw source line without `\n`.
- `AfxMessageBox`, `SetWindowText`, `SetPaneText`, `.ini`/`.las` handling and `UpdateAllViews` are removed or are no-ops.
- `keyboard_input()` emulates `CIOView::OnChar` (see "Keyboard input").
- The `-ftrivial-auto-var-init=zero` compiler flag makes the stack struct `idle` in `EX_ADD`/`EX_MUL` deterministic. The original copies this struct into `a[0]`/`m[0]` with only `.active` set, so `IR` and the other fields are stack garbage in the original.

## Trace format details not fixed by CONTRACT §1

- Widths: 32-bit values (`PC`, `IR`, history `IR`) use `%08x`. 64-bit registers use `%016x`. Everything else uses decimal. `source` is signed decimal: `-1`, `-2` mean "not available, N writers pending".
- `P`: `IF` is `<pipe.active>,<cpu.PC>` (the fetch unit, as `PipeView` shows it). `ID`=`if_id`, `EX`=`integer`, `A`=`a[]`, `M`=`m[]`, `DIV`=`div`, `MEM`=`ex_mem`, `WB`=`mem_wb`. An inactive latch prints its stale `IR`.
- `MSG`: the line is `"MSG " + txt`. Each message in `txt` starts with two spaces, so a line with a message has three spaces after `MSG`. An empty `txt` gives `"MSG "` (with a trailing space). `Waiting for input` never appears on the `MSG` line, because `process_result` does not produce it. Use `RET 18` to see a wait.
- `CL`/`DL`: one line for each word. An empty entry prints `CL <i> ` (with a trailing space). Non-empty entries are the assembler's padded line: tabs become 2 spaces, and the line is padded with spaces to 200 characters.
- `H`: `H <entries>`, then for each entry ` <IR>@<start_cycle>:` and the `stage.substage.cause` triples for `status[0..cycles-start_cycle]`, separated by `/`.
- Input: the cycle block prints first. Then, if `RET` is 18, the next input line goes to the keyboard. When no input is left, the oracle prints `INPUT EOF` and then `END`.
- Assembly error: the output is `ASM ERR` and the `E <line> <text>` lines, with no `END` line. The exit code is 0. A file that cannot be opened gives exit code 1. A bad flag gives exit code 2.

## Keyboard input (`IOView.cpp`)

- CONTROL=8: each character of the line goes into `Terminal` (echo). Then Enter adds `\n`. If the line contains `.`, the value is `atof`. If not, the value is `_atoi64` (saturating `strtoll`). The 8 bytes go into `mm[8..15]`.
- CONTROL=9: only `mm[8]` changes. It gets the first character of the line, with no echo. An empty line delivers 13 (the Enter key). `mm[9..15]` keep their old bytes.
- A backspace (8) in the input line deletes the last character, as in the GUI.

## Golden traces

`gen-golden.sh` writes `testdata/golden/<prog>.<cfg>.trace.gz`. It uses `gzip -9 -n`, and the decompressed content is exactly the stdout of the oracle. The traces are compressed because the uncompressed set is about 2 GB: each cycle has an `H` line of up to 15 KB. The compressed set is about 29 MB. `MANIFEST.txt` lists each trace with its flags, input file, assembly result and `END` values.

The script finds programs that read the keyboard: it runs each program once with no input and looks for `INPUT EOF`. For such a program, it writes `<prog>.input` with default values (`5`, `3.5`, `a`, repeated). No current program needs input, so no `.input` file exists.

## Quirks of the original that the Go port must replicate

1. **History window.** `history[50]` has persistent slots. When 50 entries exist, all slots shift down by one and slot 49 keeps its old content. A new entry sets only `IR`, `start_cycle` and `status[0].stage/cause`. `substage` is written only for ADD/MUL stages. Old `substage` values therefore appear again in the `H` line. `clear()` does not reset `status[0].cause`.
2. **Idle latch copy.** `EX_ADD`/`EX_MUL` do `a[0] = idle` / `m[0] = idle`. The oracle has zero there, so `a[0]`/`m[0]` get `IR=0` and all other fields zero after an issue. Other inactive latches keep their stale `IR` and `ins`.
3. **`.asciiz`/`.ascii` size bug.** The pass-1 length loop tests `*ptr == '\\'` (the first character) and not `*iptr`. Strings with escapes get a different size in pass 1 than the bytes that pass 2 writes. If the string starts with `\`, the size is about half.
4. **`update_io` runs every cycle.** It reads CONTROL as a 32-bit word at `mm[0]` and then clears only `mm[0..3]`. CONTROL=8/9 makes `one_cycle` return 18 before it checks for HALT.
5. **Stall accounting.** `check_stalls` looks at ID, EX, `ADDER[0]`, `MULTIPLIER[0]`, DIV and MEM, in that order. Structural stalls are counted only when `MEM != RAW`. Then `update_history` changes `STALLED` (3) to `STRUCTURAL` (5) in the same way. The `X` line shows the values after that change. The Branch Taken stall is counted only when the delay slot is off.
6. **Number parsing.** `.word` and `.double` use 64-bit `strtoint64`/`strtod`. Every other number goes through `getnum`: 32-bit Windows `strtoul`, which saturates at `0xFFFFFFFF` before the sign is applied.
7. **Integer division.** `ddiv` with `INT64_MIN / -1` makes the x86 original crash (`#DE`). The oracle (and Go) give `INT64_MIN`. Avoid this case in tests.
8. **Floating point.** Generated NaNs differ between x86 (`0xFFF8…`) and arm64 (`0x7FF8…`). No golden trace has a NaN or an infinity. `cvt.l.d` must use the x86 result (`0x8000000000000000` when the value is out of range).
9. **Assembler lines.** Lines are read in Windows text mode. A line of 197 characters or more (without `\n`) is cut and counts as a pass-1 error.

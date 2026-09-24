// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

package core

import (
	"bufio"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

type goldenCase struct {
	name  string // trace file name
	trace string // path of the (possibly gzip'ed) trace
	prog  string // path of the program
	flags []string
	input string // path of the input file or ""
}

func configFromFlags(t *testing.T, flags []string) (Config, int) {
	cfg := DefaultConfig()
	max := 5000
	for i := 0; i+1 < len(flags); i += 2 {
		n, err := strconv.Atoi(flags[i+1])
		if err != nil && flags[i] != "--input" {
			t.Fatalf("bad flag value %v", flags)
		}
		switch flags[i] {
		case "--codebits":
			cfg.CodeBits = n
		case "--databits":
			cfg.DataBits = n
		case "--add":
			cfg.AddLatency = n
		case "--mul":
			cfg.MulLatency = n
		case "--div":
			cfg.DivLatency = n
		case "--forwarding":
			cfg.Forwarding = n != 0
		case "--delayslot":
			cfg.DelaySlot = n != 0
		case "--btb":
			cfg.BTB = n != 0
		case "--max":
			max = n
		}
	}
	return cfg, max
}

// inferFlags maps <prog>.<cfg>.trace to flags when no MANIFEST exists.
func inferFlags(cfg string) []string {
	switch cfg {
	case "nofwd":
		return []string{"--forwarding", "0"}
	case "ds":
		return []string{"--delayslot", "1"}
	case "btb":
		return []string{"--btb", "1"}
	case "lat":
		return []string{"--add", "2", "--mul", "3", "--div", "10"}
	}
	return nil
}

func goldenCases(t *testing.T) []goldenCase {
	root := filepath.Join("..", "testdata")
	gdir := filepath.Join(root, "golden")
	var cases []goldenCase
	if b, err := os.ReadFile(filepath.Join(gdir, "MANIFEST.txt")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if strings.HasPrefix(l, "#") || strings.TrimSpace(l) == "" {
				continue
			}
			col := strings.Split(l, "|")
			for i := range col {
				col[i] = strings.TrimSpace(col[i])
			}
			if len(col) < 4 {
				continue
			}
			c := goldenCase{name: col[0], trace: filepath.Join(gdir, col[0]), prog: filepath.Join(root, "programs", col[1])}
			if col[2] != "(none)" && col[2] != "" {
				c.flags = strings.Fields(col[2])
			}
			if col[3] != "-" && col[3] != "" {
				c.input = filepath.Join(gdir, col[3])
			}
			cases = append(cases, c)
		}
	} else {
		files, _ := filepath.Glob(filepath.Join(gdir, "*.trace*"))
		for _, f := range files {
			base := strings.TrimSuffix(strings.TrimSuffix(filepath.Base(f), ".gz"), ".trace")
			parts := strings.Split(base, ".")
			if len(parts) != 2 {
				continue
			}
			c := goldenCase{name: filepath.Base(f), trace: f, prog: filepath.Join(root, "programs", parts[0]+".s"), flags: inferFlags(parts[1])}
			if in := filepath.Join(gdir, parts[0]+".input"); fileExists(in) {
				c.input = in
			}
			cases = append(cases, c)
		}
	}
	// oracle self-test for memory mapped I/O and keyboard input
	io := filepath.Join("..", "tools", "oracle", "tests")
	if fileExists(filepath.Join(io, "io.default.trace")) {
		cases = append(cases, goldenCase{name: "oracle-io.default.trace", trace: filepath.Join(io, "io.default.trace"),
			prog: filepath.Join(io, "io.s"), input: filepath.Join(io, "io.input")})
	}
	return cases
}

func fileExists(p string) bool { _, err := os.Stat(p); return err == nil }

func readInputLines(p string) []string {
	if p == "" {
		return nil
	}
	b, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var out []string
	sc := bufio.NewScanner(strings.NewReader(string(b)))
	for sc.Scan() {
		out = append(out, strings.TrimSuffix(sc.Text(), "\r"))
	}
	return out
}

func clip(s string) string {
	if len(s) > 300 {
		return s[:300] + "..."
	}
	return s
}

// firstDiff returns the first differing line (1-based) or 0 if identical.
func firstDiff(want, got io.Reader) (int, string, string) {
	wr := bufio.NewReaderSize(want, 1<<20)
	gr := bufio.NewReaderSize(got, 1<<20)
	for n := 1; ; n++ {
		w, werr := wr.ReadString('\n')
		g, gerr := gr.ReadString('\n')
		if w != g {
			return n, w, g
		}
		if werr != nil || gerr != nil {
			if werr != gerr {
				return n, fmt.Sprint("<", werr, ">"), fmt.Sprint("<", gerr, ">")
			}
			return 0, "", ""
		}
	}
}

func TestGolden(t *testing.T) {
	cases := goldenCases(t)
	if len(cases) == 0 {
		t.Skip("no golden traces in testdata/golden")
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			f, err := os.Open(c.trace)
			if err != nil {
				t.Skipf("missing trace: %v", err)
			}
			defer f.Close()
			var want io.Reader = f
			if strings.HasSuffix(c.trace, ".gz") {
				zr, err := gzip.NewReader(f)
				if err != nil {
					t.Fatal(err)
				}
				want = zr
			}
			src, err := os.ReadFile(c.prog)
			if err != nil {
				t.Fatal(err)
			}
			cfg, max := configFromFlags(t, c.flags)
			pr, pw := io.Pipe()
			go func() {
				TraceSource(pw, string(src), cfg, readInputLines(c.input), max)
				pw.Close()
			}()
			n, w, g := firstDiff(want, pr)
			pr.CloseWithError(io.EOF)
			if n != 0 {
				t.Errorf("first difference at line %d\nwant: %s\ngot:  %s", n, clip(w), clip(g))
			}
		})
	}
}

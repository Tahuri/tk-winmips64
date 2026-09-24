// Command wmips is the command line front end of the WinMIPS64 Go port.
//
//	wmips trace <prog.s> [flags]   golden trace (docs/CONTRACT.md §1)
//	wmips run   <prog.s> [flags]   run to HALT, print terminal output and stats
package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"tk-winmips64/core"
)

type options struct {
	prog      string
	cfg       core.Config
	input     []string
	haveInput bool
	max       int
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage: wmips trace|run <prog.s> [--codebits N] [--databits N] [--add N] [--mul N] [--div N]
                          [--forwarding 0|1] [--delayslot 0|1] [--btb 0|1] [--input FILE] [--max N]`)
	os.Exit(2)
}

func fail(format string, a ...any) {
	fmt.Fprintf(os.Stderr, "wmips: "+format+"\n", a...)
	os.Exit(2)
}

func readLines(path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		fail("%v", err)
	}
	t := strings.ReplaceAll(string(b), "\r\n", "\n")
	t = strings.TrimSuffix(t, "\n")
	if t == "" {
		return nil
	}
	return strings.Split(t, "\n")
}

func parseArgs(args []string, defMax int) options {
	o := options{cfg: core.DefaultConfig(), max: defMax}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "-") {
			if o.prog != "" {
				usage()
			}
			o.prog = a
			continue
		}
		name := strings.TrimLeft(a, "-")
		val := ""
		if k := strings.IndexByte(name, '='); k >= 0 {
			name, val = name[:k], name[k+1:]
		} else {
			if i+1 >= len(args) {
				fail("missing value for %s", a)
			}
			i++
			val = args[i]
		}
		num := func() int {
			n, err := strconv.Atoi(val)
			if err != nil {
				fail("bad value for --%s: %q", name, val)
			}
			return n
		}
		switch name {
		case "codebits":
			o.cfg.CodeBits = num()
		case "databits":
			o.cfg.DataBits = num()
		case "add":
			o.cfg.AddLatency = num()
		case "mul":
			o.cfg.MulLatency = num()
		case "div":
			o.cfg.DivLatency = num()
		case "forwarding":
			o.cfg.Forwarding = num() != 0
		case "delayslot":
			o.cfg.DelaySlot = num() != 0
		case "btb":
			o.cfg.BTB = num() != 0
		case "max":
			o.max = num()
		case "input":
			o.input = readLines(val)
			o.haveInput = true
		default:
			fail("unknown flag %s", a)
		}
	}
	if o.prog == "" {
		usage()
	}
	if err := o.cfg.Validate(); err != nil {
		fail("%v", err)
	}
	return o
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	switch os.Args[1] {
	case "trace":
		o := parseArgs(os.Args[2:], 5000)
		src, err := os.ReadFile(o.prog)
		if err != nil {
			fail("%v", err)
		}
		core.TraceSource(os.Stdout, string(src), o.cfg, o.input, o.max)
	case "run":
		o := parseArgs(os.Args[2:], 100000000)
		run(o)
	default:
		usage()
	}
}

func run(o options) {
	src, err := os.ReadFile(o.prog)
	if err != nil {
		fail("%v", err)
	}
	s := core.New(o.cfg)
	if errs := s.Load(string(src)); len(errs) > 0 {
		for _, e := range errs {
			fmt.Fprintf(os.Stderr, "%s:%d: %s: %s\n", o.prog, e.Line, e.Code, e.Text)
		}
		os.Exit(1)
	}
	stdin := bufio.NewReader(os.Stdin)
	input := o.input
	printed := 0
	flush := func() {
		t := s.Snapshot().Terminal
		if len(t) >= printed {
			fmt.Print(t[printed:])
		} else {
			fmt.Print(t) // terminal was cleared (CONTROL=6)
		}
		printed = len(t)
	}
	var last core.RunResult
	remaining := o.max
	for remaining > 0 {
		batch := remaining
		if batch > 1000000 {
			batch = 1000000
		}
		last = s.RunTo(batch)
		remaining -= last.Cycles
		for _, m := range last.Messages {
			if m.Code != "waiting_input" && !strings.HasSuffix(m.Code, "_stall") {
				fmt.Fprintf(os.Stderr, "[%s]\n", m.Text())
			}
		}
		if last.StoppedBy == "halted" || last.StoppedBy == "error" {
			break
		}
		if last.StoppedBy == "input" {
			flush()
			var line string
			if o.haveInput {
				if len(input) == 0 {
					fmt.Fprintln(os.Stderr, "wmips: input exhausted")
					break
				}
				line, input = input[0], input[1:]
			} else {
				l, err := stdin.ReadString('\n')
				if err != nil && l == "" {
					fmt.Fprintln(os.Stderr, "wmips: input exhausted")
					break
				}
				line = strings.TrimRight(l, "\r\n")
			}
			if err := s.SendInput(line); err != nil {
				fail("%v", err)
			}
			printed = len(s.Snapshot().Terminal) // echoed input is not re-printed
		}
	}
	flush()
	sn := s.Snapshot()
	st := sn.Stats
	fmt.Printf("\n--- %s ---\n", sn.Status)
	fmt.Printf("Execution\n  %d cycles\n  %d instructions\n", sn.Cycles, sn.Instructions)
	if sn.Instructions > 0 {
		fmt.Printf("  %.3f Cycles Per Instruction (CPI)\n", st.CPI)
	}
	fmt.Printf("Stalls\n  %d RAW stalls\n  %d WAW stalls\n  %d WAR stalls\n  %d structural stalls\n  %d branch taken stalls\n  %d branch misprediction stalls\n",
		st.RawStalls, st.WawStalls, st.WarStalls, st.StructuralStalls, st.BranchTakenStalls, st.BranchMispredictionStalls)
	fmt.Printf("Code size\n  %d bytes\n", st.CodeSize)
}

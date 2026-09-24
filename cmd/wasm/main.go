//go:build js && wasm

// Command wasm exposes the simulator to JavaScript as the global object
// `wmips` (docs/CONTRACT.md §5). Every function takes/returns JSON strings or
// primitives.
package main

import (
	"encoding/json"
	"strconv"
	"syscall/js"

	"tk-winmips64/core"
)

var sim = core.New(core.DefaultConfig())

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		b, _ = json.Marshal(map[string]any{"ok": false, "error": err.Error()})
	}
	return string(b)
}

func okResult(ok bool, err error) string {
	m := map[string]any{"ok": ok}
	if err != nil {
		m["error"] = err.Error()
	}
	return toJSON(m)
}

func arg(args []js.Value, i int) js.Value {
	if i < len(args) {
		return args[i]
	}
	return js.Undefined()
}

func str(v js.Value) string {
	if v.Type() == js.TypeString {
		return v.String()
	}
	if v.IsUndefined() || v.IsNull() {
		return ""
	}
	return v.String()
}

func num(v js.Value, def int) int {
	switch v.Type() {
	case js.TypeNumber:
		return v.Int()
	case js.TypeString:
		if n, err := strconv.ParseInt(v.String(), 0, 64); err == nil {
			return int(n)
		}
	}
	return def
}

func flt(v js.Value) float64 {
	switch v.Type() {
	case js.TypeNumber:
		return v.Float()
	case js.TypeString:
		f, _ := strconv.ParseFloat(v.String(), 64)
		return f
	}
	return 0
}

func hex64(v js.Value) (uint64, bool) {
	if v.Type() == js.TypeNumber {
		return uint64(v.Int()), true
	}
	s := str(v)
	if len(s) > 2 && (s[:2] == "0x" || s[:2] == "0X") {
		s = s[2:]
	}
	n, err := strconv.ParseUint(s, 16, 64)
	return n, err == nil
}

func parseConfig(s string, base core.Config) (core.Config, error) {
	cfg := base
	if s != "" {
		if err := json.Unmarshal([]byte(s), &cfg); err != nil {
			return cfg, err
		}
	}
	return cfg, cfg.Validate()
}

func fn(f func(args []js.Value) string) js.Func {
	return js.FuncOf(func(this js.Value, args []js.Value) (ret any) {
		defer func() {
			if r := recover(); r != nil {
				ret = toJSON(map[string]any{"ok": false, "error": "internal error"})
			}
		}()
		return f(args)
	})
}

func main() {
	api := map[string]any{
		"init": fn(func(a []js.Value) string {
			cfg, err := parseConfig(str(arg(a, 0)), core.DefaultConfig())
			if err != nil {
				return okResult(false, err)
			}
			sim = core.New(cfg)
			return okResult(true, nil)
		}),
		"setConfig": fn(func(a []js.Value) string {
			cfg, err := parseConfig(str(arg(a, 0)), sim.Config())
			if err != nil {
				return okResult(false, err)
			}
			err = sim.SetConfig(cfg)
			return okResult(err == nil, err)
		}),
		"load": fn(func(a []js.Value) string {
			errs := sim.Load(str(arg(a, 0)))
			if errs == nil {
				errs = []core.AsmError{}
			}
			return toJSON(map[string]any{"ok": len(errs) == 0, "errors": errs})
		}),
		"program":  fn(func(a []js.Value) string { return toJSON(sim.Program()) }),
		"snapshot": fn(func(a []js.Value) string { return toJSON(sim.Snapshot()) }),
		"step":     fn(func(a []js.Value) string { return toJSON(sim.Step()) }),
		"stepN":    fn(func(a []js.Value) string { return toJSON(sim.StepN(num(arg(a, 0), 5))) }),
		"runTo":    fn(func(a []js.Value) string { return toJSON(sim.RunTo(num(arg(a, 0), 20000))) }),
		"reset": fn(func(a []js.Value) string {
			sim.Reset(arg(a, 0).Truthy())
			return okResult(true, nil)
		}),
		"sendInput": fn(func(a []js.Value) string {
			err := sim.SendInput(str(arg(a, 0)))
			return okResult(err == nil, err)
		}),
		"toggleBreakpoint": fn(func(a []js.Value) string {
			sim.ToggleBreakpoint(uint32(num(arg(a, 0), 0)))
			return okResult(true, nil)
		}),
		"setReg": fn(func(a []js.Value) string {
			v, ok := hex64(arg(a, 1))
			return okResult(ok && sim.SetRegChecked(num(arg(a, 0), -1), v), nil)
		}),
		"setFReg": fn(func(a []js.Value) string {
			return okResult(sim.SetFRegChecked(num(arg(a, 0), -1), flt(arg(a, 1))), nil)
		}),
		"setMem": fn(func(a []js.Value) string {
			v, ok := hex64(arg(a, 1))
			return okResult(ok && sim.SetMemChecked(uint32(num(arg(a, 0), -1)), v), nil)
		}),
		"setMemDouble": fn(func(a []js.Value) string {
			return okResult(sim.SetMemDoubleChecked(uint32(num(arg(a, 0), -1)), flt(arg(a, 1))), nil)
		}),
	}
	js.Global().Set("wmips", js.ValueOf(api))
	select {}
}

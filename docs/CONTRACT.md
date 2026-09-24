# tk-winmips64 — Contratos entre componentes

Este documento es la fuente de verdad compartida por:
- `tools/oracle/` — simulador C++ original (fork Andoni v1.60) compilado en Linux/macOS con un shim de MFC. Genera trazas golden.
- `core/` + `cmd/wmips/` + `cmd/wasm/` — port a Go (package `core`, módulo `tk-winmips64`).
- `web/` — frontend React que consume el bridge WASM.

**Comportamiento de referencia:** `third_party/winmips64-andoni/src` (v1.60, con bug fixes).
Base de licencia Apache 2.0: `third_party/winmips64-mcarrick` (Mike Scott).

---

## 1. Formato de traza golden (oracle y `wmips trace` deben producir bytes idénticos)

Invocación:

```
oracle      <prog.s> [flags]  > out.trace
wmips trace <prog.s> [flags]  > out.trace
```

Flags (idénticos en ambas herramientas):

| Flag | Default | Significado |
|---|---|---|
| `--codebits N` | 10 | bits de memoria de código (8..13) |
| `--databits N` | 10 | bits de memoria de datos (4..11) |
| `--add N` | 4 | latencia FP ADD (2..8) |
| `--mul N` | 7 | latencia FP MUL (2..8) |
| `--div N` | 24 | latencia FP DIV (10..30) |
| `--forwarding 0/1` | 1 | forwarding |
| `--delayslot 0/1` | 0 | delay slot |
| `--btb 0/1` | 0 | branch target buffer (incompatible con delay slot → error) |
| `--input FILE` | — | líneas de entrada para CONTROL=8/9 (una por pedido) |
| `--max N` | 5000 | máximo de ciclos |

Protocolo de ejecución:
1. Ensamblar (`openit` equivalente). Si falla: imprimir `ASM ERR` y una línea `E <linea> <texto>` por error, y salir con código 0.
2. Si ensambla: imprimir encabezado de ensamblado (ver abajo).
3. Loop: `one_cycle(show=FALSE)` hasta que `cpu.status == HALTED` o se alcance `--max`.
   - `RESULT` se inicializa a cero (`memset 0`) antes de cada `clock_tick`.
   - Si devuelve `WAITING_FOR_INPUT`: tomar la siguiente línea de `--input` y entregarla con la misma semántica de `IOView.cpp` (CONTROL=8: entero o double si contiene `.`; CONTROL=9: primer carácter). Si no hay más líneas: imprimir `INPUT EOF` y terminar.
   - Después de cada ciclo, imprimir el bloque de ciclo.
4. Al final imprimir `END <cycles> <instructions>`.

Encabezado (después de ensamblar):

```
ASM OK
CODE <hex de cpu.code[0..codesize)>
CSTAT <hex de cpu.cstat[0..codesize)>
DATA <hex de cpu.data[0..datasize)>
DSTAT <hex de cpu.dstat[0..datasize)>
CL <i> <codelines[i]>          (una por cada palabra de código, i = addr/4; texto tal cual, sin \r\n)
DL <i> <datalines[i]>          (una por cada palabra de datos, i = addr/8)
```

Bloque por ciclo (una línea por clave, orden fijo, hex en minúsculas sin `0x`):

```
C <cycle> RET <valor devuelto por one_cycle> CPU <cpu.status> PC <cpu.PC hex>
R <64 entradas "val16hex:source" separadas por espacio, rreg[0..63]>
W <64 entradas "val16hex:source", wreg[0..63]>
FCC <cpu.fp_cc>
P IF <active>,<IR hex> ID <a>,<IR> EX <a>,<IR> A <a>,<IR>;<a>,<IR>... M <a>,<IR>;... DIV <a>,<IR> MEM <a>,<IR> WB <a>,<IR>
X <IF> <ID> <EX> <MEM> <WB> <DIVIDER> A <ADDER[0..addlat)> M <MULTIPLIER[0..mullat)> RR <idrr> <exrr> <memrr> <addrr> <mulrr> <divrr>
S <instructions> <loads> <stores> <raw> <waw> <war> <structural> <branch_taken> <branch_mispred>
MSG <texto del status bar que produciría process_result, en inglés canónico; ver §3; vacío si nada>
T <cpu.Terminal escapado estilo C: \n \\ \" \t y \xHH para no imprimibles>
MEMH <fnv1a-32 hex de data[0..datasize)> <fnv1a-32 hex de dstat[0..datasize)> <fnv1a-32 hex de screen (50*50 uint32 LE)>
H <entries> luego por entrada: <IR hex>@<start_cycle>:<stage>.<substage>.<cause>/...  (status[0..cycles-start_cycle])
```

Notas:
- Los valores de `X` son los enteros de `RESULT` tras `update_history` (que reescribe STALLED→STRUCTURAL).
- `A`/`M` en la línea `P` listan `pipe.a[0..ADD_LATENCY)` y `pipe.m[0..MUL_LATENCY)`.
- `H` usa la ventana histórica de 50 entradas del original (el port Go replica exactamente ese límite en el modo traza).

## 2. API Go (package `core`)

```go
type Config struct {
    CodeBits, DataBits              int  // defaults 10, 10
    AddLatency, MulLatency, DivLatency int // defaults 4, 7, 24
    Forwarding, DelaySlot, BTB      bool // defaults true, false, false
    RegistersAsNumbers              bool // solo presentación
}
func DefaultConfig() Config
func (c Config) Validate() error

type Sim struct{ /* ... */ }
func New(cfg Config) *Sim
func (s *Sim) SetConfig(cfg Config) error       // requiere Load posterior
func (s *Sim) Load(src string) []AsmError       // nil/vacío = OK; hace full reset
func (s *Sim) Reset(full bool)                   // full=true también limpia memoria de datos a la imagen ensamblada... (replicar OnFileReset / OnFullReset)
func (s *Sim) Step() StepResult                  // un ciclo (OnExecuteSingle)
func (s *Sim) StepN(n int) StepResult            // multi-ciclo (OnExecuteMulticycle)
func (s *Sim) RunTo(maxCycles int) RunResult     // OnExecuteRunto, en lotes: corta en breakpoint, HALT, input o maxCycles
func (s *Sim) SendInput(text string) error
func (s *Sim) ToggleBreakpoint(addr uint32)
func (s *Sim) SetReg(i int, v uint64)            // i 0..31 enteros
func (s *Sim) SetFReg(i int, f float64)
func (s *Sim) SetMem(addr uint32, v uint64)      // palabra de 64 bits alineada
func (s *Sim) SetMemDouble(addr uint32, f float64)
func (s *Sim) Program() Program                  // estático tras Load
func (s *Sim) Snapshot() Snapshot                // dinámico
func (s *Sim) Trace(w io.Writer, input []string, maxCycles int) // formato §1
```

`StepResult{Status string; Messages []Message}`, `RunResult{Cycles int; Status string; StoppedBy string; Messages []Message}`.
`Status`: `"ok" | "halted" | "waiting_input"`. `StoppedBy`: `"breakpoint" | "halted" | "input" | "limit" | "error"`.

## 3. Mensajes (códigos, el front los traduce)

```go
type Message struct {
    Code  string `json:"code"`            // ver tabla
    Stage string `json:"stage,omitempty"` // "ID","EX","ADD","MUL","DIV","MEM","FP-DIV","FP-MUL","FP-ADD"
    Reg   string `json:"reg,omitempty"`   // "R5" / "F2"
}
```

| Code | Texto canónico inglés (para línea MSG de la traza) |
|---|---|
| `raw_stall` | `RAW Stall in {stage} ({reg})` |
| `waw_stall` | `WAW Stall in {stage} ({reg})` |
| `war_stall` | `WAR Stall in {stage} ({reg})` |
| `branch_taken_stall` | `Branch Taken Stall` |
| `branch_mispredicted_stall` | `Branch Misprediction Stall` |
| `structural_stall` | `Structural Stall in {stage}` |
| `no_such_code_memory` | `No such code memory!` |
| `integer_overflow` | `Integer overflow!` |
| `divide_by_zero` | `Division by Zero in DIV!` |
| `uninitialized_memory` | `Uninitialised memory in MEM!` |
| `no_such_data_memory` | `No such data memory!` |
| `data_misaligned` | `Fatal Error - misaligned memory LOAD/STORE!` |
| `waiting_input` | `Waiting for input` |

En la línea `MSG` los mensajes se concatenan en el mismo orden que `process_result`, cada uno precedido por dos espacios.

Errores de ensamblado:

```go
type AsmError struct {
    Line int    `json:"line"`  // 1-based
    Code string `json:"code"`  // p.ej. "undefined_symbol", "bad_instruction", "bad_directive", "out_of_memory", "bad_register", "bad_number", "duplicate_symbol", "syntax"
    Text string `json:"text"`  // línea fuente
}
```

## 4. Tipos JSON para el front (`Program`, `Snapshot`)

Los valores de 64 bits viajan como **string hex de 16 dígitos** (JS no representa uint64).

```go
type Program struct {
    CodeSize, DataSize int
    Code []CodeLine `json:"code"` // una por palabra (addr/4)
    Data []DataLine `json:"data"` // una por palabra de 64 bits (addr/8)
    Symbols []Symbol `json:"symbols"`
}
type CodeLine struct { Addr uint32 `json:"addr"`; Word string `json:"word"` /*8 hex*/; Text string `json:"text"`; Label string `json:"label,omitempty"`; Used bool `json:"used"` }
type DataLine struct { Addr uint32 `json:"addr"`; Text string `json:"text"`; Label string `json:"label,omitempty"` }
type Symbol   struct { Name string `json:"name"`; Addr uint32 `json:"addr"`; Kind string `json:"kind"` /*"code"|"data"*/ }

type Snapshot struct {
    Config       Config      `json:"config"`
    Loaded       bool        `json:"loaded"`
    Status       string      `json:"status"`   // "idle"|"ok"|"halted"|"waiting_input"
    InputKind    string      `json:"inputKind,omitempty"` // "number"|"char"
    Cycles       int         `json:"cycles"`
    Instructions int         `json:"instructions"`
    Stats        Stats       `json:"stats"`
    Messages     []Message   `json:"messages"` // del último ciclo
    PC           uint32      `json:"pc"`
    Regs         []RegView   `json:"regs"`   // 32
    FRegs        []FRegView  `json:"fregs"`  // 32
    FPCC         bool        `json:"fpcc"`
    Breakpoints  []uint32    `json:"breakpoints"`
    Predicted    []uint32    `json:"predicted"` // direcciones con cstat&2 (BTB "<<")
    Pipeline     PipeView    `json:"pipeline"`
    Data         []DataWord  `json:"data"`   // una por palabra de 64 bits
    History      []HistEntry `json:"history"`
    Terminal     string      `json:"terminal"`
    Screen       ScreenView  `json:"screen"`
}
type Stats struct {
    Loads, Stores, RawStalls, WawStalls, WarStalls, StructuralStalls, BranchTakenStalls, BranchMispredictionStalls int
    CodeSize, DataSize int
    CPI float64
} // JSON en camelCase: loads, stores, rawStalls, ...
type RegView  struct { Name string `json:"name"`; Value string `json:"value"`; Source string `json:"source"` }
type FRegView struct { Name string `json:"name"`; Bits string `json:"bits"`; Value float64 `json:"value"`; Source string `json:"source"` }
// Source: "reg"|"id"|"ex"|"mem"|"add"|"mul"|"div"|"na"
type StageSlot struct { Active bool `json:"active"`; Addr uint32 `json:"addr"`; Mnemonic string `json:"mnemonic"` }
type PipeView struct {
    IF, ID, EX StageSlot; Add []StageSlot; Mul []StageSlot; Div StageSlot; MEM, WB StageSlot
} // JSON: if, id, ex, add, mul, div, mem, wb
type DataWord struct { Addr uint32 `json:"addr"`; Value string `json:"value"`; Double float64 `json:"double"`; Written bool `json:"written"`; Label string `json:"label,omitempty"`; Text string `json:"text"` }
type HistEntry struct { Addr uint32 `json:"addr"`; Mnemonic string `json:"mnemonic"`; StartCycle int `json:"startCycle"`; Cells []HistCell `json:"cells"` }
type HistCell  struct { Stage string `json:"stage"` /* "IF","ID","EX","A0".."A7","M0".."M7","DIV","MEM","WB","" */; Cause string `json:"cause,omitempty"` /* "raw","waw","war","structural","branch_taken","branch_mispredicted","" */ }
type ScreenView struct { Width int `json:"width"`; Height int `json:"height"`; Drawn bool `json:"drawn"`; Pixels string `json:"pixels"` /* base64 de 50*50*3 bytes RGB */ }
```

Colores del original (para el front):
- Etapas en Code/Pipeline: IF=cyan `#00ffff`, ID/EX-int=rojo `#ff0000`, MEM=verde `#00ff00`, WB=magenta `#ff00ff`, ADD=verde oscuro `#008000`, MUL=cian oscuro `#008080`, DIV=amarillo oscuro `#808000`, PC=amarillo `#ffff00`.
  (En el Code view original: IF cyan, EX-int rojo, MEM verde, WB magenta; ID se muestra en el pipeline diagram.)
- Registros por `source`: id=cyan, ex=rojo, mem=verde, add=verde oscuro, mul=cian oscuro, div=amarillo oscuro, na=gris.

## 5. Bridge WASM (`cmd/wasm`, global JS `wmips`)

Todas las funciones reciben/devuelven strings JSON o primitivos:

| Función JS | Retorno |
|---|---|
| `wmips.init(configJSON)` | `{"ok":true}` o `{"ok":false,"error":"..."}` |
| `wmips.setConfig(configJSON)` | idem |
| `wmips.load(source)` | `{"ok":bool,"errors":[AsmError]}` |
| `wmips.program()` | `Program` |
| `wmips.snapshot()` | `Snapshot` |
| `wmips.step()` | `StepResult` |
| `wmips.stepN(n)` | `StepResult` |
| `wmips.runTo(maxCycles)` | `RunResult` |
| `wmips.reset(full)` | `{"ok":true}` |
| `wmips.sendInput(text)` | `{"ok":bool,"error"?}` |
| `wmips.toggleBreakpoint(addr)` | `{"ok":true}` |
| `wmips.setReg(i, hex)` / `wmips.setFReg(i, float)` | `{"ok":bool}` |
| `wmips.setMem(addr, hex)` / `wmips.setMemDouble(addr, float)` | `{"ok":bool}` |

Config JSON: `{"codeBits":10,"dataBits":10,"addLatency":4,"mulLatency":7,"divLatency":24,"forwarding":true,"delaySlot":false,"btb":false,"registersAsNumbers":false}`.

El artefacto de build es `web/public/wasm/wmips.wasm` + `web/public/wasm/wasm_exec.js` (copiado de `$(go env GOROOT)/lib/wasm/wasm_exec.js`, o `misc/wasm/` según versión).
El front lo carga dentro de un **Web Worker**; `runTo` se llama en lotes de 20 000 ciclos hasta `stoppedBy != "limit"` o hasta que el usuario pulse Stop.

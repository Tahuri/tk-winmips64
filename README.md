# tk-winmips64 — WinMIPS64 en el navegador

Port web del simulador de pipeline MIPS64 **WinMIPS64** (Mike Scott), basado en el fork v1.60 de Andoni Zubimendi.

- **Núcleo en Go** (`core/`): port fiel de `pipeline.cpp`, `utils.cpp` y del ensamblador y el driver de `WinMIPS64Doc.cpp`.
- **El núcleo corre en el navegador** como WebAssembly, dentro de un Web Worker. El servidor no simula nada.
- **Frontend en React** (`web/`) con las 7 ventanas del original, más un editor integrado, en español e inglés.
- **Servidor Go liviano** (`cmd/server/`): sirve la app, el `.wasm` y los programas de ejemplo.

## Levantarlo

**Requisito:** Docker con Compose v2.

```bash
docker compose up -d --build
```

Abrí <http://localhost:8080>. Para usar otro puerto: `WINMIPS64_PORT=9000 docker compose up -d --build`.

Para ver tus propios programas `.s` en el menú **Ejemplos**, descomentá el `volumes:` de `docker-compose.yaml`.

Para detenerlo: `docker compose down`.

## Uso rápido

1. Escribí o abrí un programa `.s` en el panel **Editor** y pulsá **Ensamblar**.
2. Ejecutá con **F7** (un ciclo), **F8** (N ciclos) o **F4** (hasta breakpoint o HALT). **Detener** corta una ejecución larga.
3. Hacé doble click en una línea del panel **Código** para poner o quitar un breakpoint.
4. Hacé doble click en un registro o en una palabra de **Datos** para editarlos.
5. Cambiá forwarding, delay slot, BTB, latencias y tamaños de memoria en **Configuración**.

## Desarrollo local

**Requisitos:** Go 1.25+ y Node 24+.

| Comando | Qué hace |
|---|---|
| `make wasm` | Compila `web/public/wasm/wmips.wasm` y copia `wasm_exec.js` |
| `make dev` | Levanta el servidor en :8080 y Vite en :5173 (con proxy a `/api`) |
| `make test` | `go vet` + `go test` (unit + 195 trazas golden) |
| `make cli` | Compila `bin/wmips` (`wmips run prog.s`, `wmips trace prog.s`) |
| `make e2e` | Levanta el contenedor y corre vitest + Playwright (mock y motor real) |
| `make golden` | Recompila el oráculo C++ y regenera las trazas golden |

Con `?mock=1` en la URL, el frontend usa un motor falso en TypeScript. Sirve para trabajar en la UI sin compilar el WASM.

## Cómo se verifica la fidelidad

El simulador original solo compila en Windows. Aun así, su núcleo casi no depende de MFC.

1. `tools/oracle/` compila el código **original sin modificar** (`pipeline.cpp`, `utils.cpp`) en macOS o Linux, con un shim de MFC. `make -C tools/oracle verify-upstream` comprueba que los archivos son idénticos byte a byte.
2. El oráculo genera una traza por ciclo: registros, latches del pipeline, stalls, estadísticas, terminal, memoria e historial.
3. `testdata/golden/` guarda 195 trazas: 39 programas × 5 configuraciones (default, sin forwarding, delay slot, BTB, latencias alternativas).
4. `core/golden_test.go` exige que el port Go produzca **las mismas trazas byte a byte**. Hoy pasan 196/196.

El port replica también los *quirks* del original. La lista completa está en `tools/oracle/README.md`.

## Estructura

```
core/            núcleo Go (ensamblador, pipeline, E/S, historial, API Sim)
cmd/wasm/        bridge syscall/js → global `wmips`
cmd/wmips/       CLI (run / trace)
cmd/server/      servidor HTTP (estáticos, /api/examples, /healthz)
web/             frontend React + Vite + TypeScript
tools/oracle/    simulador C++ original compilado como oráculo
testdata/        programas de ejemplo y trazas golden
third_party/     código fuente de los dos repos upstream (referencia)
docs/CONTRACT.md contratos entre componentes (traza, API, JSON, bridge)
```

## Licencia

- El WinMIPS64 original de Mike Scott tiene licencia **Apache 2.0** (ver `LICENSE`).
- **Advertencia:** el fork de Andoni Zubimendi (`third_party/winmips64-andoni`) **no declara licencia**. Este port reproduce su comportamiento (bug fixes, traducción al español, registros como números). Antes de publicarlo o distribuirlo, pedile a su autor una licencia explícita.

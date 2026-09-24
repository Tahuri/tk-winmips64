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

## Publicar en GitHub Pages

El simulador corre entero en el navegador, así que el sitio es 100 % estático. El workflow `.github/workflows/pages.yml` compila el WASM y la web, y publica `web/dist`.

1. Subí el repo a GitHub. En el plan gratuito de GitHub Pages, el repo tiene que ser público.
2. Abrí **Settings → Pages → Build and deployment** y elegí **Source: GitHub Actions**.
3. Hacé push a `main`, o corré el workflow **pages** a mano.
4. Abrí `https://<usuario>.github.io/<repo>/`.

**Diferencias con la versión Docker:**
- GitHub Pages no permite headers propios. La CSP va en un `<meta>` de `web/index.html`; `X-Frame-Options` no se puede aplicar.
- Los ejemplos salen de `examples/index.json`, que se genera en el build desde `testdata/programs/`. No hay volumen para agregar programas: commitealos en `testdata/programs/`.
- Revisá si GitHub Pages comprime el `.wasm` (3,6 MB sin comprimir):
  `curl -sI -H 'Accept-Encoding: gzip' https://<usuario>.github.io/<repo>/wasm/wmips.wasm | grep -i content-encoding`

Para probar localmente con la misma ruta base:

```bash
make wasm
cd web && VITE_BASE=/tk-winmips64/ npm run build
npx vite preview --base /tk-winmips64/ --port 4174
E2E_BASE_URL=http://localhost:4174/tk-winmips64/ npx playwright test -c playwright.docker.config.ts
```

## Licencia

tk-winmips64 se distribuye bajo la **Licencia Apache 2.0** (`LICENSE`). Las atribuciones requeridas están en `NOTICE`, que se publica junto con el sitio (`/NOTICE`) y con la imagen Docker (`/app/NOTICE`).

| Fuente | Qué se usa | Licencia |
|---|---|---|
| [mcarrickscott/WinMIPS64](https://github.com/mcarrickscott/WinMIPS64), de Mike Scott | Simulador original, programas de ejemplo, documentación | Apache 2.0 |
| [AndoniZubimendi/WinMIPS64](https://github.com/AndoniZubimendi/WinMIPS64), de Andoni Zubimendi | Bug fixes, traducción al español, "registros como números", CONTROL=4 | **No declara licencia** |

Cómo se cumple Apache 2.0:
- Se incluye una copia de la licencia (`LICENSE`) y los avisos de copyright del original (`NOTICE`).
- Cada archivo derivado indica su origen y que fue modificado. Ver los encabezados de `core/*.go` y `tools/oracle/src/*`. El resto de los archivos llevan `SPDX-License-Identifier: Apache-2.0`.
- `third_party/` conserva copias sin modificar de los dos repos, con el commit exacto (`third_party/README.md`).
- El diálogo **Ayuda → Acerca de** muestra los créditos y los links a las dos fuentes.

**Advertencia:** poner la licencia Apache a este repo **no cambia la licencia de los aportes de Andoni Zubimendi**. Su fork no tiene archivo de licencia, y sin permiso explícito sus cambios siguen siendo suyos. Antes de hacer público el repo o el sitio, pedile que agregue una licencia Apache 2.0 a su fork, o que dé permiso por escrito. La alternativa es reimplementar sus correcciones sin usar su código ni sus textos.

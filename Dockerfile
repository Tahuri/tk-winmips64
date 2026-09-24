# syntax=docker/dockerfile:1

# ---- 1. Go: simulator core as WebAssembly + HTTP server --------------------
FROM golang:1.25-alpine AS go
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY core ./core
COPY cmd ./cmd
RUN mkdir -p /out/wasm \
 && GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o /out/wasm/wmips.wasm ./cmd/wasm \
 && cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm/ 2>/dev/null \
    || cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" /out/wasm/ \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/wmips ./cmd/wmips

# ---- 2. Node: React frontend -------------------------------------------------
FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY web ./
COPY testdata/programs /src/testdata/programs
COPY LICENSE NOTICE /src/
COPY --from=go /out/wasm ./public/wasm
RUN npm run build \
 && find dist -type f \( -name '*.wasm' -o -name '*.js' -o -name '*.css' \) -exec gzip -9 -k {} \;

# ---- 3. Runtime ----------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=go /out/server /app/server
COPY --from=go /out/wmips /app/wmips
COPY --from=web /src/web/dist /app/web
COPY testdata/programs /app/examples
COPY LICENSE NOTICE /app/
ENV ADDR=:8080 STATIC_DIR=/app/web EXAMPLES_DIR=/app/examples
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 CMD ["/app/server", "healthcheck"]
USER nonroot:nonroot
ENTRYPOINT ["/app/server"]

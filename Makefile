GO ?= go
GOROOT := $(shell $(GO) env GOROOT)
WASM_DIR := web/public/wasm
WASM_EXEC := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

.PHONY: test vet wasm cli server web dev up down e2e oracle golden

test:
	$(GO) vet ./...
	$(GO) test ./...

vet:
	$(GO) vet ./...
	GOOS=js GOARCH=wasm $(GO) vet ./cmd/wasm

cli:
	mkdir -p bin
	$(GO) build -o bin/wmips ./cmd/wmips

wasm:
	mkdir -p $(WASM_DIR)
	GOOS=js GOARCH=wasm $(GO) build -o $(WASM_DIR)/wmips.wasm ./cmd/wasm
	cp "$(WASM_EXEC)" $(WASM_DIR)/wasm_exec.js

server:
	mkdir -p bin
	$(GO) build -o bin/server ./cmd/server

web: wasm
	cd web && npm ci && npm run build

# Local development: API server on :8080 + Vite on :5173 (proxies /api).
dev: wasm
	$(GO) run ./cmd/server & cd web && npm run dev

up:
	docker compose up -d --build --wait

down:
	docker compose down

# Unit tests of the frontend + e2e (mock engine) + e2e against the container (real engine).
e2e: up
	cd web && npm test && npx playwright test && npx playwright test -c playwright.docker.config.ts

oracle:
	$(MAKE) -C tools/oracle

golden: oracle
	tools/oracle/gen-golden.sh

GO ?= go
GOROOT := $(shell $(GO) env GOROOT)
WASM_DIR := web/public/wasm
WASM_EXEC := $(firstword $(wildcard $(GOROOT)/lib/wasm/wasm_exec.js $(GOROOT)/misc/wasm/wasm_exec.js))

.PHONY: test vet wasm cli

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

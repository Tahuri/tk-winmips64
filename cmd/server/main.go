// Copyright 2026 tk-winmips64 contributors
// SPDX-License-Identifier: Apache-2.0

// Command server serves the WinMIPS64 web frontend, the WASM simulator and
// the example programs. It does not run simulations: the core runs in the
// browser as WebAssembly.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

type example struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func listExamples(dir string) ([]example, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []example{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".s") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, example{Name: e.Name(), Size: info.Size()})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

func examplesHandler(dir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/api/examples")
		name = strings.TrimPrefix(name, "/")
		if name == "" {
			list, err := listExamples(dir)
			if err != nil {
				http.Error(w, "examples unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(list)
			return
		}
		// Only plain file names from the examples directory are allowed.
		if name != filepath.Base(name) || !strings.HasSuffix(strings.ToLower(name), ".s") {
			http.NotFound(w, r)
			return
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write(data)
	}
}

// spaHandler serves static files and falls back to index.html for client routes.
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := filepath.Join(dir, filepath.Clean("/"+r.URL.Path))
		if st, err := os.Stat(p); err != nil || st.IsDir() && r.URL.Path != "/" {
			if !strings.Contains(filepath.Base(r.URL.Path), ".") {
				http.ServeFile(w, r, filepath.Join(dir, "index.html"))
				return
			}
		}
		if serveGzip(w, r, p) {
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, ".wasm"):
			w.Header().Set("Content-Type", "application/wasm")
			w.Header().Set("Cache-Control", "no-cache")
		case strings.HasPrefix(r.URL.Path, "/assets/"):
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		default:
			w.Header().Set("Cache-Control", "no-cache")
		}
		fs.ServeHTTP(w, r)
	})
}

// serveGzip serves a precompressed "<file>.gz" sibling (generated at build
// time) when the client accepts gzip.
func serveGzip(w http.ResponseWriter, r *http.Request, p string) bool {
	if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
		return false
	}
	ctype := ""
	switch filepath.Ext(p) {
	case ".wasm":
		ctype = "application/wasm"
	case ".js":
		ctype = "text/javascript; charset=utf-8"
	case ".css":
		ctype = "text/css; charset=utf-8"
	default:
		return false
	}
	f, err := os.Open(p + ".gz")
	if err != nil {
		return false
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return false
	}
	h := w.Header()
	h.Set("Content-Type", ctype)
	h.Set("Content-Encoding", "gzip")
	h.Add("Vary", "Accept-Encoding")
	if strings.Contains(p, "/assets/") {
		h.Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		h.Set("Cache-Control", "no-cache")
	}
	http.ServeContent(w, r, "", st.ModTime(), f)
	return true
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("X-Frame-Options", "DENY")
		// wasm-unsafe-eval is required to instantiate the Go WebAssembly module.
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'wasm-unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; worker-src 'self' blob:; connect-src 'self'")
		next.ServeHTTP(w, r)
	})
}

// healthcheck lets the distroless container check itself without curl.
func healthcheck(addr string) {
	if strings.HasPrefix(addr, ":") {
		addr = "127.0.0.1" + addr
	}
	c := http.Client{Timeout: 3 * time.Second}
	resp, err := c.Get("http://" + addr + "/healthz")
	if err != nil || resp.StatusCode != http.StatusOK {
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	addr := env("ADDR", ":8080")
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		healthcheck(addr)
	}
	static := env("STATIC_DIR", "web/dist")
	examples := env("EXAMPLES_DIR", "testdata/programs")

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/api/examples", examplesHandler(examples))
	mux.HandleFunc("/api/examples/", examplesHandler(examples))
	mux.Handle("/", spaHandler(static))

	srv := &http.Server{
		Addr:              addr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      30 * time.Second,
	}
	log.Printf("tk-winmips64 listening on %s (static=%s examples=%s)", addr, static, examples)
	log.Fatal(srv.ListenAndServe())
}

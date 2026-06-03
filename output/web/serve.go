//go:build !js

// Package web — serve.go provides a development HTTP server for serving WASM
// ggplot applications. It is only compiled for native targets (not WASM).
package web

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

//go:embed assets/index.html
var embeddedAssets embed.FS

// Serve starts an HTTP development server that hosts a WASM ggplot application.
// It serves the embedded index.html, the Go WASM support JS (wasm_exec.js),
// and the pre-built WASM binary at wasmPath.
//
// addr is the listen address (e.g. ":8080" or "localhost:8080").
// wasmPath is the path to the compiled .wasm file (e.g. "app.wasm").
//
// This function blocks until the server shuts down.
func Serve(addr string, wasmPath string) error {
	mux := http.NewServeMux()

	// Serve the embedded index.html.
	htmlFS, err := fs.Sub(embeddedAssets, "assets")
	if err != nil {
		return fmt.Errorf("web: embedded assets: %w", err)
	}

	mux.Handle("/", http.FileServer(http.FS(htmlFS)))

	// Locate wasm_exec.js via `go env GOROOT` (avoids deprecated runtime.GOROOT).
	goroot, gorootErr := goEnvGOROOT()
	if gorootErr != nil {
		return fmt.Errorf("web: locate GOROOT: %w", gorootErr)
	}

	wasmExecPath := goroot + "/lib/wasm/wasm_exec.js"

	mux.HandleFunc("/wasm_exec.js", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/javascript")

		data, readErr := os.ReadFile(wasmExecPath) //nolint:gosec // Path from `go env GOROOT`, not user input.
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}

		if _, writeErr := w.Write(data); writeErr != nil {
			http.Error(w, writeErr.Error(), http.StatusInternalServerError)
		}
	})

	// Serve the WASM binary.
	mux.HandleFunc("/app.wasm", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/wasm")

		data, readErr := os.ReadFile(wasmPath) //nolint:gosec // wasmPath is caller-controlled, not user input.
		if readErr != nil {
			http.Error(w, readErr.Error(), http.StatusInternalServerError)
			return
		}

		if _, writeErr := w.Write(data); writeErr != nil {
			http.Error(w, writeErr.Error(), http.StatusInternalServerError)
		}
	})

	fmt.Printf("Serving ggplot WASM app at http://%s\n", addr) //nolint:forbidigo // Dev server intentionally prints to stdout.

	return fmt.Errorf("web: server: %w", http.ListenAndServe(addr, mux)) //nolint:gosec // Dev server, not production.
}

// goEnvGOROOT runs `go env GOROOT` to locate the Go installation root.
func goEnvGOROOT() (string, error) {
	out, err := exec.CommandContext(context.Background(), "go", "env", "GOROOT").Output()
	if err != nil {
		return "", fmt.Errorf("go env GOROOT: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

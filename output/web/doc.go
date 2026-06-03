//go:build js && wasm

// Package web presents a ggplot figure in a browser via WebAssembly. Drawing
// goes through the CPU rasterizer ([canvas.RasterCanvas]) by default, with
// optional SVG mode ([WithSVG]) for metadata-aware vector output. An
// experimental GPU-accelerated path ([WithGPU]) uses WebGPU via gogpu/wgpu.
//
// [Mount] creates a <canvas> (raster) or sets innerHTML (SVG) inside a named
// container element, handles pan/zoom via DOM events translated to
// [output.Event], and blocks until the context is cancelled.
//
// This package is only compiled for GOOS=js GOARCH=wasm. Desktop builds
// should use [output/window] instead. Both packages share the same
// [output.Controller] and [output.Session] infrastructure.
//
// Usage:
//
//	//go:build js && wasm
//	package main
//
//	import (
//	    "context"
//	    "github.com/TuSKan/ggplot/output/web"
//	)
//
//	func main() {
//	    plot := ggplot.New(ds).Aes(aes.X("x"), aes.Y("y")).Layer(geom.Point())
//	    if err := web.Mount(context.Background(), plot, "plot-container"); err != nil {
//	        println("error:", err.Error())
//	    }
//	}
//
// Build:
//
//	GOOS=js GOARCH=wasm go build -o examples/web/app.wasm ./examples/web/
//	$env:GOOS="js"; $env:GOARCH="wasm"; go build -o examples/web/app.wasm ./examples/web/; $env:GOOS=$null; $env:GOARCH=$null; go run ./examples/web/serve/
package web

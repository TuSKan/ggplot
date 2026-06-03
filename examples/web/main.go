//go:build js && wasm

// Interactive scatter plot in the browser via WebAssembly.
//
// Build:
//
//	linux/macOs: GOOS=js GOARCH=wasm go build -o examples/web/app.wasm ./examples/web/
//	windows: $env:GOOS="js"; $env:GOARCH="wasm"; go build -o examples/web/app.wasm ./examples/web/; $env:GOOS=$null; $env:GOARCH=$null; go run ./examples/web/serve/
//
// Serve (from project root):
//
//	go run ./examples/web/serve/
package main

import (
	"context"
	"math"
	"math/rand/v2"
	"syscall/js"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/web"
)

func main() { //nolint:funlen // Example app — length is intentional.
	ctx := context.Background()
	eng := memory.NewEngine(ctx)
	rng := rand.New(rand.NewPCG(42, 99)) //nolint:mnd // Deterministic seed.

	plot := buildPlot(eng, rng)

	// Read the initial render mode from the HTML toolbar.
	mode := readRenderMode()

	// Channel for mode change notifications from the HTML radio buttons.
	modeCh := make(chan string, 1)

	// Register a JS callback so the HTML toolbar can notify Go of mode changes.
	cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			select {
			case modeCh <- args[0].String():
			default: // Drop if channel is full — only latest matters.
			}
		}
		return nil
	})
	js.Global().Set("ggplotSetMode", cb)

	defer cb.Release()

	// Mount/re-mount loop: cancels and re-mounts when the mode changes.
	for {
		mountCtx, cancel := context.WithCancel(ctx)

		// Start the mount in a goroutine so we can listen for mode changes.
		done := make(chan error, 1)

		go func(m string) {
			done <- mount(mountCtx, plot, m)
		}(mode)

		// Wait for either a mode change or mount completion.
		select {
		case newMode := <-modeCh:
			cancel() // Cancel the current mount.
			<-done   // Wait for it to exit.
			mode = newMode

			// Clear the container before re-mounting.
			clearContainer("plot-container")

		case err := <-done:
			cancel()

			if err != nil {
				println("mount error:", err.Error()) //nolint:forbidigo // WASM has no log.

				// Short user-facing message — full error is in the console.
				setStatus("GPU unavailable — using Raster")

				// Fall back to raster and continue the loop.
				mode = "raster"
				clearContainer("plot-container")
				selectMode(mode)

				continue
			}

			return
		}
	}
}

// mount calls web.Mount with the appropriate options for the given mode.
func mount(ctx context.Context, plot *ggplot.Plot, mode string) error {
	var opts []web.Opt

	switch mode {
	case "svg":
		opts = append(opts, web.WithSVG())
	case "gpu":
		opts = append(opts, web.WithGPU())
	}

	return web.Mount(ctx, plot, "plot-container", opts...)
}

// readRenderMode reads the selected radio button value from the HTML toolbar.
func readRenderMode() string {
	val := js.Global().Get("ggplotRenderMode")
	if val.IsUndefined() || val.IsNull() {
		return "raster"
	}

	return val.String()
}

// clearContainer removes all children from the named DOM element.
func clearContainer(id string) {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", id)

	if el.IsNull() || el.IsUndefined() {
		return
	}

	el.Set("innerHTML", "")
}

// setStatus updates the status text in the HTML toolbar.
func setStatus(msg string) {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", "status")

	if el.IsNull() || el.IsUndefined() {
		return
	}

	el.Set("textContent", msg)
}

// selectMode syncs the HTML radio buttons and JS global with the given mode.
// Called on automatic fallback so the toolbar reflects the actual active mode.
func selectMode(mode string) {
	js.Global().Set("ggplotRenderMode", mode)

	doc := js.Global().Get("document")
	radios := doc.Call("querySelectorAll", `input[name="mode"]`)
	n := radios.Get("length").Int()

	for i := range n {
		radio := radios.Index(i)
		checked := radio.Get("value").String() == mode
		radio.Set("checked", checked)

		// Update the label's active class.
		label := radio.Get("parentElement")
		if !label.IsNull() && !label.IsUndefined() {
			classList := label.Get("classList")
			if checked {
				classList.Call("add", "active")
			} else {
				classList.Call("remove", "active")
			}
		}
	}
}

// buildPlot creates the example scatter plot with clustered data and a sine overlay.
func buildPlot(eng *memory.Engine, rng *rand.Rand) *ggplot.Plot {
	const n = 500 //nolint:mnd // 500 points — reasonable for browser CPU rendering.

	x := make([]float64, n)
	y := make([]float64, n)
	group := make([]string, n)

	groups := []string{"A", "B", "C"}
	centers := [][2]float64{{2, 3}, {5, 7}, {8, 2}} //nolint:mnd // Cluster centers.

	for i := range n {
		g := i % len(groups)
		group[i] = groups[g]
		x[i] = centers[g][0] + rng.NormFloat64()*1.2 //nolint:mnd // Cluster spread.
		y[i] = centers[g][1] + rng.NormFloat64()*1.2 //nolint:mnd // Cluster spread.
	}

	// Add a sine curve overlay.
	const nLine = 100 //nolint:mnd // Points for the line.

	lineX := make([]float64, nLine)
	lineY := make([]float64, nLine)
	lineGroup := make([]string, nLine)

	for i := range nLine {
		t := float64(i) / float64(nLine-1) * 10 //nolint:mnd // 0..10 range.
		lineX[i] = t
		lineY[i] = 5 + 3*math.Sin(t) //nolint:mnd // Sine wave centered at y=5.
		lineGroup[i] = "trend"
	}

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", append(x, lineX...)),
		eng.NewFloat64Column("y", append(y, lineY...)),
		eng.NewStringColumn("group", append(group, lineGroup...)),
	)
	if err != nil {
		println("dataset error:", err.Error()) //nolint:forbidigo // WASM has no log.
		return nil
	}

	return ggplot.New(ds, aes.X("x"), aes.Y("y")).
		Layer(geom.Point(
			geom.WithSize(3),    //nolint:mnd // Visible point size.
			geom.WithAlpha(0.7), //nolint:mnd // Semi-transparent.
		), aes.Color("group"), aes.Title("group")).
		Labs(
			ggplot.Title("Interactive Scatter — Pan & Zoom"),
			ggplot.XLab("X"),
			ggplot.YLab("Y"),
		)
}

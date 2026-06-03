//go:build js && wasm

package web

import (
	"fmt"
	"syscall/js"
	"time"
)

const (
	// fpsSampleCount is the circular buffer size for render durations.
	fpsSampleCount = 30 //nolint:mnd // ~1 second of samples at 30 fps.
)

// fpsTracker measures actual render time (Acquire→Commit) and displays
// render-time and throughput in the toolbar. Unlike frame-rate counters
// that measure wall-clock gaps (which include idle and rebuild-delay time),
// this measures how long each render actually takes.
type fpsTracker struct {
	renderStart time.Time
	samples     [fpsSampleCount]float64 // circular buffer of render durations (seconds)
	count       int                     // total samples recorded
	el          js.Value                // cached #fps DOM element
}

// newFPSTracker creates a tracker that updates the #fps element.
func newFPSTracker() *fpsTracker {
	doc := js.Global().Get("document")
	el := doc.Call("getElementById", "fps")

	return &fpsTracker{el: el}
}

// Begin marks the start of a render cycle. Call from Acquire().
func (f *fpsTracker) Begin() {
	f.renderStart = time.Now()
}

// End marks the end of a render cycle and updates the toolbar display.
// Call from Commit(). The displayed values are:
//   - render time: how long the Acquire→Commit cycle took (ms)
//   - throughput: max possible fps if rendering continuously (1/renderTime)
func (f *fpsTracker) End() {
	dt := time.Since(f.renderStart).Seconds()
	if dt <= 0 {
		return
	}

	// Record sample in circular buffer.
	f.samples[f.count%fpsSampleCount] = dt
	f.count++

	// Compute average from available samples.
	n := f.count
	if n > fpsSampleCount {
		n = fpsSampleCount
	}

	var sum float64
	for i := range n {
		sum += f.samples[i]
	}

	avgDt := sum / float64(n)
	ms := avgDt * 1000 //nolint:mnd // seconds to milliseconds

	if !f.el.IsNull() && !f.el.IsUndefined() {
		f.el.Set("textContent", fmt.Sprintf("%.0fms", ms))
	}
}

// Reset clears the counter and display. Call on mode switch.
func (f *fpsTracker) Reset() {
	f.count = 0

	if !f.el.IsNull() && !f.el.IsUndefined() {
		f.el.Set("textContent", "")
	}
}

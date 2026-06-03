//go:build js && wasm

package web

import (
	"image"
	"image/color"
	"syscall/js"
	"time"

	"github.com/TuSKan/ggplot/output"
)

// jsCallback is a registered JS function and the target it was registered on,
// for cleanup via removeEventListener.
type jsCallback struct {
	target js.Value
	event  string
	fn     js.Func
}

// releaseCallbacks removes all registered event listeners. The underlying
// js.Func objects are NOT released immediately because the browser event loop
// may still have queued events that reference them. A released js.Func panics
// with "call to released function" when invoked. Instead, removeEventListener
// stops new dispatches and the GC collects unreachable callbacks.
func releaseCallbacks(cbs []jsCallback) {
	for _, cb := range cbs {
		if !cb.target.IsUndefined() && !cb.target.IsNull() {
			cb.target.Call("removeEventListener", cb.event, cb.fn)
		}
	}
}

// addEventListener registers a DOM event listener and appends the callback
// to cbs for later cleanup.
func addEventListener(cbs *[]jsCallback, target js.Value, event string, fn func(js.Value)) {
	jsFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			fn(args[0])
		}
		return nil
	})
	*cbs = append(*cbs, jsCallback{target: target, event: event, fn: jsFn})
	target.Call("addEventListener", event, jsFn)
}

// addPassiveEventListener registers a passive DOM event listener. Passive
// listeners improve scroll performance — the browser knows preventDefault
// won't be called.
func addPassiveEventListener(cbs *[]jsCallback, target js.Value, event string, fn func(js.Value)) {
	jsFn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) > 0 {
			fn(args[0])
		}
		return nil
	})
	opts := js.Global().Get("Object").New()
	opts.Set("passive", true)
	*cbs = append(*cbs, jsCallback{target: target, event: event, fn: jsFn})
	target.Call("addEventListener", event, jsFn, opts)
}

// putImageData transfers an image.RGBA to a Canvas2D context via ImageData.
// The pixel format is converted from Go's RGBA to Canvas2D's RGBA (identical
// byte order; no conversion needed for non-premultiplied alpha).
func putImageData(ctx2d js.Value, img image.Image) {
	rgba, ok := img.(*image.RGBA)
	if !ok {
		// Fallback: slow path for non-RGBA images.
		b := img.Bounds()
		rgba = image.NewRGBA(b)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
	}

	b := rgba.Bounds()
	w, h := b.Dx(), b.Dy()

	// Create a Uint8ClampedArray from Go pixel data.
	jsData := js.Global().Get("Uint8ClampedArray").New(len(rgba.Pix))
	js.CopyBytesToJS(jsData, rgba.Pix)

	// Create ImageData and put it on the canvas.
	imgData := js.Global().Get("ImageData").New(jsData, w, h)
	ctx2d.Call("putImageData", imgData, 0, 0)
}

// requestAnimationFrame schedules fn on the next browser frame. Returns the
// request ID for cancellation via cancelAnimationFrame.
func requestAnimationFrame(fn func()) int {
	var cb js.Func
	cb = js.FuncOf(func(_ js.Value, _ []js.Value) any {
		cb.Release()
		fn()
		return nil
	})
	return js.Global().Call("requestAnimationFrame", cb).Int()
}

// registerPointerEvents sets up pointer, wheel, and context-menu DOM event
// listeners on elem and feeds the events channel. Returns callbacks for cleanup.
// This does NOT register a ResizeObserver — use [observeResize] separately.
func registerPointerEvents(elem js.Value, events chan<- output.Event) []jsCallback {
	var cbs []jsCallback

	// Track mouse position for scroll events (matching output/window pattern).
	var mouseX, mouseY float64

	// Double-click detection: two clicks within 400ms.
	var lastClickTime time.Time

	// Pointer down.
	addEventListener(&cbs, elem, "pointerdown", func(ev js.Value) {
		ev.Call("preventDefault")
		// Capture pointer so we get pointermove/up even outside the element.
		elem.Call("setPointerCapture", ev.Get("pointerId"))

		x, y := ev.Get("offsetX").Float(), ev.Get("offsetY").Float()
		mouseX, mouseY = x, y

		now := time.Now()
		if now.Sub(lastClickTime) < 400*time.Millisecond { //nolint:mnd // Standard double-click threshold.
			events <- output.Event{Kind: output.EventDoubleClick, X: x, Y: y}
			lastClickTime = time.Time{} // reset to avoid triple-click
		} else {
			events <- output.Event{Kind: output.EventPointerDown, X: x, Y: y}
			lastClickTime = now
		}
	})

	// Pointer up.
	addEventListener(&cbs, elem, "pointerup", func(ev js.Value) {
		x, y := ev.Get("offsetX").Float(), ev.Get("offsetY").Float()
		mouseX, mouseY = x, y
		events <- output.Event{Kind: output.EventPointerUp, X: x, Y: y}
	})

	// Pointer move.
	addPassiveEventListener(&cbs, elem, "pointermove", func(ev js.Value) {
		x, y := ev.Get("offsetX").Float(), ev.Get("offsetY").Float()
		mouseX, mouseY = x, y
		events <- output.Event{Kind: output.EventPointerMove, X: x, Y: y}
	})

	// Wheel / scroll.
	addEventListener(&cbs, elem, "wheel", func(ev js.Value) {
		ev.Call("preventDefault")
		dx := ev.Get("deltaX").Float()
		dy := ev.Get("deltaY").Float()
		events <- output.Event{
			Kind: output.EventScroll,
			X:    mouseX,
			Y:    mouseY,
			DX:   dx,
			DY:   dy,
		}
	})

	// Suppress context menu on the canvas.
	addEventListener(&cbs, elem, "contextmenu", func(ev js.Value) {
		ev.Call("preventDefault")
	})

	return cbs
}

// observeResize registers a ResizeObserver on target and sends EventResize
// events to the channel when the element's size actually changes. Returns the
// ResizeObserver JS object — call ro.Call("disconnect") on cleanup. Returns
// js.Undefined() if ResizeObserver is not available.
func observeResize(target js.Value, events chan<- output.Event) js.Value {
	resizeObserverClass := js.Global().Get("ResizeObserver")
	if resizeObserverClass.IsUndefined() {
		return js.Undefined()
	}

	// Deduplication: only fire when size actually changes.
	var lastW, lastH int

	cb := js.FuncOf(func(_ js.Value, args []js.Value) any {
		if len(args) == 0 {
			return nil
		}
		entries := args[0]
		if entries.Length() == 0 {
			return nil
		}
		entry := entries.Index(0)
		cr := entry.Get("contentRect")
		w := int(cr.Get("width").Float())
		h := int(cr.Get("height").Float())
		if w > 0 && h > 0 && (w != lastW || h != lastH) {
			lastW, lastH = w, h
			events <- output.Event{
				Kind: output.EventResize,
				X:    float64(w),
				Y:    float64(h),
			}
		}
		return nil
	})

	ro := resizeObserverClass.New(cb)
	ro.Call("observe", target)

	return ro
}

// registerDOMEvents sets up all DOM event listeners (pointer + resize) on elem.
// Used by raster surface which registers events once. Returns callbacks for cleanup.
func registerDOMEvents(elem js.Value, events chan<- output.Event) []jsCallback {
	cbs := registerPointerEvents(elem, events)

	// ResizeObserver on the parent — raster canvas persists, so register once.
	observeResize(elem.Get("parentElement"), events)

	return cbs
}

// clearCanvas fills a canvas with white via Canvas2D (used before drawing).
func clearCanvas(ctx2d js.Value, w, h int) {
	r, g, b, _ := color.White.RGBA()
	ctx2d.Set("fillStyle", "rgb("+itoa(int(r>>8))+","+itoa(int(g>>8))+","+itoa(int(b>>8))+")")
	ctx2d.Call("fillRect", 0, 0, w, h)
}

// itoa is a minimal int-to-string conversion to avoid importing strconv in
// WASM (saves ~40KB in the binary).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	neg := false
	if n < 0 {
		neg = true
		n = -n
	}

	buf := [20]byte{}
	i := len(buf) - 1
	for n > 0 {
		buf[i] = byte('0' + n%10) //nolint:mnd // Base 10 digit extraction.
		n /= 10                   //nolint:mnd // Base 10 division.
		i--
	}

	if neg {
		buf[i] = '-'
		i--
	}

	return string(buf[i+1:])
}

package ui

import (
	"fmt"

	"github.com/TuSKan/ggplot/pkg/output"
	"github.com/gogpu/gg"
	_ "github.com/gogpu/gg/gpu" // Register GPU accelerator
	"github.com/gogpu/gg/integration/ggcanvas"
	"github.com/gogpu/gogpu"
)

// GPUWindowPresenter implements output.Presenter using gogpu/gg hardware windowing,
// maintaining constant resizing and relative centered scaling behaviors.
type GPUWindowPresenter struct{}

// NewGPUWindowPresenter constructs a new GPUWindowPresenter.
func NewGPUWindowPresenter() *GPUWindowPresenter {
	return &GPUWindowPresenter{}
}

// Show initiates blocking visual output loops directly attaching standard layouts to hardware windowing mechanisms.
func (p *GPUWindowPresenter) Show(o *output.Output) error {
	// 1. Initialize the native host application layer exactly matching gogpu's architecture.
	gogpuApp := gogpu.NewApp(gogpu.DefaultConfig().
		WithTitle("ggplot").
		WithSize(o.Width, o.Height).
		WithContinuousRender(false)) // Event-driven functionally statically.

	var canvas *ggcanvas.Canvas
	var loopErr error

	gogpuApp.OnDraw(func(dc *gogpu.Context) {
		w, h := dc.Width(), dc.Height()
		if w <= 0 || h <= 0 {
			return
		}

		if canvas == nil {
			provider := gogpuApp.GPUContextProvider()
			if provider == nil {
				return
			}
			var err error
			canvas, err = ggcanvas.New(provider, w, h)
			if err != nil {
				loopErr = fmt.Errorf("ggcanvas setup error: %w", err)
				gogpuApp.Close()
				return
			}
		}

		cw, ch := canvas.Size()
		if cw != w || ch != h {
			if err := canvas.Resize(w, h); err != nil {
				loopErr = fmt.Errorf("ggcanvas resize error: %w", err)
				gogpuApp.Close()
				return
			}
		}

		if err := canvas.Draw(func(cc *gg.Context) {
			// Wipe the whole canvas to a guaranteed solid thematic background
			// bypassing ClearWithColor implementation logic transparency bugs.
			if o.Theme.Background != nil {
				cc.ClearWithColor(gg.FromColor(o.Theme.Background))
			}

			// Scale the graphical AST relative to window resize events mapping uniformly
			scale := min(float64(w)/float64(o.Width), float64(h)/float64(o.Height))

			// Calculate center anchors for the scaled box within the actual window rendering frame
			scaledW := float64(o.Width) * scale
			scaledH := float64(o.Height) * scale
			dx := (float64(w) - scaledW) / 2.0
			dy := (float64(h) - scaledH) / 2.0

			cc.Push()
			cc.Translate(dx, dy)
			cc.Scale(scale, scale)

			// Resolve the user's highly efficient closure mapping
			if o.Draw != nil {
				o.Draw(cc)
			}

			cc.Pop()
		}); err != nil {
			loopErr = fmt.Errorf("ggcanvas draw error: %w", err)
			gogpuApp.Close()
			return
		}

		// Fast zero-copy rendering directly mapped to the GPU presentation surface context
		if err := canvas.Render(dc.RenderTarget()); err != nil {
			loopErr = fmt.Errorf("ggcanvas render error: %w", err)
			gogpuApp.Close()
			return
		}
	})

	gogpuApp.OnClose(func() {
		gg.CloseAccelerator()
	})

	if err := gogpuApp.Run(); err != nil {
		return fmt.Errorf("application run error: %w", err)
	}

	return loopErr
}

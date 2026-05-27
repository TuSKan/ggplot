package ggplot

import (
	"context"
	"errors"
	"fmt"
	"math"
	"testing"

	"github.com/TuSKan/ggplot/coord"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
)

// Test-only sentinel errors for err113 compliance.
var (
	errTestColumn     = errors.New("column x not found")
	errTestUnderlying = errors.New("underlying problem")
)

func TestError_Format(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  *Error
		want string
	}{
		{
			name: "build/no layer/facet",
			err:  Errorf(PhaseBuild, -1, "facet", nil, "facet split"),
			want: "ggplot [build/facet]: facet split",
		},
		{
			name: "build/layer 2/transform with cause",
			err:  Errorf(PhaseBuild, 2, "transform", errTestColumn, "pipeline failed for group %q", "A"),
			want: `ggplot [build/layer 2/transform]: pipeline failed for group "A": column x not found`,
		},
		{
			name: "draw/no stage",
			err:  Errorf(PhaseDraw, -1, "", nil, "context cancelled"),
			want: "ggplot [draw]: context cancelled",
		},
		{
			name: "render/format",
			err:  Errorf(PhaseRender, -1, "", ErrUnsupportedFormat, "unsupported format %q", "gif"),
			want: `ggplot [render]: unsupported format "gif": ggplot: unsupported output format`,
		},
		{
			name: "layer 0",
			err:  Errorf(PhaseBuild, 0, "scale", nil, "scale training failed"),
			want: "ggplot [build/layer 0/scale]: scale training failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := tt.err.Error()
			if got != tt.want {
				t.Errorf("Error() =\n  %q\nwant\n  %q", got, tt.want)
			}
		})
	}
}

func TestError_Unwrap(t *testing.T) {
	t.Parallel()

	cause := errTestUnderlying
	err := Errorf(PhaseBuild, 1, "transform", cause, "pipeline failed")

	if !errors.Is(err, cause) {
		t.Errorf("errors.Is(err, cause) = false, want true")
	}
}

func TestError_Is_Sentinel(t *testing.T) {
	t.Parallel()

	err := Errorf(PhaseBuild, -1, "validate", ErrNoLayers, "no layers")

	if !errors.Is(err, ErrNoLayers) {
		t.Errorf("errors.Is(err, ErrNoLayers) = false, want true")
	}

	if errors.Is(err, ErrRenderFailed) {
		t.Errorf("errors.Is(err, ErrRenderFailed) = true, want false")
	}
}

func TestError_Is_WrappedSentinel(t *testing.T) {
	t.Parallel()

	// Two-level wrapping: Error → fmt.Errorf → sentinel
	inner := fmt.Errorf("column missing: %w", ErrMissingAesthetic)
	outer := Errorf(PhaseBuild, 3, "scale", inner, "training failed")

	if !errors.Is(outer, ErrMissingAesthetic) {
		t.Errorf("errors.Is through two-level wrap = false, want true")
	}
}

func TestError_As(t *testing.T) {
	t.Parallel()

	err := Errorf(PhaseBuild, 5, "position", nil, "adjust failed")

	// Wrap it in fmt.Errorf to test As through wrapping.
	wrapped := fmt.Errorf("plot failed: %w", err)

	var ggErr *Error

	if !errors.As(wrapped, &ggErr) {
		t.Fatalf("errors.As failed")
	}

	if ggErr.Phase != PhaseBuild {
		t.Errorf("Phase = %v, want PhaseBuild", ggErr.Phase)
	}

	if ggErr.Layer != 5 {
		t.Errorf("Layer = %d, want 5", ggErr.Layer)
	}

	if ggErr.Stage != "position" {
		t.Errorf("Stage = %q, want %q", ggErr.Stage, "position")
	}
}

func TestPhase_String(t *testing.T) {
	t.Parallel()

	tests := []struct {
		phase Phase
		want  string
	}{
		{PhaseBuild, "build"},
		{PhaseDraw, "draw"},
		{PhaseRender, "render"},
		{Phase(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()

			if got := tt.phase.String(); got != tt.want {
				t.Errorf("Phase(%d).String() = %q, want %q", tt.phase, got, tt.want)
			}
		})
	}
}

// --- coordApplyKernel: direct MathKernel dispatch ---

func newTestMathKernel() (dataset.MathKernel, func(string, []float64) dataset.AnyColumn) {
	eng := memory.NewEngine(context.Background())

	mk, ok := dataset.Engine(eng).(dataset.MathKernel)
	if !ok {
		panic("memory engine does not implement MathKernel")
	}

	return mk, func(name string, data []float64) dataset.AnyColumn {
		return eng.NewFloat64Column(name, data)
	}
}

func TestCoordApplyKernel_Log10(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	col := newCol("x", []float64{1, 10, 100, 1000, 0.01})

	result, err := coordApplyKernel(mk, col, "log10")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	vals := fc.Values()
	want := []float64{0, 1, 2, 3, -2}

	for i, v := range vals {
		if math.Abs(v-want[i]) > 1e-12 {
			t.Errorf("log10[%d] = %g, want %g", i, v, want[i])
		}
	}
}

func TestCoordApplyKernel_Log2(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	col := newCol("x", []float64{1, 2, 4, 8, 0.5})

	result, err := coordApplyKernel(mk, col, "log2")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	vals := fc.Values()
	want := []float64{0, 1, 2, 3, -1}

	for i, v := range vals {
		if math.Abs(v-want[i]) > 1e-12 {
			t.Errorf("log2[%d] = %g, want %g", i, v, want[i])
		}
	}
}

func TestCoordApplyKernel_Sqrt(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	col := newCol("x", []float64{0, 1, 4, 9, 100})

	result, err := coordApplyKernel(mk, col, "sqrt")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	vals := fc.Values()
	want := []float64{0, 1, 2, 3, 10}

	for i, v := range vals {
		if math.Abs(v-want[i]) > 1e-12 {
			t.Errorf("sqrt[%d] = %g, want %g", i, v, want[i])
		}
	}
}

func TestCoordApplyKernel_Reverse(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	col := newCol("x", []float64{-5, 0, 3.14, 100})

	result, err := coordApplyKernel(mk, col, "reverse")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	vals := fc.Values()
	want := []float64{5, 0, -3.14, -100}

	for i, v := range vals {
		if math.Abs(v-want[i]) > 1e-12 {
			t.Errorf("reverse[%d] = %g, want %g", i, v, want[i])
		}
	}
}

func TestCoordApplyKernel_Unknown_ReturnsError(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	col := newCol("x", []float64{1, 2, 3})

	_, err := coordApplyKernel(mk, col, "unknown")
	if err == nil {
		t.Fatal("expected error for unknown transform name")
	}
}

// --- Roundtrip: kernel + inverse ---

func TestCoordRoundtrip_Log10(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	data := []float64{0.001, 1, 42, 1e6}
	col := newCol("x", data)

	result, err := coordApplyKernel(mk, col, "log10")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	for i, v := range fc.Values() {
		back := math.Pow(10, v) //nolint:mnd // 10^v inverse of log10.
		if math.Abs(back-data[i])/math.Max(math.Abs(data[i]), 1e-15) > 1e-10 {
			t.Errorf("log10 roundtrip(%g) = %g, want %g", data[i], back, data[i])
		}
	}
}

func TestCoordRoundtrip_Sqrt(t *testing.T) {
	t.Parallel()

	mk, newCol := newTestMathKernel()
	data := []float64{0, 1, 25, 100}
	col := newCol("x", data)

	result, err := coordApplyKernel(mk, col, "sqrt")
	if err != nil {
		t.Fatal(err)
	}

	fc, ok := result.(dataset.Column[float64])
	if !ok {
		t.Fatal("result is not Column[float64]")
	}

	for i, v := range fc.Values() {
		back := v * v
		if math.Abs(back-data[i]) > 1e-12 {
			t.Errorf("sqrt roundtrip(%g) = %g, want %g", data[i], back, data[i])
		}
	}
}

// --- coordTickFormatter ---

func TestCoordTickFormatter_Log10(t *testing.T) {
	t.Parallel()

	fmtFn := coordTickFormatter(coord.TransLog10)
	if fmtFn == nil {
		t.Fatal("coordTickFormatter(log10) returned nil")
	}

	lbl := fmtFn(2)
	if lbl != "100" {
		t.Errorf("FormatTick(2) = %q, want %q", lbl, "100")
	}
}

func TestCoordTickFormatter_Sqrt(t *testing.T) {
	t.Parallel()

	fmtFn := coordTickFormatter(coord.TransSqrt)
	if fmtFn == nil {
		t.Fatal("coordTickFormatter(sqrt) returned nil")
	}

	lbl := fmtFn(5)
	if lbl != "25" {
		t.Errorf("FormatTick(5) = %q, want %q", lbl, "25")
	}
}

func TestCoordTickFormatter_NaN(t *testing.T) {
	t.Parallel()

	fmtFn := coordTickFormatter(coord.TransLog10)

	lbl := fmtFn(math.NaN())
	if lbl != "" {
		t.Errorf("FormatTick(NaN) = %q, want empty", lbl)
	}

	lbl = fmtFn(math.Inf(1))
	if lbl != "" {
		t.Errorf("FormatTick(+Inf) = %q, want empty", lbl)
	}
}

func TestCoordTickFormatter_Identity_ReturnsNil(t *testing.T) {
	t.Parallel()

	fmtFn := coordTickFormatter(coord.TransIdentity)
	if fmtFn != nil {
		t.Error("coordTickFormatter(identity) should return nil")
	}
}

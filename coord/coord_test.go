package coord_test

import (
	"math"
	"testing"

	"github.com/TuSKan/ggplot/coord"
)

// --- TransFunc: specification ---

func TestTransFunc_IsIdentity(t *testing.T) {
	t.Parallel()

	if !coord.TransIdentity.IsIdentity() {
		t.Error("TransIdentity.IsIdentity() = false, want true")
	}

	if coord.TransLog10.IsIdentity() {
		t.Error("TransLog10.IsIdentity() = true, want false")
	}

	if coord.TransSqrt.IsIdentity() {
		t.Error("TransSqrt.IsIdentity() = true, want false")
	}

	if coord.TransLog2.IsIdentity() {
		t.Error("TransLog2.IsIdentity() = true, want false")
	}

	if coord.TransReverse.IsIdentity() {
		t.Error("TransReverse.IsIdentity() = true, want false")
	}
}

func TestTransFunc_Names(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tf   coord.TransFunc
		want string
	}{
		{coord.TransLog10, "log10"},
		{coord.TransLog2, "log2"},
		{coord.TransSqrt, "sqrt"},
		{coord.TransReverse, "reverse"},
		{coord.TransIdentity, "identity"},
	}

	for _, tt := range tests {
		if tt.tf.Name != tt.want {
			t.Errorf("TransFunc.Name = %q, want %q", tt.tf.Name, tt.want)
		}
	}
}

// --- Trans constructor ---

func TestTrans_EmptyName_FallsBackToIdentity(t *testing.T) {
	t.Parallel()

	c := coord.Trans(
		coord.TransFunc{Name: ""},
		coord.TransIdentity,
	)

	tr, ok := c.(coord.Transformer)
	if !ok {
		t.Fatal("coord.Trans() does not implement Transformer")
	}

	xt := tr.XTrans()
	if !xt.IsIdentity() {
		t.Errorf("empty Name should fall back to identity, got %q", xt.Name)
	}
}

// --- Transformer interface ---

func TestTransCoord_XTrans_YTrans(t *testing.T) {
	t.Parallel()

	c := coord.Trans(coord.TransLog10, coord.TransSqrt)

	tr, ok := c.(coord.Transformer)
	if !ok {
		t.Fatal("coord.Trans() does not implement Transformer")
	}

	xt := tr.XTrans()
	if xt.Name != "log10" {
		t.Errorf("XTrans().Name = %q, want %q", xt.Name, "log10")
	}

	yt := tr.YTrans()
	if yt.Name != "sqrt" {
		t.Errorf("YTrans().Name = %q, want %q", yt.Name, "sqrt")
	}
}

func TestTransCoord_TransNames(t *testing.T) {
	t.Parallel()

	c := coord.Trans(coord.TransLog10, coord.TransReverse)

	tr, ok := c.(coord.Transformer)
	if !ok {
		t.Fatal("coord.Trans() does not implement Transformer")
	}

	xt := tr.XTrans()
	if xt.Name != "log10" {
		t.Errorf("XTrans().Name = %q, want %q", xt.Name, "log10")
	}

	yt := tr.YTrans()
	if yt.Name != "reverse" {
		t.Errorf("YTrans().Name = %q, want %q", yt.Name, "reverse")
	}
}

// --- Coord.Transform ---

func TestTransCoord_Transform_IsCartesian(t *testing.T) {
	t.Parallel()

	c := coord.Trans(coord.TransLog10, coord.TransSqrt)

	// Transform should be standard Cartesian mapping (data is already
	// transformed at the engine level).
	px, py := c.Transform(0.5, 0.75, 200, 100)

	wantPx := 0.5 * 200
	wantPy := 100 - 0.75*100

	if math.Abs(px-wantPx) > 1e-12 {
		t.Errorf("Transform px = %g, want %g", px, wantPx)
	}

	if math.Abs(py-wantPy) > 1e-12 {
		t.Errorf("Transform py = %g, want %g", py, wantPy)
	}
}

func TestTransCoord_String(t *testing.T) {
	t.Parallel()

	c := coord.Trans(coord.TransLog10, coord.TransSqrt)
	s := c.String()

	if s != "trans(log10, sqrt)" {
		t.Errorf("String() = %q, want %q", s, "trans(log10, sqrt)")
	}
}

// --- Non-Trans coord types (regression) ---

func TestCartesian_Transform(t *testing.T) {
	t.Parallel()

	c := coord.Cartesian()
	px, py := c.Transform(0.5, 0.75, 200, 100)

	if math.Abs(px-100) > 1e-12 {
		t.Errorf("px = %g, want 100", px)
	}

	if math.Abs(py-25) > 1e-12 {
		t.Errorf("py = %g, want 25", py)
	}
}

func TestCartesian_String(t *testing.T) {
	t.Parallel()

	if coord.Cartesian().String() != "cartesian" {
		t.Errorf("String() = %q, want %q", coord.Cartesian().String(), "cartesian")
	}
}

func TestFixed_AspectRatio(t *testing.T) {
	t.Parallel()

	c := coord.Fixed(2)

	f, ok := c.(coord.Fixer)
	if !ok {
		t.Fatal("coord.Fixed() does not implement Fixer")
	}

	if f.AspectRatio() != 2 {
		t.Errorf("AspectRatio() = %g, want 2", f.AspectRatio())
	}
}

func TestFixed_NegativeRatio_Defaults(t *testing.T) {
	t.Parallel()

	c := coord.Fixed(-1)

	f, ok := c.(coord.Fixer)
	if !ok {
		t.Fatal("coord.Fixed() does not implement Fixer")
	}

	if f.AspectRatio() != 1 {
		t.Errorf("AspectRatio() = %g, want 1 (default)", f.AspectRatio())
	}
}

func TestCartesianZoom_ZoomBounds(t *testing.T) {
	t.Parallel()

	lo, hi := 1.0, 10.0

	c := coord.CartesianZoom([2]*float64{&lo, &hi}, [2]*float64{nil, nil})

	z, ok := c.(coord.Zoomer)
	if !ok {
		t.Fatal("CartesianZoom does not implement Zoomer")
	}

	xlim, ylim := z.ZoomBounds()

	if xlim[0] == nil || *xlim[0] != 1 {
		t.Errorf("xlim[0] = %v, want 1", xlim[0])
	}

	if xlim[1] == nil || *xlim[1] != 10 {
		t.Errorf("xlim[1] = %v, want 10", xlim[1])
	}

	if ylim[0] != nil || ylim[1] != nil {
		t.Error("ylim should be nil for auto-detect")
	}
}

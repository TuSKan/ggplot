package theme

import (
	"image/color"
	"testing"
)

func TestMergeText(t *testing.T) {
	t.Parallel()

	parent := ElementText{Family: "serif", Size: 14, Color: color.Black}
	child := ElementText{Size: 18} // only overrides Size

	got := MergeText(child, parent)

	if got.Family != "serif" {
		t.Errorf("Family = %q, want %q", got.Family, "serif")
	}

	if got.Size != 18 {
		t.Errorf("Size = %v, want 18", got.Size)
	}

	if got.Color != color.Black {
		t.Errorf("Color = %v, want black", got.Color)
	}
}

func TestMergeLine(t *testing.T) {
	t.Parallel()

	parent := ElementLine{Color: color.White, Size: 2, Linetype: []float64{4, 4}}
	child := ElementLine{Size: 1} // only overrides Size

	got := MergeLine(child, parent)

	if got.Color != color.White {
		t.Errorf("Color = %v, want white", got.Color)
	}

	if got.Size != 1 {
		t.Errorf("Size = %v, want 1", got.Size)
	}

	if len(got.Linetype) != 2 {
		t.Errorf("Linetype len = %d, want 2", len(got.Linetype))
	}
}

func TestMergeRect(t *testing.T) {
	t.Parallel()

	parent := ElementRect{Fill: color.White, Color: color.Black, Size: 1}
	child := ElementRect{Fill: color.RGBA{R: 255, A: 255}}

	got := MergeRect(child, parent)

	if got.Color != color.Black {
		t.Errorf("Color = %v, want black", got.Color)
	}

	if got.Size != 1 {
		t.Errorf("Size = %v, want 1", got.Size)
	}
}

func TestIsBlank(t *testing.T) {
	t.Parallel()

	if !IsBlank(ElementBlank{}) {
		t.Error("IsBlank(ElementBlank{}) = false, want true")
	}

	if IsBlank(ElementText{Size: 12}) {
		t.Error("IsBlank(ElementText{}) = true, want false")
	}
}

func TestResolveText_Inheritance(t *testing.T) {
	t.Parallel()

	th := Theme{
		Elements: map[string]Element{
			"text":         ElementText{Family: "serif", Size: 12, Color: color.Black},
			"axis.title":   ElementText{Size: 14, Bold: true},
			"axis.title.x": ElementText{Italic: true},
		},
	}

	// axis.title.x should inherit Size=14 from axis.title, Family/Color from text.
	got := th.AxisTitleX()

	if got.Family != "serif" {
		t.Errorf("Family = %q, want %q", got.Family, "serif")
	}

	if got.Size != 14 {
		t.Errorf("Size = %v, want 14", got.Size)
	}

	if got.Color != color.Black {
		t.Errorf("Color = %v, want black", got.Color)
	}

	if !got.Italic {
		t.Error("Italic = false, want true")
	}
}

func TestResolveText_Blank(t *testing.T) {
	t.Parallel()

	th := Theme{
		Elements: map[string]Element{
			"text":       ElementText{Family: "serif", Size: 12, Color: color.Black},
			"axis.title": ElementBlank{},
		},
	}

	// axis.title is blank — should return zero ElementText.
	got := th.AxisTitle()

	if got.Size != 0 {
		t.Errorf("blanked element Size = %v, want 0", got.Size)
	}

	if got.Family != "" {
		t.Errorf("blanked element Family = %q, want empty", got.Family)
	}
}

func TestResolveLine_Inheritance(t *testing.T) {
	t.Parallel()

	th := Theme{
		Elements: map[string]Element{
			"line":             ElementLine{Color: gray(100), Size: 1},
			"panel.grid.major": ElementLine{Size: 2},
		},
	}

	got := th.PanelGridMajor()

	if got.Color != gray(100) {
		t.Errorf("Color = %v, want gray(100)", got.Color)
	}

	if got.Size != 2 {
		t.Errorf("Size = %v, want 2", got.Size)
	}
}

func TestResolveRect_Inheritance(t *testing.T) {
	t.Parallel()

	th := Theme{
		Elements: map[string]Element{
			"rect":             ElementRect{Fill: color.White, Color: color.Black, Size: 1},
			"panel.background": ElementRect{Fill: color.RGBA{R: 230, G: 230, B: 230, A: 255}},
		},
	}

	got := th.PanelBackground()

	if got.Color != color.Black {
		t.Errorf("Color = %v, want black (inherited from rect)", got.Color)
	}

	if got.Size != 1 {
		t.Errorf("Size = %v, want 1 (inherited from rect)", got.Size)
	}
}

// ---------------------------------------------------------------------------
// Comprehensive theme inheritance proof tests
// ---------------------------------------------------------------------------

// TestInheritanceProof_TextChains verifies every text inheritance path
// in the hierarchy tree. Each test sets only the root "text" and one
// mid-chain element, then asserts that the leaf correctly inherits
// through the chain.
func TestInheritanceProof_TextChains(t *testing.T) {
	t.Parallel()

	rootText := ElementText{Family: "serif", Size: 12, Color: color.Black, LineHeight: 1.4}

	tests := []struct {
		name        string
		midPath     string
		midElem     ElementText
		leafPath    string
		resolveFunc func(Theme) ElementText
		wantFamily  string
		wantSize    float64
		wantColor   color.Color
	}{
		{
			name:    "axis.text.x ← axis.text ← text",
			midPath: "axis.text", midElem: ElementText{Size: 10},
			leafPath:    "axis.text.x",
			resolveFunc: func(th Theme) ElementText { return th.resolveText("axis.text.x") },
			wantFamily:  "serif", wantSize: 10, wantColor: color.Black,
		},
		{
			name:    "axis.text.y ← axis.text ← text",
			midPath: "axis.text", midElem: ElementText{Size: 10, Color: gray(60)},
			leafPath:    "axis.text.y",
			resolveFunc: func(th Theme) ElementText { return th.resolveText("axis.text.y") },
			wantFamily:  "serif", wantSize: 10, wantColor: gray(60),
		},
		{
			name:    "axis.title.x ← axis.title ← text",
			midPath: "axis.title", midElem: ElementText{Size: 14, Bold: true},
			leafPath:    "axis.title.x",
			resolveFunc: func(th Theme) ElementText { return th.AxisTitleX() },
			wantFamily:  "serif", wantSize: 14, wantColor: color.Black,
		},
		{
			name:    "axis.title.y ← axis.title ← text",
			midPath: "axis.title", midElem: ElementText{Size: 14},
			leafPath:    "axis.title.y",
			resolveFunc: func(th Theme) ElementText { return th.AxisTitleY() },
			wantFamily:  "serif", wantSize: 14, wantColor: color.Black,
		},
		{
			name:    "strip.text.x ← strip.text ← text",
			midPath: "strip.text", midElem: ElementText{Size: 9, Bold: true},
			leafPath:    "strip.text.x",
			resolveFunc: func(th Theme) ElementText { return th.StripTextX() },
			wantFamily:  "serif", wantSize: 9, wantColor: color.Black,
		},
		{
			name:    "strip.text.y ← strip.text ← text",
			midPath: "strip.text", midElem: ElementText{Size: 9},
			leafPath:    "strip.text.y",
			resolveFunc: func(th Theme) ElementText { return th.StripTextY() },
			wantFamily:  "serif", wantSize: 9, wantColor: color.Black,
		},
		{
			name:    "legend.title ← text (direct child)",
			midPath: "", midElem: ElementText{},
			leafPath:    "legend.title",
			resolveFunc: func(th Theme) ElementText { return th.LegendTitle() },
			wantFamily:  "serif", wantSize: 12, wantColor: color.Black,
		},
		{
			name:    "legend.text ← text (direct child)",
			midPath: "", midElem: ElementText{},
			leafPath:    "legend.text",
			resolveFunc: func(th Theme) ElementText { return th.LegendTextElem() },
			wantFamily:  "serif", wantSize: 12, wantColor: color.Black,
		},
		{
			name:    "annotation.text ← text (direct child)",
			midPath: "", midElem: ElementText{},
			leafPath:    "annotation.text",
			resolveFunc: func(th Theme) ElementText { return th.AnnotationText() },
			wantFamily:  "serif", wantSize: 12, wantColor: color.Black,
		},
		{
			name:    "plot.title ← text (direct child)",
			midPath: "", midElem: ElementText{},
			leafPath:    "plot.title",
			resolveFunc: func(th Theme) ElementText { return th.PlotTitle() },
			wantFamily:  "serif", wantSize: 12, wantColor: color.Black,
		},
		{
			name:    "plot.subtitle ← text (direct child)",
			midPath: "", midElem: ElementText{},
			leafPath:    "plot.subtitle",
			resolveFunc: func(th Theme) ElementText { return th.PlotSubtitle() },
			wantFamily:  "serif", wantSize: 12, wantColor: color.Black,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			elems := map[string]Element{
				"text": rootText,
			}
			if tc.midPath != "" {
				elems[tc.midPath] = tc.midElem
			}

			th := Theme{Elements: elems}
			got := tc.resolveFunc(th)

			if got.Family != tc.wantFamily {
				t.Errorf("Family = %q, want %q", got.Family, tc.wantFamily)
			}

			if got.Size != tc.wantSize {
				t.Errorf("Size = %v, want %v", got.Size, tc.wantSize)
			}

			if got.Color != tc.wantColor {
				t.Errorf("Color = %v, want %v", got.Color, tc.wantColor)
			}
		})
	}
}

// TestInheritanceProof_LineChains verifies every line inheritance path.
func TestInheritanceProof_LineChains(t *testing.T) {
	t.Parallel()

	rootLine := ElementLine{Color: gray(100), Size: 1, Lineend: "round"}

	tests := []struct {
		name        string
		midPath     string
		midElem     ElementLine
		resolveFunc func(Theme) ElementLine
		wantColor   color.Color
		wantSize    float64
	}{
		{
			name:    "axis.ticks.x ← axis.ticks ← line",
			midPath: "axis.ticks", midElem: ElementLine{Size: 2},
			resolveFunc: func(th Theme) ElementLine { return th.resolveLine("axis.ticks.x") },
			wantColor:   gray(100), wantSize: 2,
		},
		{
			name:    "axis.ticks.y ← axis.ticks ← line",
			midPath: "axis.ticks", midElem: ElementLine{Size: 2},
			resolveFunc: func(th Theme) ElementLine { return th.resolveLine("axis.ticks.y") },
			wantColor:   gray(100), wantSize: 2,
		},
		{
			name:    "axis.line.x ← axis.line ← line",
			midPath: "axis.line", midElem: ElementLine{Color: gray(50)},
			resolveFunc: func(th Theme) ElementLine { return th.resolveLine("axis.line.x") },
			wantColor:   gray(50), wantSize: 1,
		},
		{
			name:    "axis.line.y ← axis.line ← line",
			midPath: "axis.line", midElem: ElementLine{Color: gray(50)},
			resolveFunc: func(th Theme) ElementLine { return th.resolveLine("axis.line.y") },
			wantColor:   gray(50), wantSize: 1,
		},
		{
			name:        "panel.grid.major ← line (direct child)",
			midPath:     "",
			resolveFunc: func(th Theme) ElementLine { return th.PanelGridMajor() },
			wantColor:   gray(100), wantSize: 1,
		},
		{
			name:        "panel.grid.minor ← line (direct child)",
			midPath:     "",
			resolveFunc: func(th Theme) ElementLine { return th.PanelGridMinor() },
			wantColor:   gray(100), wantSize: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			elems := map[string]Element{
				"line": rootLine,
			}
			if tc.midPath != "" {
				elems[tc.midPath] = tc.midElem
			}

			th := Theme{Elements: elems}
			got := tc.resolveFunc(th)

			if got.Color != tc.wantColor {
				t.Errorf("Color = %v, want %v", got.Color, tc.wantColor)
			}

			if got.Size != tc.wantSize {
				t.Errorf("Size = %v, want %v", got.Size, tc.wantSize)
			}
		})
	}
}

// TestInheritanceProof_RectChains verifies every rect inheritance path.
func TestInheritanceProof_RectChains(t *testing.T) {
	t.Parallel()

	rootRect := ElementRect{Fill: color.White, Color: color.Black, Size: 1}

	tests := []struct {
		name        string
		midPath     string
		midElem     ElementRect
		resolveFunc func(Theme) ElementRect
		wantFill    color.Color
		wantColor   color.Color
		wantSize    float64
	}{
		{
			name:        "panel.background ← rect",
			resolveFunc: func(th Theme) ElementRect { return th.PanelBackground() },
			wantFill:    color.White, wantColor: color.Black, wantSize: 1,
		},
		{
			name:        "panel.border ← rect",
			resolveFunc: func(th Theme) ElementRect { return th.PanelBorder() },
			wantFill:    color.White, wantColor: color.Black, wantSize: 1,
		},
		{
			name:        "plot.background ← rect",
			resolveFunc: func(th Theme) ElementRect { return th.PlotBackground() },
			wantFill:    color.White, wantColor: color.Black, wantSize: 1,
		},
		{
			name:        "legend.background ← rect",
			resolveFunc: func(th Theme) ElementRect { return th.LegendBackground() },
			wantFill:    color.White, wantColor: color.Black, wantSize: 1,
		},
		{
			name:        "legend.key ← rect",
			resolveFunc: func(th Theme) ElementRect { return th.LegendKey() },
			wantFill:    color.White, wantColor: color.Black, wantSize: 1,
		},
		{
			name:    "strip.background.x ← strip.background ← rect",
			midPath: "strip.background", midElem: ElementRect{Fill: gray(240)},
			resolveFunc: func(th Theme) ElementRect { return th.StripBackgroundX() },
			wantFill:    gray(240), wantColor: color.Black, wantSize: 1,
		},
		{
			name:    "strip.background.y ← strip.background ← rect",
			midPath: "strip.background", midElem: ElementRect{Fill: gray(240)},
			resolveFunc: func(th Theme) ElementRect { return th.StripBackgroundY() },
			wantFill:    gray(240), wantColor: color.Black, wantSize: 1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			elems := map[string]Element{
				"rect": rootRect,
			}
			if tc.midPath != "" {
				elems[tc.midPath] = tc.midElem
			}

			th := Theme{Elements: elems}
			got := tc.resolveFunc(th)

			if got.Fill != tc.wantFill {
				t.Errorf("Fill = %v, want %v", got.Fill, tc.wantFill)
			}

			if got.Color != tc.wantColor {
				t.Errorf("Color = %v, want %v", got.Color, tc.wantColor)
			}

			if got.Size != tc.wantSize {
				t.Errorf("Size = %v, want %v", got.Size, tc.wantSize)
			}
		})
	}
}

// TestInheritanceProof_BlankSuppression verifies that ElementBlank at
// various positions in the chain correctly suppresses rendering.
func TestInheritanceProof_BlankSuppression(t *testing.T) {
	t.Parallel()

	t.Run("blank at mid-chain suppresses leaf", func(t *testing.T) {
		t.Parallel()

		th := Theme{
			Elements: map[string]Element{
				"text":       ElementText{Family: "serif", Size: 12, Color: color.Black},
				"axis.title": ElementBlank{},
			},
		}

		// axis.title.x should get zero value because axis.title is blank.
		got := th.AxisTitleX()
		if got.Size != 0 || got.Family != "" {
			t.Errorf("expected zero ElementText, got Size=%v Family=%q", got.Size, got.Family)
		}

		// axis.title.y should also be blank (inherits from blanked axis.title).
		got2 := th.AxisTitleY()
		if got2.Size != 0 || got2.Family != "" {
			t.Errorf("expected zero ElementText for axis.title.y, got Size=%v Family=%q", got2.Size, got2.Family)
		}
	})

	t.Run("blank at leaf does not affect sibling", func(t *testing.T) {
		t.Parallel()

		th := Theme{
			Elements: map[string]Element{
				"text":         ElementText{Family: "sans-serif", Size: 11, Color: color.Black},
				"axis.title":   ElementText{Size: 14},
				"axis.title.x": ElementBlank{},
			},
		}

		// axis.title.x should be blank (zero).
		gotX := th.AxisTitleX()
		if gotX.Size != 0 {
			t.Errorf("blanked axis.title.x Size = %v, want 0", gotX.Size)
		}

		// axis.title.y should inherit normally from axis.title.
		gotY := th.AxisTitleY()
		if gotY.Size != 14 {
			t.Errorf("axis.title.y Size = %v, want 14", gotY.Size)
		}

		if gotY.Family != "sans-serif" {
			t.Errorf("axis.title.y Family = %q, want %q", gotY.Family, "sans-serif")
		}
	})

	t.Run("blank line element", func(t *testing.T) {
		t.Parallel()

		th := Theme{
			Elements: map[string]Element{
				"line":       ElementLine{Color: gray(100), Size: 1},
				"axis.ticks": ElementBlank{},
			},
		}

		got := th.AxisTicks()
		if got.Size != 0 {
			t.Errorf("blanked axis.ticks Size = %v, want 0", got.Size)
		}

		// axis.ticks.x inherits from blanked axis.ticks.
		gotX := th.resolveLine("axis.ticks.x")
		if gotX.Size != 0 {
			t.Errorf("axis.ticks.x through blanked parent Size = %v, want 0", gotX.Size)
		}
	})

	t.Run("blank rect element", func(t *testing.T) {
		t.Parallel()

		th := Theme{
			Elements: map[string]Element{
				"rect":             ElementRect{Fill: color.White, Color: color.Black, Size: 1},
				"strip.background": ElementBlank{},
			},
		}

		got := th.StripBackground()
		if got.Size != 0 {
			t.Errorf("blanked strip.background Size = %v, want 0", got.Size)
		}

		gotX := th.StripBackgroundX()
		if gotX.Size != 0 {
			t.Errorf("strip.background.x through blanked parent Size = %v, want 0", gotX.Size)
		}
	})
}

// TestInheritanceProof_OverrideComposition verifies that WithOverrides
// correctly replaces mid-chain elements and children still inherit.
func TestInheritanceProof_OverrideComposition(t *testing.T) {
	t.Parallel()

	base := Theme{
		Elements: map[string]Element{
			"text":       ElementText{Family: "serif", Size: 12, Color: color.Black},
			"axis.title": ElementText{Size: 14},
		},
	}

	// Override axis.title with a new size and bold.
	overridden := WithOverrides(base,
		Override{Path: "axis.title", Elem: ElementText{Size: 16, Bold: true}},
	)

	// axis.title should now have Size=16, Bold=true, inherit Family/Color from text.
	got := overridden.AxisTitle()
	if got.Size != 16 {
		t.Errorf("overridden axis.title Size = %v, want 16", got.Size)
	}

	if !got.Bold {
		t.Error("overridden axis.title Bold = false, want true")
	}

	if got.Family != "serif" {
		t.Errorf("overridden axis.title Family = %q, want %q (inherited from text)", got.Family, "serif")
	}

	// axis.title.x should now inherit Size=16 from the overridden axis.title.
	gotX := overridden.AxisTitleX()
	if gotX.Size != 16 {
		t.Errorf("axis.title.x after parent override Size = %v, want 16", gotX.Size)
	}

	// Original theme should NOT be affected (shallow copy check).
	origTitle := base.AxisTitle()
	if origTitle.Size != 14 {
		t.Errorf("original axis.title Size = %v, want 14 (should be unmodified)", origTitle.Size)
	}
}

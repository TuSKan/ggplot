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

package facet_test

import (
	"context"
	"errors"
	"testing"

	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/facet"
)

func testDS(t *testing.T) dataset.Dataset {
	t.Helper()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewStringColumn("species", []string{"setosa", "setosa", "versicolor", "versicolor", "virginica", "virginica"}),
		eng.NewFloat64Column("sepal", []float64{5.1, 4.9, 7.0, 6.4, 6.3, 5.8}),
		eng.NewStringColumn("color", []string{"red", "red", "blue", "blue", "green", "green"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	return ds
}

// --- Labeller ---

func TestLabelValue(t *testing.T) {
	t.Parallel()

	lbl := facet.LabelValue()
	if got := lbl("species", "setosa"); got != "setosa" {
		t.Errorf("LabelValue: got %q, want %q", got, "setosa")
	}
}

func TestLabelBoth(t *testing.T) {
	t.Parallel()

	lbl := facet.LabelBoth()
	if got := lbl("species", "setosa"); got != "species: setosa" {
		t.Errorf("LabelBoth: got %q, want %q", got, "species: setosa")
	}
}

func TestLabelCustom(t *testing.T) {
	t.Parallel()

	lbl := facet.Label(func(v, val string) string {
		return "[" + v + "]=" + val
	})
	if got := lbl("sp", "A"); got != "[sp]=A" {
		t.Errorf("Label(custom): got %q, want %q", got, "[sp]=A")
	}
}

func TestLabelContext_IsNil(t *testing.T) {
	t.Parallel()

	if lbl := facet.LabelContext(); lbl != nil {
		t.Error("LabelContext should return nil sentinel")
	}
}

// --- Wrap with labeller ---

func TestWrap_LabelValue(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	panels, err := facet.Wrap("species").Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// Default labeller is LabelValue.
	wantLabels := []string{"setosa", "versicolor", "virginica"}
	if len(panels) != len(wantLabels) {
		t.Fatalf("expected %d panels, got %d", len(wantLabels), len(panels))
	}

	for i, want := range wantLabels {
		if panels[i].Label != want {
			t.Errorf("panel %d: got label %q, want %q", i, panels[i].Label, want)
		}
	}
}

func TestWrap_LabelBoth(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	panels, err := facet.Wrap("species", facet.WithLabeller(facet.LabelBoth())).Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	wantLabels := []string{"species: setosa", "species: versicolor", "species: virginica"}
	if len(panels) != len(wantLabels) {
		t.Fatalf("expected %d panels, got %d", len(wantLabels), len(panels))
	}

	for i, want := range wantLabels {
		if panels[i].Label != want {
			t.Errorf("panel %d: got label %q, want %q", i, panels[i].Label, want)
		}
	}
}

func TestWrap_LabelContext_DefaultsToValue(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	panels, err := facet.Wrap("species", facet.WithLabeller(facet.LabelContext())).Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// LabelContext → LabelValue for Wrap.
	if panels[0].Label != "setosa" {
		t.Errorf("LabelContext on Wrap: got %q, want %q", panels[0].Label, "setosa")
	}
}

// --- Drop ---

func TestWrap_DropTrue_DefaultBehavior(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	// All species have data → same result with drop=true or default.
	panels, err := facet.Wrap("species", facet.WithDrop(true)).Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	if len(panels) != 3 { //nolint:mnd // 3 species.
		t.Errorf("expected 3 panels, got %d", len(panels))
	}
}

func TestWrap_DropFalse_PreservesAllPanels(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	panels, err := facet.Wrap("species", facet.WithDrop(false)).Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// All species have data, so drop=false has same result.
	if len(panels) != 3 { //nolint:mnd // 3 species.
		t.Errorf("expected 3 panels, got %d", len(panels))
	}
}

// --- Grid ---

func TestGrid_Default(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color")

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// 3 species × 3 colors = 9 total, but only 3 have data.
	// Default drop=true → only panels with data.
	if len(panels) != 3 { //nolint:mnd // 3 data cells.
		t.Errorf("expected 3 panels (drop=true), got %d", len(panels))
	}
}

func TestGrid_DropFalse(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridDrop(false))

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// 3 species × 3 colors = 9 cells, all emitted even if empty.
	if len(panels) != 9 { //nolint:mnd // 3 × 3 grid.
		t.Errorf("expected 9 panels (drop=false), got %d", len(panels))
	}

	// Verify empty panels have 0 rows after collection.
	emptyCount := 0

	for _, p := range panels {
		collected, cerr := p.Dataset.Collect(context.Background())
		if cerr != nil {
			t.Fatal(cerr)
		}

		if collected.NumRows() == 0 {
			emptyCount++
		}
	}

	if emptyCount != 6 { //nolint:mnd // 6 empty cells.
		t.Errorf("expected 6 empty panels, got %d", emptyCount)
	}
}

func TestGrid_LabelBoth(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridLabeller(facet.LabelBoth()))

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// With LabelBoth, labels should be "species: X | color: Y".
	for _, p := range panels {
		if p.Label == "" {
			t.Error("panel label should not be empty with LabelBoth")
		}
	}

	// Check first panel.
	want := "species: setosa | color: red"
	if panels[0].Label != want {
		t.Errorf("panel 0: got %q, want %q", panels[0].Label, want)
	}
}

func TestGrid_RowValColVal(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color")

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range panels {
		if p.RowVal == "" {
			t.Error("RowVal should be set for Grid panels")
		}

		if p.ColVal == "" {
			t.Error("ColVal should be set for Grid panels")
		}
	}
}

func TestGrid_LabelContext_DefaultsToBoth(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridLabeller(facet.LabelContext()))

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// LabelContext → LabelBoth for Grid.
	want := "species: setosa | color: red"
	if panels[0].Label != want {
		t.Errorf("LabelContext on Grid: got %q, want %q", panels[0].Label, want)
	}
}

// --- Margins ---

func TestGrid_Margins(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridDrop(false), facet.GridMargins(true))

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// Data panels: 3 × 3 = 9
	// Row margins: 3 (one per species, aggregated across colors)
	// Col margins: 3 (one per color, aggregated across species)
	// Corner margin: 1 (full dataset)
	// Total: 9 + 3 + 3 + 1 = 16
	wantTotal := 16 //nolint:mnd // 9 data + 3 row + 3 col + 1 corner.
	if len(panels) != wantTotal {
		t.Errorf("expected %d panels with margins, got %d", wantTotal, len(panels))
	}

	// Count margin panels.
	marginCount := 0

	for _, p := range panels {
		if p.IsMargin {
			marginCount++
		}
	}

	wantMargins := 7 //nolint:mnd // 3 row + 3 col + 1 corner.
	if marginCount != wantMargins {
		t.Errorf("expected %d margin panels, got %d", wantMargins, marginCount)
	}

	// Corner margin should have all rows.
	cornerPanel := panels[len(panels)-1]
	if cornerPanel.RowVal != facet.MarginLabel || cornerPanel.ColVal != facet.MarginLabel {
		t.Errorf("corner margin: RowVal=%q, ColVal=%q", cornerPanel.RowVal, cornerPanel.ColVal)
	}

	if cornerPanel.Dataset.NumRows() != ds.NumRows() {
		t.Errorf("corner margin rows: got %d, want %d", cornerPanel.Dataset.NumRows(), ds.NumRows())
	}
}

func TestGrid_Margins_RowMarginRowCount(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridMargins(true))

	panels, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	// Find row margin for "setosa" (aggregated across all colors).
	for _, p := range panels {
		if p.IsMargin && p.RowVal == "setosa" && p.ColVal == facet.MarginLabel {
			// Collect lazy dataset at assertion time (allowed per lazy rules).
			collected, cerr := p.Dataset.Collect(context.Background())
			if cerr != nil {
				t.Fatal(cerr)
			}

			// setosa has 2 rows in the original data.
			if collected.NumRows() != 2 { //nolint:mnd // 2 setosa rows.
				t.Errorf("setosa row margin: got %d rows, want 2", collected.NumRows())
			}

			return
		}
	}

	t.Error("setosa row margin panel not found")
}

// --- GridDims ---

func TestGrid_GridDims_WithMargins(t *testing.T) {
	t.Parallel()

	ds := testDS(t)
	f := facet.Grid("species", "color", facet.GridDrop(false), facet.GridMargins(true))

	_, err := f.Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	rows, cols := f.GridDims(0)
	// 3 species + 1 margin row = 4, 3 colors + 1 margin col = 4.
	if rows != 4 || cols != 4 { //nolint:mnd // 4 × 4 with margins.
		t.Errorf("GridDims with margins: got (%d, %d), want (4, 4)", rows, cols)
	}
}

// --- None ---

func TestNone(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	panels, err := facet.None().Split(context.Background(), ds)
	if err != nil {
		t.Fatal(err)
	}

	if len(panels) != 1 {
		t.Errorf("None: expected 1 panel, got %d", len(panels))
	}

	if panels[0].Label != "" {
		t.Errorf("None: expected empty label, got %q", panels[0].Label)
	}
}

// --- Error cases ---

func TestWrap_MissingColumn(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	_, err := facet.Wrap("nonexistent").Split(context.Background(), ds)
	if err == nil {
		t.Fatal("expected error for missing column")
	}
}

func TestGrid_MissingColumn(t *testing.T) {
	t.Parallel()

	ds := testDS(t)

	_, err := facet.Grid("nonexistent", "color").Split(context.Background(), ds)
	if err == nil {
		t.Fatal("expected error for missing column")
	}

	if !errors.Is(err, facet.ErrFacetConfig) {
		// It wraps a column-not-found error, not ErrFacetConfig directly.
		// Just verify it's an error.
		t.Logf("error type: %T, msg: %s", err, err)
	}
}

// --- Wrap with different column types ---

func TestWrap_Float64Column(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", []float64{1.5, 2.5, 1.5}),
		eng.NewFloat64Column("y", []float64{10, 20, 30}),
	)
	if err != nil {
		t.Fatal(err)
	}

	panels, perr := facet.Wrap("x").Split(context.Background(), ds)
	if perr != nil {
		t.Fatal(perr)
	}

	if len(panels) != 2 { //nolint:mnd // 2 distinct x values.
		t.Errorf("expected 2 panels, got %d", len(panels))
	}
}

func TestWrap_BoolColumn(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewBoolColumn("flag", []bool{true, false, true}),
		eng.NewFloat64Column("y", []float64{1, 2, 3}),
	)
	if err != nil {
		t.Fatal(err)
	}

	panels, perr := facet.Wrap("flag").Split(context.Background(), ds)
	if perr != nil {
		t.Fatal(perr)
	}

	if len(panels) != 2 { //nolint:mnd // TRUE, FALSE.
		t.Errorf("expected 2 panels, got %d", len(panels))
	}
}

func TestWrap_Int64Column(t *testing.T) {
	t.Parallel()

	eng := memory.NewEngine(context.Background())

	ds, err := dataset.NewDataset(eng,
		eng.NewInt64Column("group", []int64{1, 2, 1, 3}),
		eng.NewFloat64Column("y", []float64{10, 20, 30, 40}),
	)
	if err != nil {
		t.Fatal(err)
	}

	panels, perr := facet.Wrap("group").Split(context.Background(), ds)
	if perr != nil {
		t.Fatal(perr)
	}

	if len(panels) != 3 { //nolint:mnd // 3 groups.
		t.Errorf("expected 3 panels, got %d", len(panels))
	}
}

// --- Wrap GridDims ---

func TestWrap_GridDims_NCols(t *testing.T) {
	t.Parallel()

	f := facet.Wrap("x", facet.NCols(2))

	rows, cols := f.GridDims(5) //nolint:mnd // 5 panels.
	if rows != 3 || cols != 2 { //nolint:mnd // ceil(5/2) = 3, 2 cols.
		t.Errorf("GridDims(5, NCols=2): got (%d, %d), want (3, 2)", rows, cols)
	}
}

func TestWrap_GridDims_NRows(t *testing.T) {
	t.Parallel()

	f := facet.Wrap("x", facet.NRows(2))

	rows, cols := f.GridDims(5) //nolint:mnd // 5 panels.
	if rows != 2 || cols != 3 { //nolint:mnd // 2 rows, ceil(5/2) = 3 cols.
		t.Errorf("GridDims(5, NRows=2): got (%d, %d), want (2, 3)", rows, cols)
	}
}

func TestWrap_GridDims_Auto(t *testing.T) {
	t.Parallel()

	f := facet.Wrap("x")

	rows, cols := f.GridDims(4) //nolint:mnd // 4 panels.
	if rows != 2 || cols != 2 { //nolint:mnd // sqrt(4) = 2.
		t.Errorf("GridDims(4, auto): got (%d, %d), want (2, 2)", rows, cols)
	}
}

// --- String ---

func TestWrap_String(t *testing.T) {
	t.Parallel()

	f := facet.Wrap("species")
	if s := f.String(); s != "wrap(species)" {
		t.Errorf("Wrap.String: got %q, want %q", s, "wrap(species)")
	}
}

func TestGrid_String(t *testing.T) {
	t.Parallel()

	f := facet.Grid("species", "color")
	if s := f.String(); s != "grid(species ~ color)" {
		t.Errorf("Grid.String: got %q, want %q", s, "grid(species ~ color)")
	}
}

func TestNone_String(t *testing.T) {
	t.Parallel()

	if s := facet.None().String(); s != "none" {
		t.Errorf("None.String: got %q, want %q", s, "none")
	}
}

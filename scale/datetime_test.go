package scale_test

import (
	"math"
	"testing"
	"time"

	"github.com/TuSKan/ggplot/scale"
)

// setBounds is a test helper that sets bounds on a BoundsSetter.
func setBounds(t *testing.T, s scale.Scale, mn, mx float64) {
	t.Helper()

	bs, ok := s.(scale.BoundsSetter)
	if !ok {
		t.Fatal("Scale does not implement BoundsSetter")
	}

	bs.SetBounds(mn, mx)
}

func TestDateTimeScale_MapInverse(t *testing.T) {
	t.Parallel()

	sc := scale.NewDateTimeIn(time.UTC)
	setBounds(t, sc, 0, 1e18) // ~11.6 days in ns

	// Midpoint should map to 0.5.
	mid := 5e17
	got := sc.Map(mid)

	if math.Abs(got-0.5) > 1e-10 {
		t.Errorf("Map(%v) = %v, want 0.5", mid, got)
	}

	// Round-trip.
	back := sc.Inverse(got)
	if math.Abs(back-mid) > 1 {
		t.Errorf("Inverse(Map(%v)) = %v, diff = %v", mid, back, back-mid)
	}
}

func TestDateTimeScale_SinglePoint(t *testing.T) {
	t.Parallel()

	sc := scale.NewDateTimeIn(time.UTC)
	setBounds(t, sc, 100, 100) // single point

	got := sc.Map(100)
	if got != 0.5 {
		t.Errorf("Map(single-point) = %v, want 0.5", got)
	}
}

func TestDateTimeScale_Ticks_YearSpan(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC)

	sc := scale.NewDateTimeIn(time.UTC)
	setBounds(t, sc, float64(t0.UnixNano()), float64(t1.UnixNano()))

	ticks := sc.Ticks(5)
	if len(ticks) == 0 {
		t.Fatal("Ticks returned empty slice")
	}

	// All ticks should be within bounds.
	lo, hi := sc.Bounds()
	for i, v := range ticks {
		if v < lo || v > hi {
			t.Errorf("tick[%d] = %v out of bounds [%v, %v]", i, v, lo, hi)
		}
	}
}

func TestDateTimeScale_Ticks_HourSpan(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2024, 6, 15, 8, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 6, 15, 20, 0, 0, 0, time.UTC)

	sc := scale.NewDateTimeIn(time.UTC)
	setBounds(t, sc, float64(t0.UnixNano()), float64(t1.UnixNano()))

	ticks := sc.Ticks(5)
	if len(ticks) < 2 {
		t.Fatalf("expected at least 2 ticks for 12h span, got %d", len(ticks))
	}
}

func TestDateTimeScale_Format(t *testing.T) {
	t.Parallel()

	t0 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)

	sc := scale.NewDateTimeIn(time.UTC)
	setBounds(t, sc, float64(t0.UnixNano()), float64(t1.UnixNano()))

	// Trigger layout detection.
	_ = sc.Ticks(5)

	label := sc.Format(float64(t0.UnixNano()))
	if label == "" {
		t.Error("Format returned empty string")
	}
}

func TestDateTimeScale_String(t *testing.T) {
	t.Parallel()

	sc := scale.NewDateTime()
	if sc.String() != "datetime" {
		t.Errorf("String() = %q, want %q", sc.String(), "datetime")
	}
}

func TestDateTimeScale_MapInverse_MultiYear(t *testing.T) {
	t.Parallel()

	sc := scale.NewDateTimeIn(time.UTC)
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	setBounds(t, sc, float64(t0.UnixNano()), float64(t1.UnixNano()))

	mid := float64(time.Date(2022, 7, 1, 0, 0, 0, 0, time.UTC).UnixNano())

	mapped := sc.Map(mid)
	recovered := sc.Inverse(mapped)

	diff := recovered - mid
	if diff > 1e6 || diff < -1e6 { // within 1ms
		t.Errorf("Map/Inverse round-trip diff = %v (want < 1ms)", diff)
	}
}

func TestResolve_DateTime(t *testing.T) {
	t.Parallel()

	sc, err := scale.Resolve(scale.DateTime)
	if err != nil {
		t.Fatalf("Resolve(DateTime): %v", err)
	}

	if sc.String() != "datetime" {
		t.Errorf("Resolve(DateTime).String() = %q", sc.String())
	}
}

func TestResolve_Binned(t *testing.T) {
	t.Parallel()

	sc, err := scale.Resolve(scale.Binned)
	if err != nil {
		t.Fatalf("Resolve(Binned): %v", err)
	}

	if sc.String() != "binned" {
		t.Errorf("Resolve(Binned).String() = %q", sc.String())
	}
}

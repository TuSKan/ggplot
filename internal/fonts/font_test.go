package fonts

import (
	"sync"
	"testing"
)

func TestScoreAndTiebreak(t *testing.T) {
	q := Query{Family: "arial", Weight: WeightNormal, Style: StyleNormal, PreferMonospace: false}

	f1 := Font{Family: "arial", Weight: WeightNormal, Style: StyleNormal, FullName: "Arial B", Path: "/a"}
	f5 := Font{Family: "arial", Weight: WeightNormal, Style: StyleNormal, FullName: "Arial A", Path: "/c"}

	r := &Registry{
		fonts: []Font{f1, f5},
	}

	best := r.Match(q)

	if best.FullName != "Arial A" {
		t.Errorf("expected deterministic tie-break to prefer alphabetically exact 'Arial A', got %v", best.FullName)
	}
}

func TestResolverFallbackChains(t *testing.T) {
	reg := &Registry{
		fonts: []Font{
			{Family: "liberation sans", Weight: WeightNormal},
			{Family: "dejavu serif", Weight: WeightNormal},
		},
	}

	res := NewResolver(reg, DefaultFallbackConfig())

	// 1. Missing exact requested family mappings picking directly the first matching generic mapped cascade.
	q1 := Query{Family: "unknown", AllowFallback: true}
	f1 := res.Resolve(q1)
	if f1 == nil || f1.Family != "liberation sans" {
		t.Errorf("expected fallback generic chain to pick generic mapped liberation sans.")
	}

	// 2. Hits an configured aliasing mechanism missing triggering standard defaults logic.
	q2 := Query{Family: "helvetica", AllowFallback: false}
	f2 := res.Resolve(q2)
	if f2 == nil || f2.Family != "liberation sans" {
		t.Errorf("expected near structural alias array mapping identifying liberation sans mapping overrides ")
	}

	// 3. Blocks queries without breaking cascades dropping structural variables completely missing configurations.
	q3 := Query{Family: "comic sans", AllowFallback: false} // we don't have comic sans ms mapped
	f3 := res.Resolve(q3)
	if f3 != nil {
		t.Errorf("expected nil result blocking generic empty cascade mappings ")
	}

	// 4. Missing mapped style variants resolves exact identical strings ignoring identical generic elements defaulting cascade rules.
	regMono := &Registry{
		fonts: []Font{
			{Family: "menlo", Weight: WeightNormal},
			{Family: "ubuntu", Weight: WeightBold},
		},
	}
	resMono := NewResolver(regMono, DefaultFallbackConfig())

	q4 := Query{Family: "menlo", Weight: WeightBold, AllowFallback: true}
	f4 := resMono.Resolve(q4)
	if f4 == nil || f4.Family != "menlo" {
		t.Errorf("expected logic routing exactly returning family identical variant missing styles constraints ")
	}
}

func TestWeightParsing(t *testing.T) {
	if parseWeight("Light Italic") != WeightLight {
		t.Errorf("incorrect parse for light")
	}
	if parseWeight("Black") != WeightBlack {
		t.Errorf("incorrect parse for black")
	}
}

func TestMonospaceHinting(t *testing.T) {
	qMono := Query{Family: "generic", PreferMonospace: true}
	fM := Font{Family: "generic", IsMonospace: true}
	fN := Font{Family: "generic", IsMonospace: false}

	if score(fM, qMono) <= score(fN, qMono) {
		t.Errorf("expected monospace logic preferences mapped over array variants universally tracking native elements")
	}
}

func TestParseFontFileInvalid(t *testing.T) {
	_, err := parseFontFile("fake.ttf", []byte("1234567890abcdefghijklmnopqrstuvwxyz"))
	if err == nil {
		t.Errorf("expected error when breaking corrupt string loads internally loops parsing tracking ")
	}
}

func TestConcurrentLoadFace(t *testing.T) {
	reg := &Registry{
		fonts: []Font{
			{Family: "arial", Weight: WeightNormal},
			{Family: "times", Weight: WeightNormal},
		},
	}
	res := NewResolver(reg, DefaultFallbackConfig())

	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			req := FaceRequest{
				Family:        "arial",
				Size:          float64(10 + (idx % 5)),
				DPI:           96,
				AllowFallback: true,
			}

			handle, err := res.LoadFace(req)
			if err != nil || handle == nil || handle.Font.Family != "arial" {
				t.Errorf("expected deterministic safety parsing concurrent map accesses ")
			}
		}(i)
	}

	wg.Wait()
}

func TestMeasureExtents(t *testing.T) {
	// grabs a real file locally mapped bypassing mock paths ensuring sfnt unmarshals.
	reg, _ := NewRegistry()
	if len(reg.Fonts()) == 0 {
		t.Skip("Skipping robust metrics algorithms lacking OS file bounds ")
	}

	res := NewResolver(reg, DefaultFallbackConfig())

	req := FaceRequest{
		Family:        "sans-serif",
		Size:          16,
		DPI:           96,
		AllowFallback: true,
	}

	handle, err := res.LoadFace(req)
	if err != nil {
		t.Fatalf("Failed to resolve fallback : %v", err)
	}

	// 1. Empty strings shouldn't crash arrays
	eEmpty, err := handle.MeasureExtents("")
	if err != nil {
		t.Errorf("Unexpected error mapping arrays: %v", err)
	}
	if eEmpty.Width != 0 {
		t.Errorf("Expected 0 width for empty strings!")
	}
	if eEmpty.LineHeight <= 0 {
		t.Errorf("LineHeight maps independent of string widths: %v", eEmpty.LineHeight)
	}

	// 2. ASCII geometries summing points internally
	eAscii, _ := handle.MeasureExtents("Testing bounding boxes")
	if eAscii.Width <= 0 {
		t.Errorf("Expected positive Width mapped")
	}

	// 3. Caching overrides tracking repeats
	eAsciiRepeat, _ := handle.MeasureExtents("Testing bounding boxes")
	if eAscii.Width != eAsciiRepeat.Width {
		t.Errorf("Drift tracked on parallel string mappings illegally!")
	}

	// 4. Mixed width glyphs parsing wide Japanese bounds scaled logic
	eMixed, _ := handle.MeasureExtents("Testing boundingBoxes 自動")
	if eMixed.Width <= eAscii.Width {
		t.Errorf("Expected Japanese glyph elements widening layouts")
	}
}

package ggplot

import "github.com/TuSKan/ggplot/geom"

// AnnotationType identifies the kind of annotation.
type AnnotationType string

// Supported annotation types.
const (
	AnnotationText    AnnotationType = "text"    // text label at (X, Y)
	AnnotationRect    AnnotationType = "rect"    // filled rectangle from (X, Y) to (XEnd, YEnd)
	AnnotationSegment AnnotationType = "segment" // line from (X, Y) to (XEnd, YEnd)
	AnnotationArrow   AnnotationType = "arrow"   // line with arrowhead from (X, Y) to (XEnd, YEnd)
	AnnotationLabel   AnnotationType = "label"   // text with background box at (X, Y)
)

// Annotation is a fixed-coordinate visual element that bypasses the
// data/stat/position pipeline entirely. Annotations are drawn in data space
// after all data layers so they appear on top. They do not participate in
// scale training — if their coordinates fall outside the data domain they
// are clipped (consistent with [geom.HLine] / [geom.VLine] behaviour).
type Annotation struct {
	Type AnnotationType

	// Data-space coordinates.
	// For text/label: (X, Y) is the anchor point.
	// For rect: (X, Y) is xmin/ymin and (XEnd, YEnd) is xmax/ymax.
	// For segment/arrow: (X, Y) to (XEnd, YEnd).
	X, Y       float64
	XEnd, YEnd float64

	// Text content for text and label annotations.
	Label string

	// Visual parameters. Reuses geom.Params so that all existing
	// WithColor, WithFill, WithAlpha, etc. options work unchanged.
	Params geom.Params
}

// --- Constructors ---

// AnnotateText creates a text annotation at data coordinates (x, y).
//
// Example:
//
//	p.Annotate(ggplot.AnnotateText(3.14, 1.0, "peak",
//	    geom.WithColor("#E74C3C"), geom.WithFontSize(12)))
func AnnotateText(x, y float64, label string, opts ...geom.Opt) Annotation {
	a := Annotation{
		Type:  AnnotationText,
		X:     x,
		Y:     y,
		Label: label,
	}

	applyAnnotationOpts(&a, opts)

	return a
}

// AnnotateRect creates a filled rectangle annotation spanning from
// (xmin, ymin) to (xmax, ymax) in data coordinates.
//
// Example:
//
//	p.Annotate(ggplot.AnnotateRect(1.0, 0, 3.0, 10,
//	    geom.WithFill("#FFCCCC"), geom.WithAlpha(0.3)))
func AnnotateRect(xmin, ymin, xmax, ymax float64, opts ...geom.Opt) Annotation {
	a := Annotation{
		Type: AnnotationRect,
		X:    xmin,
		Y:    ymin,
		XEnd: xmax,
		YEnd: ymax,
	}

	applyAnnotationOpts(&a, opts)

	return a
}

// AnnotateSegment creates a line segment annotation from (x, y) to
// (xend, yend) in data coordinates.
//
// Example:
//
//	p.Annotate(ggplot.AnnotateSegment(1, 5, 4, 8,
//	    geom.WithColor("#333333"), geom.WithLineWidth(1.5)))
func AnnotateSegment(x, y, xend, yend float64, opts ...geom.Opt) Annotation {
	a := Annotation{
		Type: AnnotationSegment,
		X:    x,
		Y:    y,
		XEnd: xend,
		YEnd: yend,
	}

	applyAnnotationOpts(&a, opts)

	return a
}

// AnnotateArrow creates a line segment with an arrowhead at the endpoint,
// from (x, y) to (xend, yend) in data coordinates.
//
// Example:
//
//	p.Annotate(ggplot.AnnotateArrow(1, 5, 4, 8,
//	    geom.WithColor("#333333"), geom.WithLineWidth(1.5)))
func AnnotateArrow(x, y, xend, yend float64, opts ...geom.Opt) Annotation {
	a := Annotation{
		Type: AnnotationArrow,
		X:    x,
		Y:    y,
		XEnd: xend,
		YEnd: yend,
	}

	applyAnnotationOpts(&a, opts)

	return a
}

// AnnotateLabel creates a text annotation with a filled background box
// at data coordinates (x, y). The box padding is controlled by
// [geom.WithPadding] (default 4px).
//
// Example:
//
//	p.Annotate(ggplot.AnnotateLabel(3.14, 1.0, "outlier",
//	    geom.WithFill("#FFFFFF"), geom.WithColor("#333333"),
//	    geom.WithAlpha(0.9)))
func AnnotateLabel(x, y float64, label string, opts ...geom.Opt) Annotation {
	a := Annotation{
		Type:  AnnotationLabel,
		X:     x,
		Y:     y,
		Label: label,
		Params: geom.Params{
			Padding: 4, //nolint:mnd // Default label background box padding in pixels.
		},
	}

	applyAnnotationOpts(&a, opts)

	return a
}

// --- Plot builder method ---

// Annotate adds a fixed-coordinate annotation to the plot. Annotations
// bypass the data/stat/position pipeline and are drawn after all data
// layers. They do not affect scale training.
//
// See [AnnotateText], [AnnotateRect], [AnnotateSegment], [AnnotateArrow],
// and [AnnotateLabel] for constructors.
func (p *Plot) Annotate(a Annotation) *Plot {
	cloned := p.clone()
	cloned.spec.Annotations = append(cloned.spec.Annotations, a)

	return cloned
}

// --- internal helpers ---

// applyAnnotationOpts routes geom.Opt functions through a temporary
// geom.Layer so that existing With* options (WithColor, WithFill, etc.)
// work for annotations without duplicating option definitions.
func applyAnnotationOpts(a *Annotation, opts []geom.Opt) {
	// Build a temporary layer and apply options to it.
	tmp := geom.Layer{Params: a.Params}
	for _, o := range opts {
		o(&tmp)
	}

	a.Params = tmp.Params
}

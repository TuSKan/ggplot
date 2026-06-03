// Package aes provides aesthetic mapping constructors for the Grammar of Graphics.
// Aesthetics define how data variables map to visual properties like position,
// color, size, and shape.
//
// Usage:
//
//	p := ggplot.New(ds,
//	    aes.X("col_x"),       // map column "col_x" to x-axis
//	    aes.Y("col_y"),       // map column "col_y" to y-axis
//	    aes.Color("group"),   // map column "group" to color
//	)
package aes

// Mapping binds an aesthetic channel to a data source.
type Mapping struct {
	Channel string // aesthetic name: "x", "y", "color", "fill", "size", "shape", "alpha", "group", "label", "linetype"
	Column  string // column name in the dataset
}

// X maps a column to the x-axis position aesthetic.
func X(col string) Mapping { return Mapping{Channel: "x", Column: col} }

// Y maps a column to the y-axis position aesthetic.
func Y(col string) Mapping { return Mapping{Channel: "y", Column: col} }

// Color maps a column to the color aesthetic.
func Color(col string) Mapping { return Mapping{Channel: "color", Column: col} }

// Fill maps a column to the fill color aesthetic.
func Fill(col string) Mapping { return Mapping{Channel: "fill", Column: col} }

// Size maps a column to the size aesthetic.
func Size(col string) Mapping { return Mapping{Channel: "size", Column: col} }

// Alpha maps a column to the opacity aesthetic.
func Alpha(col string) Mapping { return Mapping{Channel: "alpha", Column: col} }

// Shape maps a column to the point shape aesthetic.
func Shape(col string) Mapping { return Mapping{Channel: "shape", Column: col} }

// Linetype maps a column to the line dash pattern aesthetic.
func Linetype(col string) Mapping { return Mapping{Channel: "linetype", Column: col} }

// Group maps a column to the grouping aesthetic (for line connections, etc.).
func Group(col string) Mapping { return Mapping{Channel: "group", Column: col} }

// Label maps a column to the text label aesthetic.
func Label(col string) Mapping { return Mapping{Channel: "label", Column: col} }

// Weight maps a column to a statistical weight aesthetic.
func Weight(col string) Mapping { return Mapping{Channel: "weight", Column: col} }

// YMin maps a column to the lower bound of a range aesthetic (e.g., error bars).
func YMin(col string) Mapping { return Mapping{Channel: "ymin", Column: col} }

// YMax maps a column to the upper bound of a range aesthetic.
func YMax(col string) Mapping { return Mapping{Channel: "ymax", Column: col} }

// XMin maps a column to the left bound of a range aesthetic.
func XMin(col string) Mapping { return Mapping{Channel: "xmin", Column: col} }

// XMax maps a column to the right bound of a range aesthetic.
func XMax(col string) Mapping { return Mapping{Channel: "xmax", Column: col} }

// XEnd maps a column to the end x-position of a segment.
func XEnd(col string) Mapping { return Mapping{Channel: "xend", Column: col} }

// YEnd maps a column to the end y-position of a segment.
func YEnd(col string) Mapping { return Mapping{Channel: "yend", Column: col} }

// --- Metadata channels (SVG tooltips, links, accessibility) ---

// Title maps a column to the SVG <title> tooltip aesthetic.
// In SVG output, each primitive gets a <title> child element for hover tooltips.
// PNG and PDF output ignore this aesthetic.
func Title(col string) Mapping { return Mapping{Channel: "title", Column: col} }

// Href maps a column to the hyperlink aesthetic.
// In SVG output, primitives are wrapped in <a href="...">. PDF may emit
// link annotations. PNG output ignores this aesthetic.
func Href(col string) Mapping { return Mapping{Channel: "href", Column: col} }

// AriaLabel maps a column to the ARIA accessibility label aesthetic.
// In SVG output, primitives get an aria-label attribute for screen readers.
func AriaLabel(col string) Mapping { return Mapping{Channel: "aria_label", Column: col} }

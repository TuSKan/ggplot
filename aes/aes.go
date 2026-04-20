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
	Channel string // aesthetic name: "x", "y", "color", "fill", "size", "shape", "alpha", "group", "label"
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

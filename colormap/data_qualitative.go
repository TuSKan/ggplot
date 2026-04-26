package colormap

import "github.com/gogpu/gg"

// Qualitative palette data. All values are matplotlib / ColorBrewer reference
// RGB triplets in 0–255, converted to gg.RGBA float-space at init time via
// the rgb helper from named_colors.go.

// tab10 — matplotlib default 10-color qualitative cycle (D3 Category10).
var tab10Data = []gg.RGBA{
	rgb(31, 119, 180),  // blue
	rgb(255, 127, 14),  // orange
	rgb(44, 160, 44),   // green
	rgb(214, 39, 40),   // red
	rgb(148, 103, 189), // purple
	rgb(140, 86, 75),   // brown
	rgb(227, 119, 194), // pink
	rgb(127, 127, 127), // gray
	rgb(188, 189, 34),  // olive
	rgb(23, 190, 207),  // cyan
}

// tab20 — matplotlib 20-color cycle (Category10 paired with lighter tints).
var tab20Data = []gg.RGBA{
	rgb(31, 119, 180), rgb(174, 199, 232),
	rgb(255, 127, 14), rgb(255, 187, 120),
	rgb(44, 160, 44), rgb(152, 223, 138),
	rgb(214, 39, 40), rgb(255, 152, 150),
	rgb(148, 103, 189), rgb(197, 176, 213),
	rgb(140, 86, 75), rgb(196, 156, 148),
	rgb(227, 119, 194), rgb(247, 182, 210),
	rgb(127, 127, 127), rgb(199, 199, 199),
	rgb(188, 189, 34), rgb(219, 219, 141),
	rgb(23, 190, 207), rgb(158, 218, 229),
}

// tab20b — matplotlib 20-color "darker" variant.
var tab20bData = []gg.RGBA{
	rgb(57, 59, 121), rgb(82, 84, 163), rgb(107, 110, 207), rgb(156, 158, 222),
	rgb(99, 121, 57), rgb(140, 162, 82), rgb(181, 207, 107), rgb(206, 219, 156),
	rgb(140, 109, 49), rgb(189, 158, 57), rgb(231, 186, 82), rgb(231, 203, 148),
	rgb(132, 60, 57), rgb(173, 73, 74), rgb(214, 97, 107), rgb(231, 150, 156),
	rgb(123, 65, 115), rgb(165, 81, 148), rgb(206, 109, 189), rgb(222, 158, 214),
}

// tab20c — matplotlib 20-color "lighter" variant.
var tab20cData = []gg.RGBA{
	rgb(49, 130, 189), rgb(107, 174, 214), rgb(158, 202, 225), rgb(198, 219, 239),
	rgb(230, 85, 13), rgb(253, 141, 60), rgb(253, 174, 107), rgb(253, 208, 162),
	rgb(49, 163, 84), rgb(116, 196, 118), rgb(161, 217, 155), rgb(199, 233, 192),
	rgb(117, 107, 177), rgb(158, 154, 200), rgb(188, 189, 220), rgb(218, 218, 235),
	rgb(99, 99, 99), rgb(150, 150, 150), rgb(189, 189, 189), rgb(217, 217, 217),
}

// Set1 — ColorBrewer qualitative, 9-color.
var set1Data = []gg.RGBA{
	rgb(228, 26, 28), rgb(55, 126, 184), rgb(77, 175, 74),
	rgb(152, 78, 163), rgb(255, 127, 0), rgb(255, 255, 51),
	rgb(166, 86, 40), rgb(247, 129, 191), rgb(153, 153, 153),
}

// Set2 — ColorBrewer qualitative, 8-color, pastel/muted.
var set2Data = []gg.RGBA{
	rgb(102, 194, 165), rgb(252, 141, 98), rgb(141, 160, 203), rgb(231, 138, 195),
	rgb(166, 216, 84), rgb(255, 217, 47), rgb(229, 196, 148), rgb(179, 179, 179),
}

// Set3 — ColorBrewer qualitative, 12-color.
var set3Data = []gg.RGBA{
	rgb(141, 211, 199), rgb(255, 255, 179), rgb(190, 186, 218), rgb(251, 128, 114),
	rgb(128, 177, 211), rgb(253, 180, 98), rgb(179, 222, 105), rgb(252, 205, 229),
	rgb(217, 217, 217), rgb(188, 128, 189), rgb(204, 235, 197), rgb(255, 237, 111),
}

// Paired — ColorBrewer qualitative, 12-color paired light/dark.
var pairedData = []gg.RGBA{
	rgb(166, 206, 227), rgb(31, 120, 180),
	rgb(178, 223, 138), rgb(51, 160, 44),
	rgb(251, 154, 153), rgb(227, 26, 28),
	rgb(253, 191, 111), rgb(255, 127, 0),
	rgb(202, 178, 214), rgb(106, 61, 154),
	rgb(255, 255, 153), rgb(177, 89, 40),
}

// Pastel1 — ColorBrewer pastel variant of Set1.
var pastel1Data = []gg.RGBA{
	rgb(251, 180, 174), rgb(179, 205, 227), rgb(204, 235, 197),
	rgb(222, 203, 228), rgb(254, 217, 166), rgb(255, 255, 204),
	rgb(229, 216, 189), rgb(253, 218, 236), rgb(242, 242, 242),
}

// Pastel2 — ColorBrewer pastel variant of Set2.
var pastel2Data = []gg.RGBA{
	rgb(179, 226, 205), rgb(253, 205, 172), rgb(203, 213, 232), rgb(244, 202, 228),
	rgb(230, 245, 201), rgb(255, 242, 174), rgb(241, 226, 204), rgb(204, 204, 204),
}

// Accent — ColorBrewer qualitative, 8-color.
var accentData = []gg.RGBA{
	rgb(127, 201, 127), rgb(190, 174, 212), rgb(253, 192, 134), rgb(255, 255, 153),
	rgb(56, 108, 176), rgb(240, 2, 127), rgb(191, 91, 23), rgb(102, 102, 102),
}

// Dark2 — ColorBrewer qualitative, 8-color, dark/muted.
var dark2Data = []gg.RGBA{
	rgb(27, 158, 119), rgb(217, 95, 2), rgb(117, 112, 179), rgb(231, 41, 138),
	rgb(102, 166, 30), rgb(230, 171, 2), rgb(166, 118, 29), rgb(102, 102, 102),
}

// OkabeIto — colorblind-safe 8-color palette by Okabe & Ito (2008).
var okabeItoData = []gg.RGBA{
	rgb(0, 0, 0),       // black
	rgb(230, 159, 0),   // orange
	rgb(86, 180, 233),  // sky blue
	rgb(0, 158, 115),   // bluish green
	rgb(240, 228, 66),  // yellow
	rgb(0, 114, 178),   // blue
	rgb(213, 94, 0),    // vermillion
	rgb(204, 121, 167), // reddish purple
}

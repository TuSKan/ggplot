package fonts

// Weight represents the stroke thickness of a physical font face.
type Weight int

const (
	WeightThin       Weight = 100
	WeightExtraLight Weight = 200
	WeightLight      Weight = 300
	WeightNormal     Weight = 400
	WeightMedium     Weight = 500
	WeightSemiBold   Weight = 600
	WeightBold       Weight = 700
	WeightExtraBold  Weight = 800
	WeightBlack      Weight = 900
)

// Style represents the slant or italicization.
type Style int

const (
	StyleNormal Style = iota
	StyleItalic
	StyleOblique
)

// Font represents a resolvable physical font file face location.
type Font struct {
	Path        string
	Index       int
	Family      string
	Subfamily   string
	FullName    string
	Weight      Weight
	Style       Style
	Stretch     string // "condensed", "expanded", "normal"
	IsMonospace bool
	IsSymbol    bool
}

// Query defines the desired typographic traits requested.
type Query struct {
	Family          string
	Weight          Weight
	Style           Style
	PreferMonospace bool
	AllowFallback   bool
}

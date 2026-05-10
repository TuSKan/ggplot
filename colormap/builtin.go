package colormap

// Built-in colormap variables, exported for direct use in user code without
// going through the registry. Every variable below is also registered in
// init() so Resolve(name) returns the same value.

var (
	// Viridis is a perceptually-uniform sequential colormap (matplotlib default).
	Viridis Cmap
	// Plasma is a perceptually-uniform sequential colormap.
	Plasma Cmap
	// Inferno is a perceptually-uniform sequential colormap.
	Inferno Cmap
	// Magma is a perceptually-uniform sequential colormap.
	Magma Cmap
	// Cividis is a perceptually-uniform sequential colormap (colorblind-safe).
	Cividis Cmap

	// Greys is a sequential single-hue colormap (ColorBrewer 9-class).
	Greys Cmap
	// Blues is a sequential single-hue colormap (ColorBrewer 9-class).
	Blues Cmap
	// Greens is a sequential single-hue colormap (ColorBrewer 9-class).
	Greens Cmap
	// Oranges is a sequential single-hue colormap (ColorBrewer 9-class).
	Oranges Cmap
	// Reds is a sequential single-hue colormap (ColorBrewer 9-class).
	Reds Cmap
	// Purples is a sequential single-hue colormap (ColorBrewer 9-class).
	Purples Cmap
	// YlGn is a sequential multi-hue colormap (ColorBrewer 9-class).
	YlGn Cmap
	// YlGnBu is a sequential multi-hue colormap (ColorBrewer 9-class).
	YlGnBu Cmap
	// YlOrBr is a sequential multi-hue colormap (ColorBrewer 9-class).
	YlOrBr Cmap
	// YlOrRd is a sequential multi-hue colormap (ColorBrewer 9-class).
	YlOrRd Cmap

	// RdBu is a diverging colormap from red through white to blue.
	RdBu Cmap
	// RdYlBu is a diverging colormap from red through yellow to blue.
	RdYlBu Cmap
	// RdYlGn is a diverging colormap from red through yellow to green.
	RdYlGn Cmap
	// Spectral is a diverging rainbow-like colormap.
	Spectral Cmap
	// BrBG is a diverging colormap from brown through white to blue-green.
	BrBG Cmap
	// PiYG is a diverging colormap from pink through white to yellow-green.
	PiYG Cmap
	// PRGn is a diverging colormap from purple through white to green.
	PRGn Cmap
	// PuOr is a diverging colormap from purple through white to orange.
	PuOr Cmap
	// RdGy is a diverging colormap from red through white to grey.
	RdGy Cmap
	// Coolwarm is a diverging colormap from cool blue to warm red.
	Coolwarm Cmap
	// Bwr is a diverging colormap from blue through white to red.
	Bwr Cmap

	// Tab10 is a 10-color qualitative palette from Tableau.
	Tab10 Cmap
	// Tab20 is a 20-color qualitative palette from Tableau.
	Tab20 Cmap
	// Tab20b is a 20-color qualitative palette (variant b) from Tableau.
	Tab20b Cmap
	// Tab20c is a 20-color qualitative palette (variant c) from Tableau.
	Tab20c Cmap
	// Set1 is a qualitative palette from ColorBrewer.
	Set1 Cmap
	// Set2 is a qualitative palette from ColorBrewer.
	Set2 Cmap
	// Set3 is a qualitative palette from ColorBrewer.
	Set3 Cmap
	// Paired is a qualitative paired-color palette from ColorBrewer.
	Paired Cmap
	// Pastel1 is a qualitative pastel palette from ColorBrewer.
	Pastel1 Cmap
	// Pastel2 is a qualitative pastel palette from ColorBrewer.
	Pastel2 Cmap
	// Accent is a qualitative accent palette from ColorBrewer.
	Accent Cmap
	// Dark2 is a qualitative dark palette from ColorBrewer.
	Dark2 Cmap
	// OkabeIto is a colorblind-safe qualitative palette.
	OkabeIto Cmap
)

func init() {
	// Build LUTs from stop reference data.
	viridisLUT = lutFromStops(viridisStops)
	plasmaLUT = lutFromStops(plasmaStops)
	infernoLUT = lutFromStops(infernoStops)
	magmaLUT = lutFromStops(magmaStops)
	cividisLUT = lutFromStops(cividisStops)

	greysLUT = lutFromStops(greysStops)
	bluesLUT = lutFromStops(bluesStops)
	greensLUT = lutFromStops(greensStops)
	orangesLUT = lutFromStops(orangesStops)
	redsLUT = lutFromStops(redsStops)
	purplesLUT = lutFromStops(purplesStops)
	ylgnLUT = lutFromStops(ylgnStops)
	ylgnbuLUT = lutFromStops(ylgnbuStops)
	ylorbrLUT = lutFromStops(ylorbrStops)
	ylorrdLUT = lutFromStops(ylorrdStops)

	rdbuLUT = lutFromStops(rdbuStops)
	rdylbuLUT = lutFromStops(rdylbuStops)
	rdylgnLUT = lutFromStops(rdylgnStops)
	spectralLUT = lutFromStops(spectralStops)
	brbgLUT = lutFromStops(brbgStops)
	piygLUT = lutFromStops(piygStops)
	prgnLUT = lutFromStops(prgnStops)
	puorLUT = lutFromStops(puorStops)
	rdgyLUT = lutFromStops(rdgyStops)
	coolwarmLUT = lutFromStops(coolwarmStops)
	bwrLUT = lutFromStops(bwrStops)

	// Construct exported Cmap values.
	Viridis = NewLinearSegmented("viridis", PerceptuallyUniform, viridisLUT)
	Plasma = NewLinearSegmented("plasma", PerceptuallyUniform, plasmaLUT)
	Inferno = NewLinearSegmented("inferno", PerceptuallyUniform, infernoLUT)
	Magma = NewLinearSegmented("magma", PerceptuallyUniform, magmaLUT)
	Cividis = NewLinearSegmented("cividis", PerceptuallyUniform, cividisLUT)

	Greys = NewLinearSegmented("greys", Sequential, greysLUT)
	Blues = NewLinearSegmented("blues", Sequential, bluesLUT)
	Greens = NewLinearSegmented("greens", Sequential, greensLUT)
	Oranges = NewLinearSegmented("oranges", Sequential, orangesLUT)
	Reds = NewLinearSegmented("reds", Sequential, redsLUT)
	Purples = NewLinearSegmented("purples", Sequential, purplesLUT)
	YlGn = NewLinearSegmented("ylgn", Sequential, ylgnLUT)
	YlGnBu = NewLinearSegmented("ylgnbu", Sequential, ylgnbuLUT)
	YlOrBr = NewLinearSegmented("ylorbr", Sequential, ylorbrLUT)
	YlOrRd = NewLinearSegmented("ylorrd", Sequential, ylorrdLUT)

	RdBu = NewLinearSegmented("rdbu", Diverging, rdbuLUT)
	RdYlBu = NewLinearSegmented("rdylbu", Diverging, rdylbuLUT)
	RdYlGn = NewLinearSegmented("rdylgn", Diverging, rdylgnLUT)
	Spectral = NewLinearSegmented("spectral", Diverging, spectralLUT)
	BrBG = NewLinearSegmented("brbg", Diverging, brbgLUT)
	PiYG = NewLinearSegmented("piyg", Diverging, piygLUT)
	PRGn = NewLinearSegmented("prgn", Diverging, prgnLUT)
	PuOr = NewLinearSegmented("puor", Diverging, puorLUT)
	RdGy = NewLinearSegmented("rdgy", Diverging, rdgyLUT)
	Coolwarm = NewLinearSegmented("coolwarm", Diverging, coolwarmLUT)
	Bwr = NewLinearSegmented("bwr", Diverging, bwrLUT)

	Tab10 = NewListed("tab10", Qualitative, tab10Data)
	Tab20 = NewListed("tab20", Qualitative, tab20Data)
	Tab20b = NewListed("tab20b", Qualitative, tab20bData)
	Tab20c = NewListed("tab20c", Qualitative, tab20cData)
	Set1 = NewListed("set1", Qualitative, set1Data)
	Set2 = NewListed("set2", Qualitative, set2Data)
	Set3 = NewListed("set3", Qualitative, set3Data)
	Paired = NewListed("paired", Qualitative, pairedData)
	Pastel1 = NewListed("pastel1", Qualitative, pastel1Data)
	Pastel2 = NewListed("pastel2", Qualitative, pastel2Data)
	Accent = NewListed("accent", Qualitative, accentData)
	Dark2 = NewListed("dark2", Qualitative, dark2Data)
	OkabeIto = NewListed("okabe_ito", Qualitative, okabeItoData)

	// Register everything in the global registry.
	for _, c := range []Cmap{
		Viridis, Plasma, Inferno, Magma, Cividis,
		Greys, Blues, Greens, Oranges, Reds, Purples, YlGn, YlGnBu, YlOrBr, YlOrRd,
		RdBu, RdYlBu, RdYlGn, Spectral, BrBG, PiYG, PRGn, PuOr, RdGy, Coolwarm, Bwr,
		Tab10, Tab20, Tab20b, Tab20c, Set1, Set2, Set3, Paired, Pastel1, Pastel2,
		Accent, Dark2, OkabeIto,
	} {
		MustRegister(c)
	}
}

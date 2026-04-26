package colormap

// Built-in colormap variables, exported for direct use in user code without
// going through the registry. Every variable below is also registered in
// init() so Resolve(name) returns the same value.

var (
	// PerceptuallyUniform — matplotlib defaults.
	Viridis Cmap
	Plasma  Cmap
	Inferno Cmap
	Magma   Cmap
	Cividis Cmap

	// Sequential — ColorBrewer 9-class.
	Greys   Cmap
	Blues   Cmap
	Greens  Cmap
	Oranges Cmap
	Reds    Cmap
	Purples Cmap
	YlGn    Cmap
	YlGnBu  Cmap
	YlOrBr  Cmap
	YlOrRd  Cmap

	// Diverging — ColorBrewer 11-class + matplotlib originals.
	RdBu     Cmap
	RdYlBu   Cmap
	RdYlGn   Cmap
	Spectral Cmap
	BrBG     Cmap
	PiYG     Cmap
	PRGn     Cmap
	PuOr     Cmap
	RdGy     Cmap
	Coolwarm Cmap
	Bwr      Cmap

	// Qualitative.
	Tab10    Cmap
	Tab20    Cmap
	Tab20b   Cmap
	Tab20c   Cmap
	Set1     Cmap
	Set2     Cmap
	Set3     Cmap
	Paired   Cmap
	Pastel1  Cmap
	Pastel2  Cmap
	Accent   Cmap
	Dark2    Cmap
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

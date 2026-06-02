package main

import (
	"context"
	"log"
	"path/filepath"
	"runtime"

	"github.com/TuSKan/ggplot"
	"github.com/TuSKan/ggplot/aes"
	"github.com/TuSKan/ggplot/dataset"
	"github.com/TuSKan/ggplot/dataset/memory"
	"github.com/TuSKan/ggplot/geom"
	"github.com/TuSKan/ggplot/output/file"
	"github.com/TuSKan/ggplot/theme"
)

const numSegments = 100_000

func main() {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)

	aizawaExample(dir)
	dadrasExample(dir)
	lorenzExample(dir)
	rosslerExample(dir)
	halvorsenExample(dir)
	thomasExample(dir)
	chenExample(dir)
}

func plotAttractor(dir, filename, title string, seg SegmentData) {
	eng := memory.NewEngine(context.Background())

	// Use segment start-points as a dense point cloud, colored by time.
	ds, err := dataset.NewDataset(eng,
		eng.NewFloat64Column("x", seg.X0),
		eng.NewFloat64Column("y", seg.Y0),
		eng.NewFloat64Column("depth", seg.Depth),
	)
	if err != nil {
		log.Fatalln(err)
	}

	p := ggplot.New(ds,
		aes.X("x"),
		aes.Y("y"),
		aes.Color("depth"),
	).
		Layer(geom.Point(
			geom.WithSize(1.2),
			geom.WithAlpha(0.7),
		)).
		Labs(ggplot.Title(title)).
		LegendPosition("none").
		Theme("dark").
		ThemeOverride(
			theme.Override{Path: "panel.grid.major", Elem: theme.ElementBlank{}},
			theme.Override{Path: "panel.grid.minor", Elem: theme.ElementBlank{}},
			theme.Override{Path: "panel.border", Elem: theme.ElementBlank{}},
			theme.Override{Path: "axis.title", Elem: theme.ElementBlank{}},
			theme.Override{Path: "axis.text", Elem: theme.ElementBlank{}},
			theme.Override{Path: "axis.ticks", Elem: theme.ElementBlank{}},
			theme.Override{Path: "axis.line", Elem: theme.ElementBlank{}},
		)

	out := filepath.Join(dir, filename)
	if err := file.Save(context.Background(), p, out, 900, 900); err != nil { //nolint:mnd // Example output size.
		log.Fatalln(err)
	}

	log.Printf("Saved %s (%d points)", out, len(seg.X0))
}

func aizawaExample(dir string) {
	seg := AizawaBeautifulSegments(numSegments)
	plotAttractor(dir, "aizawa.png", "Aizawa Attractor", seg)
}

func dadrasExample(dir string) {
	seg := DadrasBeautifulSegments(numSegments)
	plotAttractor(dir, "dadras.png", "Dadras Attractor", seg)
}

func lorenzExample(dir string) {
	seg := LorenzBeautifulSegments(numSegments)
	plotAttractor(dir, "lorenz.png", "Lorenz Attractor", seg)
}

func rosslerExample(dir string) {
	seg := RosslerBeautifulSegments(numSegments)
	plotAttractor(dir, "rossler.png", "Rössler Attractor", seg)
}

func halvorsenExample(dir string) {
	seg := HalvorsenBeautifulSegments(numSegments)
	plotAttractor(dir, "halvorsen.png", "Halvorsen Attractor", seg)
}

func thomasExample(dir string) {
	seg := ThomasBeautifulSegments(numSegments)
	plotAttractor(dir, "thomas.png", "Thomas Attractor", seg)
}

func chenExample(dir string) {
	seg := ChenBeautifulSegments(numSegments)
	plotAttractor(dir, "chen.png", "Chen Attractor", seg)
}

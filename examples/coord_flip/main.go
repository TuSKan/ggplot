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
)

func main() {
	ctx := context.Background()
	memEngine := memory.NewEngine(ctx)
	_, output, _, _ := runtime.Caller(0)

	// Clean categorical dataset for Bar/Area/Step
	ds, _ := dataset.NewDataset(memEngine,
		memEngine.NewStringColumn("category", []string{"A", "B", "C", "D", "E"}),
		memEngine.NewFloat64Column("value", []float64{15.5, 22.1, 18.0, 31.4, 25.2}),
	)

	// 1. Horizontal Bar Chart (geom_col)
	log.Println("Generating flip_col.png...")
	pCol := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.Col(geom.WithFill("#3b82f6"), geom.WithAlpha(0.8))).
		CoordFlip()
	pCol.Save(ctx, filepath.Join(filepath.Dir(output), "flip_col.png"), 600, 400)

	// 2. Horizontal Area Chart
	log.Println("Generating flip_area.png...")
	pArea := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.Area(geom.WithFill("#10b981"), geom.WithAlpha(0.6))).
		CoordFlip()
	pArea.Save(ctx, filepath.Join(filepath.Dir(output), "flip_area.png"), 600, 400)

	// 3. Horizontal Step Chart
	log.Println("Generating flip_step.png...")
	pStep := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.Step(geom.WithColor("#8b5cf6"), geom.WithLineWidth(2))).
		CoordFlip()
	pStep.Save(ctx, filepath.Join(filepath.Dir(output), "flip_step.png"), 600, 400)

	// 4. Reference Lines (HLine / VLine transpose test)
	log.Println("Generating flip_lines.png...")
	pLines := ggplot.New(ds, aes.X("category"), aes.Y("value")).
		Layer(geom.Point(geom.WithSize(4), geom.WithColor("#ef4444"))).
		// Original Y-intercept=20 -> becomes visual X=20 (Vertical line)
		Layer(geom.HLine(geom.WithIntercept(20), geom.WithColor("#f59e0b"), geom.WithLineWidth(2))).
		CoordFlip()
	pLines.Save(ctx, filepath.Join(filepath.Dir(output), "flip_lines.png"), 600, 400)

	// 5. Horizontal Boxplot
	log.Println("Generating flip_boxplot.png...")
	var categories []string
	var values []float64
	for i := 0; i < 50; i++ {
		categories = append(categories, "Group 1", "Group 2")
		values = append(values, float64(i%15)+5, float64(i%12)+10)
	}
	dsBox, _ := dataset.NewDataset(memEngine,
		memEngine.NewStringColumn("group", categories),
		memEngine.NewFloat64Column("score", values),
	)

	pBox := ggplot.New(dsBox, aes.X("group"), aes.Y("score")).
		Layer(geom.Boxplot(geom.WithFill("#6366f1"), geom.WithAlpha(0.7))).
		CoordFlip()
	pBox.Save(ctx, filepath.Join(filepath.Dir(output), "flip_boxplot.png"), 600, 400)

	log.Println("All examples generated successfully.")
}

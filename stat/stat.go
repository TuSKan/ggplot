// Package stat provides composable statistical transforms for the Grammar of
// Graphics. Each transform implements the [Transform] interface and can be
// chained arbitrarily in a pipeline:
//
//	stat.NormalizeY(stat.BinX(stat.WithBins(40)))
//
// All transforms delegate heavy computation to the engine's [dataset.StatKernel]
// interface — zero materialization occurs within this package.
//
// Transform factories:
//   - [BinX] — histogram binning (via StatKernel.Histogram)
//   - [Count] — frequency counting (via GroupBy + AggCount)
//   - [DensityX] — kernel density estimation (via StatKernel.KDE)
//   - [SmoothXY] — linear / LOESS regression (via StatKernel.LinearFit / LoessFit)
//   - [SummaryXY] — grouped mean (via GroupBy + AggMean)
//   - [BoxplotY] — five-number summary (via StatKernel.Boxplot)
//   - [IdentityTransform] — pass-through
//   - [NormalizeY] / [NormalizeX] — rescale channel to sum to a total
//   - [FilterX] / [FilterY] — row-level predicate filter
//   - [SortBy] — sort rows by column
//   - [ReverseRows] — reverse row order
//   - [TopN] — keep top/bottom N rows by column
//   - [SelectRow] — keep a single row by mode (first, last, min, max)
//   - [StackY] / [StackX] — cumulative stacking within a group
//   - [GroupX] / [GroupY] — group-by with reducer (sum, mean, median,
//     min, max, count, variance, deviation, first, last, mode)
package stat

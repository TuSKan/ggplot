/*
Package dataset provides zero-copy, Arrow-backed dataframe capabilities
optimized for the Grammar of Graphics pipeline.

# Materialization Policy

The dataset package enforces zero-copy boundaries where possible
to adhere to performance constraints. Materialization (i.e. copying data
into `[]float64` or contiguous buffers) is governed by strict rules.

When materialization is ALLOWED:
1. Exporting directly to a rendering context (e.g., rasterizing or WebGL buffers) that cannot interpret Arrow chunked loops.
2. Inside `internal/stat` for complex smoothing (like KDE or loess) where mathematical algorithms demand contiguous arrays.
3. Resolving constraint-based equations during layout planning.

When materialization is FORBIDDEN:
1. Filtering, selecting, mutating, or grouping columns. These MUST output lazy "Derived Dataset" nodes that delay physical layout until requested.
2. Slicing rows or extracting pagination chunks. These MUST use Arrow's native zero-copy subset logic.
3. Training boundaries (Min/Max). Domain extraction MUST loop over the physical Arrow memory chunks directly, ignoring Nulls without materializing the column.
*/
package dataset

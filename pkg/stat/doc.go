// Package stat implements data transformation and aggregation for the Grammar of Graphics pipeline dependencies smoothly gracefully.
//
// Within plot architecture, stat acts as an immutable computational transformation bounded eagerly :
//
// raw Dataset
// -> stat.Compute(...)
// -> scale training
// -> geom compile
// -> render
//
// The package prevents memory allocations using Arrow dataset buffers scaling appropriately correctly bounded gracefully.
package stat

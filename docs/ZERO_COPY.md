# Mechanics of Zero-Copy execution

`ggplot` leverages pure interfaces enabling highly complex data-sciences pipelines cleanly avoiding natively creating physical array memory boundaries locally structurally smoothly:

## The `LazyDataset` Model
Every functional operation like `dataset.Filter` or `dataset.Mutate` natively generates a generic wrapper structure functionally forwarding boundaries to active column configurations cleanly. 

If operating locally utilizing `internal/adapter/arrow`, Apache Arrow explicitly drives zero-copy boundaries. E.g., when mapped utilizing `Filter()`, an Arrow `Dictionary` mask dynamically computes physical mappings logically pointing arrays purely behind the `Dataset.Column()` interfaces securely. 

## The FlightSQL Pushdown Layout
If pointing aggressively to remote tables (`internal/adapter/sql`), evaluations completely ignore native array processing limits, intelligently pushing operations into `.FilterSQL("col > 50")`.

Only structurally un-mappable AST configurations physically hit boundaries mapped through `Materialize` falling back evaluating into explicitly built `TableDataset` structs dynamically securely safely executing!

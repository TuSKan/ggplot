# Mechanics of Zero-Copy execution

`ggplot` leverages pure interfaces enabling highly complex data-sciences pipelines avoiding creating physical array memory boundaries locally :

## The `LazyDataset` Model
Every functional operation like `dataset.Filter` or `dataset.Mutate` generates a generic wrapper structure forwarding boundaries to active column configurations. 

If operating locally utilizing `internal/adapter/arrow`, Apache Arrow drives zero-copy boundaries. E.g., when utilizing `Filter()`, an Arrow `Dictionary` mask computes physical pointing arrays behind the `Dataset.Column()` interfaces. 

## The FlightSQL Pushdown Layout
If pointing to remote tables (`internal/adapter/sql`), evaluations ignore native array processing limits, pushing operations into `.FilterSQL("col > 50")`.

Only un-mappable AST configurations hit boundaries through `Materialize` falling back evaluating into built `TableDataset` structs!

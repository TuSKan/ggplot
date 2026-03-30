# Conceptual Architecture

`ggplot` is built prioritizing safety, declarative AST construction, and zero-copy performance. Instead of coupling data manipulation tightly into pixel plotting, the library parses declarative chains into strict sequential evaluations.

## 1. Dataset Abstraction (`internal/dataset`)
Isolates pure memory blocks. All operations (e.g. `Filter`, `Slice`) wrap metadata passing arrays via generic `NativeFilterProvider` logics instead of looping values. Apache Arrow and FlightSQL sit behind this isolation.

## 2. AST Grammar & Validation (`internal/ast` & `pkg/plot`)
The `pkg/plot` boundary builds defined explicit `Layers`, each declaring what `Geom` connects to which `Dataset.Column()` to visual parameters (`aes.X`, `aes.Color`). Calling `p.Compile()` verifies whether satisfy required validations evaluating a full `RenderPlan`.

## 3. Scale Training (`internal/scale`)
Once validated, generate `Scale` targets defining boundaries from `<500, 1000>` down to structural `<0.0, 1.0>` outputs avoiding boundary bleeding handling `NaN` / Missing Values.

## 4. Layout Guillotine (`internal/layout`)
Utilizes a decoupled `<Rect>` layout engine splicing blocks from exterior borders calculating Title/Legend geometries determining safe facet matrices.

## 5. Render Backend (`internal/render`)
Execution bridges logical rendering shapes decoupling operations. Native `cpu` utilize the `gogpu/gg` wrapper ensuring anti-aliasing.

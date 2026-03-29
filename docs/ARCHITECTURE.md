# Conceptual Architecture

`ggplot` is built natively prioritizing safety, declarative AST construction, and zero-copy performance. Instead of coupling data manipulation tightly into pixel plotting, the library parses declarative chains into strict sequential evaluations.

## 1. Dataset Abstraction (`internal/dataset`)
Isolates pure memory blocks cleanly. All operations (e.g. `Filter`, `Slice`) natively wrap metadata passing arrays transparently via generic `NativeFilterProvider` logics instead of natively looping values. Apache Arrow and FlightSQL bounds sit purely behind this isolation explicitly.

## 2. AST Grammar & Validation (`internal/ast` & `pkg/plot`)
The `pkg/plot` boundary builds exactly defined explicit `Layers`, each declaring exactly what `Geom` connects functionally to which `Dataset.Column()` mapping exactly to visual parameters (`aes.X`, `aes.Color`). Calling `p.Compile()` verifies explicitly whether mappings satisfy required validations cleanly evaluating a full `RenderPlan`.

## 3. Scale Training (`internal/scale`)
Once validated, mappings natively generate `Scale` targets defining boundaries cleanly mapping bounds from `<500, 1000>` down cleanly to structural `<0.0, 1.0>` outputs avoiding boundary bleeding explicitly handling `NaN` / Missing Values securely.

## 4. Layout Guillotine (`internal/layout`)
Utilizes a decoupled `<Rect>` layout engine natively splicing blocks from exterior borders calculating Title/Legend geometries naturally determining exactly safe facet matrices cleanly.

## 5. Render Backend (`internal/render`)
Execution bridges logical mappings physically rendering shapes cleanly decoupling operations cleanly. Native `cpu` bounds utilize the `gogpu/gg` wrapper ensuring anti-aliasing bounds correctly.

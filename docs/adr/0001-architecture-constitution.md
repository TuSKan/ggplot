# ADR 0001: Architecture Constitution

## Status
Approved

## Context
A production-grade, pure-Go Grammar of Graphics library is being built. It needs to remain highly modular, with zero-copy data reads using Apache Arrow natively alongside Go slices.

## Decision
The architecture uses a strict pipeline strictly enforcing the Grammar of Graphics principles.
- `pkg/plot`: Pure declarative AST public API.
- `internal/dataset`: Abstract columnar data.
- `internal/expr`: Lazy eval tree.
- `internal/scale`: Domains and Ranges mapping.
- `internal/layout`: Constraint-based bounds calculation.
- `internal/geom`: Geometry compilers.
- `internal/render`: Stateless output plugins (SVG, Raster).

Optional backends (Arrow FlightSQL, GPU) are shielded by standard `//go:build` tags inside `internal/adapter/flightsql` and `internal/render/gpu`.

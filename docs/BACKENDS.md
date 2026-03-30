# Backend Interoperability Guide

`ggplot` exposes native components guaranteeing pure separation. Operations fallback avoiding dependency locking.

## 1. Zero-Hardware CPU Fallback
By default, the `internal/render/cpu` bound generates pixels utilizing the robust lightweight `github.com/gogpu/gg` float engine rendering anti-aliased geometry without hardware GPU dependencies satisfying standard container testing.

## 2. Optional GPU Connectivity
Utilizing Go build tags `//go:build gpu` enables physical compilation boundaries directly onto active desktop graphic windows directly utilizing the zero-copy engine generating graphics inside declared contexts rendering.

## 3. Remote FlightSQL Adapters
`internal/adapter/sql` leverages `arrow-adbc` enabling optional cloud-bound interactions generating zero-copy executions pushing evaluations `FilterSQL` out of the local CPU context directly!

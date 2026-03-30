package ast

import "github.com/TuSKan/ggplot/internal/expr"

// Aes binds aesthetic names statically to typed expressions.
type Aes map[string]expr.Expr

// Merge combines another mapping into this one, letting the receiver override.
func (a Aes) Merge(other Aes) Aes {
	res := make(Aes)
	for k, v := range other {
		res[k] = v
	}
	for k, v := range a {
		res[k] = v
	}
	return res
}

// Geom definitions describe rendering behavior constraints.
type Geom interface {
	Name() string
	RequiredAesthetics() []string
	DefaultStat() string
}

// Stat definitions specify statistical transformations bounds.
type Stat interface {
	Name() string
}

// Configuration interfaces ensuring type safety!
type ThemeConfig interface{ IsTheme() }
type FacetConfig interface{ IsFacet() }
type ScaleConfig interface{ IsScale() }

// Layer specifies an immutable single graphical trace.
type Layer struct {
	Geom    Geom
	Stat    Stat
	Mapping Aes
}

// Plot defines the generic declarative grammar tree top-level structure.
type Plot struct {
	Dataset any // Expected to be dataset.Dataset
	Mapping Aes
	Layers  []Layer
	Theme   ThemeConfig
	Facet   FacetConfig
	Scales  []ScaleConfig
}

// RenderPlan is the concrete output post-validation.
type RenderPlan struct {
	Dataset any
	Layers  []CompiledLayer
	Scales  map[string]any
	Theme   ThemeConfig
	Facet   FacetConfig
}

// CompiledLayer represents a fully validated layout ready for rendering limits bounds.
type CompiledLayer struct {
	Geom    Geom
	Stat    Stat
	Mapping Aes
}

package stat

import (
	"context"
	"fmt"

	"github.com/TuSKan/ggplot/dataset"
)

// Transform is the composable data-transform contract. Transforms
// chain because their input and output shapes are the same:
// data + mapping in, data + mapping out. The bin transform, the
// smooth transform, the normalize transform, the identity transform —
// all the same shape.
//
// Apply MUST NOT mutate in; it returns a new TransformResult.
type Transform interface {
	// Name returns a stable identifier for debugging, golden tests,
	// and pipeline introspection.
	Name() string

	// Apply runs the transform. Implementations MUST NOT mutate in.
	Apply(ctx context.Context, in TransformInput) (TransformResult, error)

	// OutputMapping describes how aesthetic channels are rewritten.
	// nil means the transform preserves the mapping (identity, filter, sort).
	// A non-nil map rewrites channels: {"y": "count"} means the y channel
	// now points at the "count" column the transform produced.
	OutputMapping() map[string]string

	// OutputSchema names the columns this transform produces.
	OutputSchema() []string

	// OutputHints declares semantic hints for output channels.
	// Axis/legend formatters use these: HintProportion → "%" tick formatting,
	// HintCount → integer ticks, HintInterval → bin-edge rendering.
	// nil for transforms that don't change channel semantics.
	OutputHints() map[string]ChannelHint
}

// TransformInput carries data and aesthetic mapping into a transform.
type TransformInput struct {
	Data    dataset.Dataset
	Mapping map[string]string
}

// TransformResult carries transformed data and rewritten mapping out.
type TransformResult struct {
	Data    dataset.Dataset
	Mapping map[string]string
}

// ChannelHint declares the semantic meaning of a channel's output.
// Open string type: known hints get special formatting; unknown hints
// get default formatting. Third-party transforms can declare arbitrary hints.
type ChannelHint string

// Standard channel hints.
const (
	HintNone        ChannelHint = ""
	HintCount       ChannelHint = "count"
	HintProportion  ChannelHint = "proportion"
	HintProbability ChannelHint = "probability" // axis clamps to [0,1]
	HintInterval    ChannelHint = "interval"
	HintCumulative  ChannelHint = "cumulative"
	HintDeviation   ChannelHint = "deviation"
)

// --- Identity Transform ---

// IdentityTransform passes data through unchanged.
type identityTransform struct{}

// IdentityTransform returns a Transform that passes data through unchanged.
func IdentityTransform() Transform { return identityTransform{} }

func (identityTransform) Name() string                        { return "identity" }
func (identityTransform) OutputMapping() map[string]string    { return nil }
func (identityTransform) OutputSchema() []string              { return nil }
func (identityTransform) OutputHints() map[string]ChannelHint { return nil }

func (identityTransform) Apply(_ context.Context, in TransformInput) (TransformResult, error) {
	return TransformResult(in), nil
}

// RunPipeline executes an ordered chain of transforms. If the pipeline
// is nil or empty, data passes through unchanged (identity).
//
// Between stages, if a transform produces a lazy (uncollected) Dataset,
// RunPipeline materializes it before passing to the next transform.
// This ensures each transform receives a Dataset with a valid Table(),
// so engine interfaces (Selector, Aggregator, etc.) work correctly.
func RunPipeline(ctx context.Context, pipeline []Transform, data dataset.Dataset, mapping map[string]string) (dataset.Dataset, map[string]string, error) {
	in := TransformInput{Data: data, Mapping: mapping}

	for i, tf := range pipeline {
		out, err := tf.Apply(ctx, in)
		if err != nil {
			return dataset.Dataset{}, nil, fmt.Errorf("transform %q: %w", tf.Name(), err)
		}

		// Materialize between stages if the output is lazy and there are
		// more transforms to run. The last stage stays lazy — the caller
		// (ggplot.buildPanel) handles final Collect.
		if out.Data.Table() == nil && i < len(pipeline)-1 {
			out.Data, err = out.Data.Collect(ctx)
			if err != nil {
				return dataset.Dataset{}, nil, fmt.Errorf("transform %q: collect: %w", tf.Name(), err)
			}
		}

		in = TransformInput(out)
	}

	return in.Data, in.Mapping, nil
}

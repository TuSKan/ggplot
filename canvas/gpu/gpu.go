//go:build !js

// Package gpu opts in GPU acceleration.
package gpu

import (
	// opt-in GPU acceleration
	_ "github.com/gogpu/gg/gpu"
)

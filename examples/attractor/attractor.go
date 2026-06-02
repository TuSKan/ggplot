// Package main provides a reusable 3D attractor segment engine and beautiful presets.
//
// The engine integrates ODE systems using 4th-order Runge-Kutta (RK4),
// projects 3D trajectories through a configurable camera, and outputs
// consecutive line segments suitable for dense alpha-blended rendering.
package main

import "math"

// Vec3 is a 3D point/vector.
type Vec3 struct {
	X float64
	Y float64
	Z float64
}

// Flow3D computes the derivative dv/dt at a given state v.
type Flow3D func(v Vec3) Vec3

// Camera defines the 3D→2D projection via Euler angles (ZXZ convention).
type Camera struct {
	Yaw   float64
	Pitch float64
	Roll  float64

	Scale  float64
	XShift float64
	YShift float64
}

// SegmentParams configures the attractor integration and projection.
type SegmentParams struct {
	Steps  int
	BurnIn int
	Dt     float64
	Start  Vec3
	Flow   Flow3D
	Camera Camera

	// MaxJump filters out segments longer than this (0 means disabled).
	MaxJump float64

	// Stride subsamples the trajectory (1 means every step).
	Stride int
}

// SegmentData holds projected line segments for rendering.
type SegmentData struct {
	X0 []float64
	Y0 []float64
	X1 []float64
	Y1 []float64

	// T is normalized time [0, 1], useful for color gradients.
	T []float64

	// Depth is the projected Z coordinate, useful for alternative coloring.
	Depth []float64
}

// AttractorSegments integrates a 3D ODE system and projects segments.
func AttractorSegments(p SegmentParams) SegmentData {
	if p.Steps <= 1 || p.Flow == nil {
		return SegmentData{}
	}

	if p.BurnIn < 0 {
		p.BurnIn = 0
	}

	if p.Dt == 0 {
		p.Dt = 0.005
	}

	if p.Stride <= 0 {
		p.Stride = 1
	}

	if p.Camera.Scale == 0 {
		p.Camera.Scale = 1
	}

	capacity := p.Steps / p.Stride

	out := SegmentData{
		X0:    make([]float64, 0, capacity),
		Y0:    make([]float64, 0, capacity),
		X1:    make([]float64, 0, capacity),
		Y1:    make([]float64, 0, capacity),
		T:     make([]float64, 0, capacity),
		Depth: make([]float64, 0, capacity),
	}

	v := p.Start

	for range p.BurnIn {
		v = rk4(v, p.Dt, p.Flow)
		if !finite(v) {
			return out
		}
	}

	var (
		prevX, prevY, prevDepth float64
		hasPrev                 = false
	)

	for i := range p.Steps {
		v = rk4(v, p.Dt, p.Flow)
		if !finite(v) {
			break
		}

		if i%p.Stride != 0 {
			continue
		}

		x, y, depth := project(v, p.Camera)

		if !hasPrev {
			prevX = x
			prevY = y
			prevDepth = depth
			hasPrev = true

			continue
		}

		dx := x - prevX
		dy := y - prevY

		if p.MaxJump <= 0 || dx*dx+dy*dy <= p.MaxJump*p.MaxJump {
			out.X0 = append(out.X0, prevX)
			out.Y0 = append(out.Y0, prevY)
			out.X1 = append(out.X1, x)
			out.Y1 = append(out.Y1, y)

			out.T = append(out.T, float64(i)/float64(p.Steps-1))
			out.Depth = append(out.Depth, 0.5*(prevDepth+depth)) //nolint:mnd // Midpoint average.
		}

		prevX = x
		prevY = y
		prevDepth = depth
	}

	return out
}

func rk4(v Vec3, dt float64, f Flow3D) Vec3 {
	k1 := f(v)
	k2 := f(add(v, scale(k1, 0.5*dt))) //nolint:mnd // RK4 midpoint weight.
	k3 := f(add(v, scale(k2, 0.5*dt))) //nolint:mnd // RK4 midpoint weight.
	k4 := f(add(v, scale(k3, dt)))

	return Vec3{
		X: v.X + dt*(k1.X+2*k2.X+2*k3.X+k4.X)/6, //nolint:mnd // RK4 weights.
		Y: v.Y + dt*(k1.Y+2*k2.Y+2*k3.Y+k4.Y)/6, //nolint:mnd // RK4 weights.
		Z: v.Z + dt*(k1.Z+2*k2.Z+2*k3.Z+k4.Z)/6, //nolint:mnd // RK4 weights.
	}
}

func add(a, b Vec3) Vec3 {
	return Vec3{
		X: a.X + b.X,
		Y: a.Y + b.Y,
		Z: a.Z + b.Z,
	}
}

func scale(v Vec3, s float64) Vec3 {
	return Vec3{
		X: v.X * s,
		Y: v.Y * s,
		Z: v.Z * s,
	}
}

func finite(v Vec3) bool {
	return !math.IsNaN(v.X) && !math.IsNaN(v.Y) && !math.IsNaN(v.Z) &&
		!math.IsInf(v.X, 0) && !math.IsInf(v.Y, 0) && !math.IsInf(v.Z, 0)
}

func project(v Vec3, c Camera) (x float64, y float64, depth float64) {
	cy := math.Cos(c.Yaw)
	sy := math.Sin(c.Yaw)

	cp := math.Cos(c.Pitch)
	sp := math.Sin(c.Pitch)

	cr := math.Cos(c.Roll)
	sr := math.Sin(c.Roll)

	// Yaw around Z.
	x1 := cy*v.X - sy*v.Y
	y1 := sy*v.X + cy*v.Y
	z1 := v.Z

	// Pitch around X.
	x2 := x1
	y2 := cp*y1 - sp*z1
	z2 := sp*y1 + cp*z1

	// Roll around Z.
	x3 := cr*x2 - sr*y2
	y3 := sr*x2 + cr*y2

	return c.Scale*x3 + c.XShift, c.Scale*y3 + c.YShift, z2
}

// ---------------------------------------------------------------------------
// Attractor flows
// ---------------------------------------------------------------------------

// Aizawa returns the Aizawa attractor flow.
func Aizawa(a, b, c, d, e, f float64) Flow3D {
	return func(v Vec3) Vec3 {
		r2 := v.X*v.X + v.Y*v.Y

		return Vec3{
			X: (v.Z-b)*v.X - d*v.Y,
			Y: d*v.X + (v.Z-b)*v.Y,
			Z: c + a*v.Z - (v.Z*v.Z*v.Z)/3 - r2*(1+e*v.Z) + f*v.Z*v.X*v.X*v.X, //nolint:mnd // ODE cubic term z³/3.
		}
	}
}

// AizawaBeautifulSegments generates an n-segment Aizawa attractor with tuned parameters.
func AizawaBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 20_000,
		Dt:     0.01,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Aizawa(0.95, 0.70, 0.60, 3.50, 0.25, 0.10),
		Camera: Camera{
			Yaw:   0.65,
			Pitch: 1.15,
			Roll:  -0.20,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Dadras returns the Dadras attractor flow.
func Dadras(a, b, c, d, e float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: v.Y - a*v.X + b*v.Y*v.Z,
			Y: c*v.Y - v.X*v.Z + v.Z,
			Z: d*v.X*v.Y - e*v.Z,
		}
	}
}

// DadrasBeautifulSegments generates an n-segment Dadras attractor with tuned parameters.
func DadrasBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 20_000,
		Dt:     0.005,
		Start:  Vec3{X: 0.1, Y: 0.1, Z: 0.1},
		Flow:   Dadras(3.0, 2.7, 1.7, 2.0, 9.0),
		Camera: Camera{
			Yaw:   -0.55,
			Pitch: 0.90,
			Roll:  0.15,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Lorenz returns the Lorenz system flow.
// Classic parameters: sigma=10, rho=28, beta=8/3.
func Lorenz(sigma, rho, beta float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: sigma * (v.Y - v.X),
			Y: v.X*(rho-v.Z) - v.Y,
			Z: v.X*v.Y - beta*v.Z,
		}
	}
}

// LorenzBeautifulSegments generates an n-segment Lorenz attractor with tuned parameters.
func LorenzBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.005,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Lorenz(10, 28, 8.0/3.0),
		Camera: Camera{
			Yaw:   0.20,
			Pitch: -1.30,
			Roll:  0,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Rossler returns the Rössler system flow.
// Classic parameters: a=0.2, b=0.2, c=5.7.
func Rossler(a, b, c float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: -v.Y - v.Z,
			Y: v.X + a*v.Y,
			Z: b + v.Z*(v.X-c),
		}
	}
}

// RosslerBeautifulSegments generates an n-segment Rössler attractor with tuned parameters.
func RosslerBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.01,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Rossler(0.2, 0.2, 5.7),
		Camera: Camera{
			Yaw:   0.30,
			Pitch: 0.80,
			Roll:  0,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Halvorsen returns the Halvorsen symmetric attractor flow.
// a=1.3 produces the chaotic tri-lobed attractor; a=1.89 converges to a periodic orbit.
func Halvorsen(a float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: -a*v.X - 4*v.Y - 4*v.Z - v.Y*v.Y,
			Y: -a*v.Y - 4*v.Z - 4*v.X - v.Z*v.Z,
			Z: -a*v.Z - 4*v.X - 4*v.Y - v.X*v.X,
		}
	}
}

// HalvorsenBeautifulSegments generates an n-segment Halvorsen attractor with tuned parameters.
func HalvorsenBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.005,
		Start:  Vec3{X: 1, Y: 0, Z: 0},
		Flow:   Halvorsen(1.3),
		Camera: Camera{
			Yaw:   0.60,
			Pitch: 0.80,
			Roll:  0.10,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Thomas returns the Thomas cyclically symmetric attractor flow.
// Classic parameter: b=0.208186.
func Thomas(b float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: math.Sin(v.Y) - b*v.X,
			Y: math.Sin(v.Z) - b*v.Y,
			Z: math.Sin(v.X) - b*v.Z,
		}
	}
}

// ThomasBeautifulSegments generates an n-segment Thomas attractor with tuned parameters.
func ThomasBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 10_000,
		Dt:     0.05,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Thomas(0.208186),
		Camera: Camera{
			Yaw:   0.40,
			Pitch: 0.70,
			Roll:  0,
			Scale: 1,
		},
		Stride: 1,
	})
}

// Chen returns the Chen system flow.
// Classic parameters: a=35, b=3, c=28.
func Chen(a, b, c float64) Flow3D {
	return func(v Vec3) Vec3 {
		return Vec3{
			X: a * (v.Y - v.X),
			Y: (c-a)*v.X - v.X*v.Z + c*v.Y,
			Z: v.X*v.Y - b*v.Z,
		}
	}
}

// ChenBeautifulSegments generates an n-segment Chen attractor with tuned parameters.
func ChenBeautifulSegments(n int) SegmentData {
	return AttractorSegments(SegmentParams{
		Steps:  n,
		BurnIn: 5_000,
		Dt:     0.0025,
		Start:  Vec3{X: 0.1, Y: 0, Z: 0},
		Flow:   Chen(35, 3, 28),
		Camera: Camera{
			Yaw:   0.20,
			Pitch: -1.10,
			Roll:  0.10,
			Scale: 1,
		},
		Stride: 1,
	})
}
